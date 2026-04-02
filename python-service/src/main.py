import asyncio
import logging
import os
import sys

import grpc
from grpc import aio

gen_path = os.path.join(os.path.dirname(__file__), "..", "gen")
sys.path.insert(0, os.path.abspath(gen_path))
sys.path.insert(0, "/app/gen")

from aggregator.v1 import analysis_pb2_grpc, chat_pb2_grpc
from grpc_server.analysis_servicer import AnalysisServicer
from grpc_server.chat_servicer import ChatServicer
from config import Config
from db.connection import Database

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(name)s %(levelname)s %(message)s",
)
logger = logging.getLogger(__name__)


async def serve():
    db = Database(Config.DATABASE_URL)
    await db.connect()
    logger.info("Database connected")

    server = aio.server()

    analysis_pb2_grpc.add_AnalysisServiceServicer_to_server(
        AnalysisServicer(db=db, api_key=Config.ANTHROPIC_API_KEY), server
    )
    chat_pb2_grpc.add_ChatServiceServicer_to_server(
      ChatServicer(db=db, api_key=Config.ANTHROPIC_API_KEY), server
  )

    listen_addr = f"0.0.0.0:{Config.GRPC_PORT}"
    server.add_insecure_port(listen_addr)

    logger.info(f"Python gRPC server starting on {listen_addr}")
    await server.start()
    logger.info("Server started successfully")

    try:
        await server.wait_for_termination()
    finally:
        await db.close()
        logger.info("Database connection closed")


if __name__ == "__main__":
    asyncio.run(serve())
