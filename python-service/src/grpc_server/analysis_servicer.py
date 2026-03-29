import logging
import os
import sys

import anthropic

gen_path = os.path.join(os.path.dirname(__file__), "..", "..", "gen")
src_path = os.path.join(os.path.dirname(__file__), "..")
sys.path.insert(0, os.path.abspath(gen_path))
sys.path.insert(0, os.path.abspath(src_path))
sys.path.insert(0, "/app/gen")
sys.path.insert(0, "/app/src")

from aggregator.v1 import analysis_pb2, analysis_pb2_grpc

logger = logging.getLogger(__name__)


class AnalysisServicer(analysis_pb2_grpc.AnalysisServiceServicer):

    def __init__(self, db, api_key: str):
        self.db = db
        self.api_key = api_key

        from ai.classifier import Classifier
        from ai.clusterer import Clusterer
        from ai.trend_detector import TrendDetector
        from ai.summarizer import Summarizer
        from embeddings.store import EmbeddingStore

        self.classifier = Classifier(api_key)
        self.clusterer = Clusterer(api_key, db)
        self.trend_detector = TrendDetector(db)
        self.summarizer = Summarizer(api_key)
        self.embedding_store = EmbeddingStore(db)

        self.voyage_api_key = os.getenv("VOYAGE_API_KEY", "")
        self.anthropic_client = anthropic.AsyncAnthropic(api_key=api_key)

    async def ClassifyProblems(self, request, context):
        logger.info(f"ClassifyProblems called with {len(request.problems)} problems")

        problems = [
            {
                "id": p.id,
                "title": p.title,
                "body": p.body,
                "tags": list(p.tags),
            }
            for p in request.problems
        ]

        results = await self.classifier.classify(problems)

        classifications = []
        for r in results:
            classifications.append(
                analysis_pb2.Classification(
                    problem_id=r["problem_id"],
                    category=r["category"],
                    subcategories=r.get("subcategories", []),
                    confidence=float(r.get("confidence", 0.5)),
                )
            )

        logger.info(f"Classified {len(classifications)} problems")
        return analysis_pb2.ClassifyProblemsResponse(classifications=classifications)

    async def EmbedProblems(self, request, context):
        logger.info(f"EmbedProblems called with {len(request.problems)} problems")

        texts = []
        problem_ids = []
        for p in request.problems:
            text = f"{p.title}\n{p.body}" if p.body else p.title
            texts.append(text)
            problem_ids.append(p.id)

        vectors = await self._generate_embeddings(texts)

        embeddings_to_save = list(zip(problem_ids, vectors))
        await self.embedding_store.save_embeddings(embeddings_to_save)

        embed_results = []
        for pid, vec in zip(problem_ids, vectors):
            embed_results.append(
                analysis_pb2.EmbedResult(
                    problem_id=pid,
                    embedding=vec,
                )
            )

        logger.info(f"Embedded {len(embed_results)} problems")
        return analysis_pb2.EmbedProblemsResponse(embeddings=embed_results)

    async def ClusterProblems(self, request, context):
        min_size = request.min_cluster_size if request.min_cluster_size > 0 else 5
        logger.info(f"ClusterProblems called (min_size={min_size})")

        results = await self.clusterer.cluster(min_cluster_size=min_size)

        for r in results:
            await self._save_cluster(r)

        clusters = [
            analysis_pb2.ClusterResult(
                label=r["label"],
                problem_ids=r["problem_ids"],
                cohesion_score=r["cohesion_score"],
            )
            for r in results
        ]

        logger.info(f"Created {len(clusters)} clusters")
        return analysis_pb2.ClusterProblemsResponse(clusters=clusters)

    async def DetectTrends(self, request, context):
        window_days = request.window_days if request.window_days > 0 else 7
        logger.info(f"DetectTrends called (window={window_days}d)")

        results = await self.trend_detector.detect(window_days=window_days)

        for r in results:
            await self._save_trend(r)

        trends = [
            analysis_pb2.TrendResult(
                cluster_id=r["cluster_id"],
                label=r["label"],
                problem_count=r["problem_count"],
                growth_rate=r["growth_rate"],
            )
            for r in results
        ]

        logger.info(f"Detected {len(trends)} trends")
        return analysis_pb2.DetectTrendsResponse(trends=trends)

    async def SummarizeCluster(self, request, context):
        logger.info(f"SummarizeCluster called for cluster {request.cluster_id}")

        problems = [
            {
                "id": p.id,
                "title": p.title,
                "body": p.body,
                "tags": list(p.tags),
            }
            for p in request.problems
        ]

        result = await self.summarizer.summarize(problems)

        await self._update_cluster_summary(
            request.cluster_id, result
        )

        return analysis_pb2.SummarizeClusterResponse(
            summary=result["summary"],
            key_themes=result["key_themes"],
            common_solutions=result["common_solutions"],
        )

    async def _generate_embeddings(self, texts: list) -> list:
        try:
            response = await self.anthropic_client.embeddings.create(
                model="voyage-3",
                input=texts,
            )
            return [item.embedding for item in response.data]
        except Exception as e:
            logger.warning(f"Voyage embedding failed: {e}, using fallback")
            return self._fallback_embeddings(texts)

    def _fallback_embeddings(self, texts: list) -> list:
        import hashlib
        import numpy as np

        vectors = []
        for text in texts:
            h = hashlib.sha512(text.encode()).digest()
            np.random.seed(int.from_bytes(h[:4], "big"))
            vec = np.random.randn(1536).tolist()
            norm = sum(x * x for x in vec) ** 0.5
            vectors.append([x / norm for x in vec])
        return vectors

    async def _save_cluster(self, cluster_data: dict):
        try:
            query = """
                    INSERT INTO problem_clusters (label, cohesion_score, problem_count)
                    VALUES ($1, $2, $3)
                        RETURNING id \
                    """
            row = await self.db.fetchrow(
                query,
                cluster_data["label"],
                cluster_data["cohesion_score"],
                len(cluster_data["problem_ids"]),
            )
            cluster_id = row["id"]

            member_query = """
                           INSERT INTO cluster_members (cluster_id, raw_problem_id)
                           VALUES ($1, $2::uuid)
                               ON CONFLICT DO NOTHING \
                           """
            for pid in cluster_data["problem_ids"]:
                await self.db.execute(member_query, cluster_id, pid)

        except Exception as e:
            logger.error(f"Failed to save cluster: {e}")

    async def _save_trend(self, trend_data: dict):
        try:
            query = """
                    INSERT INTO trend_snapshots
                    (cluster_id, label, problem_count, growth_rate,
                     window_start, window_end)
                    VALUES ($1::uuid, $2, $3, $4, $5::timestamptz, $6::timestamptz) \
                    """
            await self.db.execute(
                query,
                trend_data["cluster_id"],
                trend_data["label"],
                trend_data["problem_count"],
                trend_data["growth_rate"],
                trend_data["window_start"],
                trend_data["window_end"],
            )
        except Exception as e:
            logger.error(f"Failed to save trend: {e}")

    async def _update_cluster_summary(self, cluster_id: str, summary_data: dict):
        try:
            query = """
                    UPDATE problem_clusters
                    SET summary = $1,
                        key_themes = $2,
                        common_solutions = $3,
                        updated_at = NOW()
                    WHERE id = $4::uuid \
                    """
            await self.db.execute(
                query,
                summary_data["summary"],
                summary_data["key_themes"],
                summary_data["common_solutions"],
                cluster_id,
            )
        except Exception as e:
            logger.error(f"Failed to update cluster summary: {e}")