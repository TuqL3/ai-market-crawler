import json
import logging
from typing import List, Dict

import anthropic

logger = logging.getLogger(__name__)

SUMMARIZE_PROMPT = """\
  You are analyzing a cluster of related software problems. \
  Generate a comprehensive summary.

  Problems in this cluster:
  {problems_text}

  Respond with ONLY a JSON object:
  {{
      "summary": "2-3 sentence overview of the common issue",
      "key_themes": ["theme1", "theme2", "theme3"],
      "common_solutions": ["solution1", "solution2", "solution3"]
  }}

  Rules:
  - summary: concise, describe the root cause pattern
  - key_themes: 2-5 recurring themes across these problems
  - common_solutions: 2-5 practical solutions mentioned or implied
  """


class Summarizer:
    def __init__(self, api_key: str):
        self.client = anthropic.AsyncAnthropic(api_key=api_key)

    async def summarize(self, problems: List[Dict]) -> Dict:
        if not problems:
            return {
                "summary": "Empty cluster",
                "key_themes": [],
                "common_solutions": [],
            }

        parts = []
        for i, p in enumerate(problems[:20]):
            body = (p.get("body") or "")[:300]
            tags = ", ".join(p.get("tags", []))
            parts.append(
                f"{i+1}. [{tags}] {p['title']}\n   {body}"
            )
        problems_text = "\n\n".join(parts)

        prompt = SUMMARIZE_PROMPT.format(problems_text=problems_text)

        try:
            response = await self.client.messages.create(
                model="claude-sonnet-4-20250514",
                max_tokens=1024,
                messages=[{"role": "user", "content": prompt}],
            )

            text = response.content[0].text.strip()
            if text.startswith("```"):
                text = text.split("\n", 1)[1].rsplit("```", 1)[0].strip()

            result = json.loads(text)

            return {
                "summary": result.get("summary", ""),
                "key_themes": result.get("key_themes", []),
                "common_solutions": result.get("common_solutions", []),
            }

        except Exception as e:
            logger.error(f"Summarization failed: {e}")
            return {
                "summary": f"Cluster of {len(problems)} related problems",
                "key_themes": [],
                "common_solutions": [],
            }
