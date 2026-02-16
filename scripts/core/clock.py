from __future__ import annotations

from datetime import datetime, timezone


class Clock:
    def now_iso(self) -> str:
        return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")

    def timestamp(self) -> str:
        return datetime.now().strftime("%Y%m%d-%H%M%S")
