import os


class Config:
    DATABASE_URL = os.getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ai_aggregator")
    GRPC_PORT = int(os.getenv("GRPC_PORT", "50052"))
    ANTHROPIC_API_KEY = os.getenv("ANTHROPIC_API_KEY", "")
    VOYAGE_API_KEY = os.getenv("VOYAGE_API_KEY", "")
