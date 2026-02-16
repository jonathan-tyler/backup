from __future__ import annotations

import subprocess
from pathlib import Path
from typing import Iterable, Protocol


class ResticClientProtocol(Protocol):
    def run(
        self,
        args: Iterable[str],
        *,
        capture_output: bool = False,
        check: bool = True,
    ) -> subprocess.CompletedProcess[str]:
        ...

    def ensure_initialized(self) -> None:
        ...

    def snapshots_json(self) -> list[dict]:
        ...


class FileScannerProtocol(Protocol):
    def find_large_files(
        self,
        paths: list[Path],
        threshold_bytes: int,
        limit: int = 20,
    ) -> list[tuple[int, str]]:
        ...

    def find_hotspots(
        self,
        paths: list[Path],
        threshold: int,
        limit: int = 20,
    ) -> list[tuple[int, str]]:
        ...


class ClockProtocol(Protocol):
    def now_iso(self) -> str:
        ...

    def timestamp(self) -> str:
        ...
