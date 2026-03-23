import logging
import os
import sys

gen_path = os.path.join(os.path.dirname(__file__), "..", "..", "gen")
sys.path.insert(0, os.path.abspath(gen_path))
sys.path.insert(0, "/app/gen")

from aggregator.v1 import analysis_pb2, analysis_pb2_grpc

logger = logging.getLogger(__name__)


class AnalysisServicer(analysis_pb2_grpc.AnalysisServiceServicer):
    """Stub implementation - returns dummy data for now."""

    async def ClassifyProblems(self, request, context):
        logger.info(f"ClassifyProblems called with {len(request.problems)} problems")
        classifications = []
        for p in request.problems:
            classifications.append(
                analysis_pb2.Classification(
                    problem_id=p.id,
                    category="uncategorized",
                    subcategories=[],
                    confidence=0.0,
                )
            )
        return analysis_pb2.ClassifyProblemsResponse(classifications=classifications)

    async def ClusterProblems(self, request, context):
        logger.info("ClusterProblems called (stub)")
        return analysis_pb2.ClusterProblemsResponse(clusters=[])

    async def DetectTrends(self, request, context):
        logger.info("DetectTrends called (stub)")
        return analysis_pb2.DetectTrendsResponse(trends=[])

    async def SummarizeCluster(self, request, context):
        logger.info("SummarizeCluster called (stub)")
        return analysis_pb2.SummarizeClusterResponse(
            summary="Stub summary",
            key_themes=[],
            common_solutions=[],
        )

    async def EmbedProblems(self, request, context):
        logger.info(f"EmbedProblems called with {len(request.problems)} problems")
        return analysis_pb2.EmbedProblemsResponse(embeddings=[])
