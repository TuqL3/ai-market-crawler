import logging
from typing import Tuple, List
import numpy as np

logger = logging.getLogger(__name__)

class EmbeddingStore:
    def __init__(self, db):
        self.db = db
    async def save_embeddings(self, embeddings: List[Tuple[str, List[float]]]) -> int | None:
        if not embeddings:
            return 0
        queue = """
                INSERT INTO problem_embeddings (raw_problem_id, embedding)
                VALUES ($1::uuid, $2::vector)
                    ON CONFLICT (raw_problem_id) DO UPDATE
                                                        SET embedding = EXCLUDED.embedding,
                                                        created_at = NOW() \
                """
        count = 0
        for problem_id, vector in embeddings:
            try:
                vec_str = "[" + ",".join(str(v) for v in vector) + "]"
                await self.db.execute(queue, problem_id, vec_str)
                count += 1
            except Exception as e:
                logger.error(f"Failed to save embedding for {problem_id}: {e}")
        logger.info(f"Saved {count}/{len(embeddings)} embeddings")
        return count

    async def search_similar(self, query_embedding: List[float], top_k: int = 10) -> List[dict]:
        vec_str = "[" + ",".join(str(v) for v in query_embedding) + "]"

        query = """
                SELECT pe.raw_problem_id,
                       1 - (pe.embedding <=> $1::vector) AS similarity,
                       rp.title, rp.url, rp.source
                FROM problem_embeddings pe
                         JOIN raw_problems rp ON rp.id = pe.raw_problem_id
                ORDER BY pe.embedding <=> $1::vector
                    LIMIT $2 
                """
        rows = await self.db.fetch(query, vec_str, top_k)
        return [
            {
                "raw_problem_id": str(row["raw_problem_id"]),
                "similarity": float(row["similarity"]),
                "title": row["title"],
                "url": row["url"],
                "source": row["source"],
            }
            for row in rows
        ]
    async def get_all_embeddings(self) -> List[Tuple[str, np.ndarray]]:
        query = """
                SELECT raw_problem_id, embedding::text
                FROM problem_embeddings \
                """
        rows = await self.db.fetch(query)

        results = []
        for row in rows:
            pid = str(row["raw_problem_id"])
            vec_text = row["embedding"].strip("[]")
            vector = np.array([float(x) for x in vec_text.split(",")])
            results.append((pid, vector))

        return results