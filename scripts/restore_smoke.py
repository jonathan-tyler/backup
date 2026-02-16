#!/usr/bin/env python3

from __future__ import annotations

import argparse
import shutil
import tempfile
from pathlib import Path

try:
    from .common import load_config, now_iso, run_restic
except ImportError:
    from common import load_config, now_iso, run_restic


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Run restore smoke test")
    parser.add_argument(
        "env_file",
        nargs="?",
        default=None,
        help="Path to env file (default: config/backup.env)",
    )
    return parser


def run(env_file: str | None = None) -> int:
    config = load_config(env_file)

    tmp_dir = Path(tempfile.mkdtemp())
    restore_dir = tmp_dir / "restore"
    restore_dir.mkdir(parents=True, exist_ok=True)

    print(f"Running restore smoke test at {now_iso()}")
    print(f"Temporary restore path: {restore_dir}")

    try:
        snapshots = run_restic(config, ["snapshots"], capture_output=True, check=False)
        if snapshots.returncode != 0:
            print("No snapshots found. Run backup first.")
            return 1

        run_restic(
            config,
            [
                "restore",
                "latest",
                "--target",
                str(restore_dir),
                "--include",
                str(config.source_notes),
                "--verify",
            ],
        )

        if not restore_dir.is_dir():
            print("Restore failed: target directory not created.")
            return 1

        restored_files = sum(1 for _ in restore_dir.rglob("*") if _.is_file())
        print(f"Restore smoke test complete. Restored files: {restored_files}")
        return 0
    finally:
        shutil.rmtree(tmp_dir, ignore_errors=True)


def main() -> int:
    args = build_parser().parse_args()
    return run(args.env_file)


if __name__ == "__main__":
    raise SystemExit(main())