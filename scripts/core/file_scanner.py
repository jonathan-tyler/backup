from __future__ import annotations

import collections
import os
from pathlib import Path


class FileScanner:
    def find_large_files(
        self,
        paths: list[Path],
        threshold_bytes: int,
        limit: int = 20,
    ) -> list[tuple[int, str]]:
        results: list[tuple[int, str]] = []
        for base_path in paths:
            if not base_path.exists():
                continue
            for root, _, files in os.walk(base_path):
                for name in files:
                    full_path = Path(root) / name
                    try:
                        size_bytes = full_path.stat().st_size
                    except OSError:
                        continue
                    if size_bytes > threshold_bytes:
                        results.append((size_bytes, str(full_path)))
        results.sort(key=lambda item: item[0], reverse=True)
        return results[:limit]

    def find_hotspots(
        self,
        paths: list[Path],
        threshold: int,
        limit: int = 20,
    ) -> list[tuple[int, str]]:
        counts: collections.Counter[str] = collections.Counter()
        for base_path in paths:
            if not base_path.exists():
                continue
            for root, _, files in os.walk(base_path):
                file_count = len(files)
                if file_count > 0:
                    counts[str(Path(root))] += file_count
        hotspots = [
            (count, directory)
            for directory, count in counts.items()
            if count > threshold
        ]
        hotspots.sort(key=lambda item: item[0], reverse=True)
        return hotspots[:limit]
