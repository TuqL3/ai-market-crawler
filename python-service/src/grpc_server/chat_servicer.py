import logging
import os
import sys

gen_path = os.path.join(os.path.dirname(__file__), "..", "..", "gen")
src_path = os.path.join(os.path.dirname(__file__), "..")
sys.path.insert(0, os.path.abspath(gen_path))
sys.path.insert(0, os.path.abspath(src_path))
sys.path.insert(0, "/app/gen")
sys.path.insert(0, "/app/src")

from aggregator.v1 import chat_pb2, chat_pb2_grpc
from ai.rag import RAGPipeline

logger = logging.getLogger(__name__)


class ChatServicer(chat_pb2_grpc.ChatServiceServicer):

    def __init__(self, db, api_key: str):
        self.rag = RAGPipeline(api_key=api_key, db=db)

    async def Ask(self, request, context):
        logger.info(f"Ask: session={request.session_id}, q={request.question[:50]}")
        result = await self.rag.ask(request.question)

        sources = [
            chat_pb2.Source(
                title=s["title"],
                url=s["url"],
                source=s["source"],
                relevance=s["relevance"],
            )
            for s in result["sources"]
        ]
        return chat_pb2.AskResponse(answer=result["answer"], sources=sources)

    async def AskStream(self, request, context):
        logger.info(f"AskStream: session={request.session_id}, q={request.question[:50]}")

        async for chunk in self.rag.ask_stream(request.question):
            sources = [
                chat_pb2.Source(
                    title=s["title"],
                    url=s["url"],
                    source=s["source"],
                    relevance=s["relevance"],
                )
                for s in chunk["sources"]
            ]
            yield chat_pb2.AskStreamResponse(
                content=chunk["content"],
                done=chunk["done"],
                sources=sources,
            )
