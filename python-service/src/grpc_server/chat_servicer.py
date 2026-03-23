import logging
import os
import sys

gen_path = os.path.join(os.path.dirname(__file__), "..", "..", "gen")
sys.path.insert(0, os.path.abspath(gen_path))
sys.path.insert(0, "/app/gen")

from aggregator.v1 import chat_pb2, chat_pb2_grpc

logger = logging.getLogger(__name__)


class ChatServicer(chat_pb2_grpc.ChatServiceServicer):
    """Stub implementation - returns dummy responses."""

    async def Ask(self, request, context):
        logger.info(f"Ask called: session={request.session_id}, q={request.question[:50]}")
        return chat_pb2.AskResponse(
            answer=f"Stub answer for: {request.question}",
            sources=[],
        )

    async def AskStream(self, request, context):
        logger.info(f"AskStream called: session={request.session_id}")
        words = f"Stub streaming answer for: {request.question}".split()
        for word in words:
            yield chat_pb2.AskStreamResponse(content=word + " ", done=False, sources=[])
        yield chat_pb2.AskStreamResponse(content="", done=True, sources=[])
