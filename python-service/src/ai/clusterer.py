import json
import logging
from typing import List, Dict

import numpy as np
import hdbscan
from sklearn.metrics.pairwise import cosine_similarity
import anthropic

logger = logging.getLogger(__name__)

LABEL_PROMPT = """\
  Given these software problem titles, generate a short label (3-7 words) \
  that describes the common theme.

  Titles:
  {titles}

  Respond with ONLY the label text, nothing else.
  """

class Clusterer:
    def __init__(self, api_key: str, db):
        self.client = anthropic.AsyncAnthropic(api_key=api_key)
        self.db = db

    async def cluster(self, min_cluster_size: int = 5) -> List[Dict]:
        from embeddings.store import EmbeddingStore
        emb_store = EmbeddingStore(self.db)
        all_embeddings = await emb_store.get_all_embeddings()

        if len(all_embeddings) < min_cluster_size:
            logger.info(f"Not enough embeddings ({len(all_embeddings)} for clustering)")
            return []

        problem_ids = [pid for pid, _ in all_embeddings]
        vectors = np.array([vec for _, vec in all_embeddings])

        clusterer = hdbscan.HDBSCAN(
            min_cluster_size = min_cluster_size,
            metric="euclidean",
            cluster_selection_method="eom",
        )
        labels = clusterer.fit_predict(vectors)

        clusters_map: Dict[int, List[int]] = {}
        for idx, label in enumerate(labels):
            if label == -1:
                continue
            clusters_map.setdefault(labels, []).append(idx)
        logger.info(f"Found {len(clusters_map)} clusters from {len(all_embeddings)} problems")

        results = []
        for cluster_label, indices in clusters_map.items():
            cluster_problem_ids = [problem_ids[i] for i in indices]
            cluster_vectors = vectors[indices]

            if len(cluster_vectors) > 1:
                sim_matrix = cosine_similarity(cluster_vectors)
                mask = np.triu(np.ones_like(sim_matrix, dtype=bool), k=1)
                cohesion = float(sim_matrix[mask].mean())
            else:
                cohesion = 1.0

            label_text = await self._generate_label(cluster_problem_ids)

            results.append({
                "label": label_text,
                "problem_ids": cluster_problem_ids,
                "cohesion_score": cohesion
            })
        return results

async def _generate_label(self, problem_ids: List[str]) -> str:
    try:
        placeholders = ", ".join(
            f"${i+1}::uuid" for i in range(min(len(problem_ids), 10))
        )
        query = f"""
                  SELECT title FROM raw_problems
                  WHERE id IN ({placeholders})
                  LIMIT 10
              """
        rows = await self.db.fetch(query, *problem_ids[:10])
        titles = "\n".join(f"- {row['title']}" for row in rows)

        response = await self.client.messages.create(
            model="claude-sonnet-4-20250514",
            max_tokens=50,
            messages=[
                {"role": "user", "content": LABEL_PROMPT.format(titles=titles)}
            ],
        )
        return response.content[0].text.strip()

    except Exception as e:
        logger.error(f"Label generation failed: {e}")
        return f"Cluster ({len(problem_ids)} problems)"
