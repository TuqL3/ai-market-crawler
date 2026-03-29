import json
from typing import List
import logging
import anthropic

logger = logging.getLogger(__name__)

CLASSIFY_PROMPT = """\
  You are a software problem classifier. Classify each problem into exactly ONE category.

  Categories:
  - bug: Software bugs, errors, crashes
  - performance: Slow performance, memory leaks
  - security: Vulnerabilities, authentication issues
  - compatibility: Version conflicts, dependency problems
  - documentation: Missing/wrong docs
  - feature-request: New feature suggestions
  - devops: CI/CD, deployment, container issues
  - testing: Test failures, coverage issues
  - ui-ux: Frontend bugs, layout issues
  - data: Database issues, data corruption
  - networking: API errors, connectivity, timeout
  - other: Anything else

  For each problem, respond with a JSON array:
  [
    {{
      "problem_id": "the-id",
      "category": "one-of-above",
      "subcategories": ["specific-tag-1", "specific-tag-2"],
      "confidence": 0.85
    }}
  ]

  Respond ONLY with JSON array.

  Problems:
  {problems_json}
  """

class Classifier:
    def __init__(self, api_key: str):
        self.client = anthropic.AsyncAnthropic(api_key=api_key)
    async def classify(self, problems: List[dict]) -> List[dict]:
        if not problems:
            return []

        problems_json = json.dumps([
            {
                "id": p["id"],
                "title": p["title"],
                "body": (p.get("body") or "")[:500],
                "tags": p.get("tags",[])
            }
            for p in problems
        ], indent=2)

        prompt = CLASSIFY_PROMPT.format(problems_json=problems_json)

        try:
            response = await self.client.messages.create(
                model="claude-sonnet-4-20250514",
                max_tokens=4096,
                messages=[{"role":"user","content":prompt}]
            )

            text = response.content[0].text.strip()

            if text.startswith("```"):
                text = text.split("\n", 1)[1].rsplit("```", 1)[0].strip()

            return json.loads(text)
        except Exception as e:
            logger.error(f"Classification failed: {e}")

            return [
                {"problem_id":p["id"], "category":"other", "subcategories":[], "confidence":0.0} for p in problems
            ]
