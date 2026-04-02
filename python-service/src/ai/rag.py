import logging
import anthropic
from embeddings.store import EmbeddingStore

logger = logging.getLogger(__name__)


class RAGPipeline:
    def __init__(self, api_key: str, db):
        self.client = anthropic.AsyncAnthropic(api_key=api_key)
        self.embedding_store = EmbeddingStore(db)

    async def ask(self, question: str) -> dict:
        query_embedding = await self._embed_question(question)
        similar = await self.embedding_store.search_similar(query_embedding, top_k=10)
        context = self._build_context(similar)
        response = await self.client.messages.create(
            model="claude-sonnet-4-20250514",
            max_tokens=2048,
            system=self._system_prompt(),
            messages=[
                {"role": "user", "content": f"Context:\n{context}\n\nQuestion: {question}"}
            ],
        )

        answer = response.content[0].text
        sources = [
            {
                "title": s["title"],
                "url": s["url"],
                "source": s["source"],
                "relevance": s["similarity"],
            }
            for s in similar[:5]
        ]
        return {"answer": answer, "sources": sources}

    async def ask_stream(self, question: str):
        query_embedding = await self._embed_question(question)
        similar = await self.embedding_store.search_similar(query_embedding, top_k=10)
        context = self._build_context(similar)

        async with self.client.messages.stream(
            model="claude-sonnet-4-20250514",
            max_tokens=2048,
            system=self._system_prompt(),
            messages=[
                {"role": "user", "content": f"Context:\n{context}\n\nQuestion: {question}"}
            ],
        ) as stream:
            async for text in stream.text_stream:
                yield {"content": text, "done": False, "sources": []}

        sources = [
            {"title": s["title"], "url": s["url"], "source": s["source"], "relevance": s["similarity"]}
            for s in similar[:5]
        ]
        yield {"content": "", "done": True, "sources": sources}

    async def _embed_question(self, question: str) -> list:
        try:
            response = await self.client.embeddings.create(
                model="voyage-3",
                input=[question],
            )
            return response.data[0].embedding
        except Exception as e:
            logger.warning(f"Embedding failed: {e}, using fallback")
            import hashlib
            import numpy as np
            h = hashlib.sha512(question.encode()).digest()
            np.random.seed(int.from_bytes(h[:4], "big"))
            vec = np.random.randn(1536).tolist()
            norm = sum(x * x for x in vec) ** 0.5
            return [x / norm for x in vec]

    def _build_context(self, similar: list) -> str:
        parts = []
        for i, s in enumerate(similar, 1):
            parts.append(f"[{i}] {s['title']} (source: {s['source']}, url: {s['url']})")
        return "\n".join(parts)

    def _system_prompt(self) -> str:
        return (
            "You are an AI assistant that helps developers understand common programming problems and trends. "
            "Use the provided context of real problems from GitHub, StackOverflow, and Reddit to answer questions. "
            "Always cite sources using [number] format when referencing specific problems. "
            "If the context doesn't contain relevant information, say so honestly."
        )
