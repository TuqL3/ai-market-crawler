import logging
from datetime import datetime, timedelta, timezone
from typing import List, Dict

logger = logging.getLogger(__name__)

class TrendDetector:
    def __init__(self, db):
        self.db = db

    async def detect(self, window_days: int = 7) -> List[Dict]:

        now = datetime.now(timezone.utc)
        current_start = now - timedelta(days=window_days)
        previous_start = current_start - timedelta(days=window_days)

        current_counts = await self._count_by_cluster(current_start, now)

        previous_counts = await self._count_by_cluster(previous_start, current_start)

        all_cluster_ids = set(current_counts.keys()) | set(previous_counts.keys())

        results = []
        for cluster_id in all_cluster_ids:
            curr = current_counts.get(cluster_id, {})
            prev = previous_counts.get(cluster_id, {})
            curr_count = curr.get("count", 0)
            prev_count = prev.get("count", 0)
            label = curr.get("label") or prev.get("label", "Unknown")

            if prev_count > 0:
                growth_rate = (curr_count - prev_count) / prev_count
            elif curr_count > 0:
                growth_rate = 1.0
            else:
                growth_rate = 0.0

            results.append({
                "cluster_id": cluster_id,
                "label": label,
                "problem_count": curr_count,
                "growth_rate": round(growth_rate, 4),
                "window_start": current_start.isoformat(),
                "window_end": now.isoformat(),
            })

        results.sort(key=lambda x: x["growth_rate"], reverse=True)

        logger.info(f"Detected trends for {len(results)} clusters (window={window_days}d)")
        return results

    async def _count_by_cluster(
            self, start: datetime, end: datetime
    ) -> Dict[str, Dict]:
        query = """
                SELECT pc.id AS cluster_id,
                       pc.label,
                       COUNT(cm.raw_problem_id) AS problem_count
                FROM problem_clusters pc
                         JOIN cluster_members cm ON cm.cluster_id = pc.id
                         JOIN raw_problems rp ON rp.id = cm.raw_problem_id
                WHERE rp.crawled_at >= $1 AND rp.crawled_at < $2
                GROUP BY pc.id, pc.label \
                """
        rows = await self.db.fetch(query, start, end)

        return {
            str(row["cluster_id"]): {
                "count": row["problem_count"],
                "label": row["label"],
            }
            for row in rows
        }