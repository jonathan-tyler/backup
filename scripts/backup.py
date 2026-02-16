#!/usr/bin/env python3

from __future__ import annotations

import argparse
import subprocess
import sys

try:
    from .common import ensure_repo_initialized
    from .common import load_config, now_iso, run_restic, timestamp
except ImportError:
    from common import ensure_repo_initialized
    from common import load_config, now_iso, run_restic, timestamp


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Run notes/repos backup")
    parser.add_argument(
        "env_file",
        nargs="?",
        default=None,
        help="Path to env file (default: config/backup.env)",
    )
    return parser


def run(env_file: str | None = None) -> int:
    config = load_config(env_file)
    ensure_repo_initialized(config)

    log_file = config.report_dir / f"backup-{timestamp()}.log"
    exclude_common = config.project_root / "config" / "excludes" / "common.exclude"
    exclude_notes_repos = (
        config.project_root / "config" / "excludes" / "notes-repos.exclude"
    )

    print(f"Starting backup at {now_iso()}")
    print(f"Repo: {config.restic_repository}")

    cmd = [
        "backup",
        str(config.source_notes),
        str(config.source_repos),
        "--exclude-file",
        str(exclude_common),
        "--exclude-file",
        str(exclude_notes_repos),
        "--tag",
        "hot",
        "--tag",
        "notes-repos",
        "--json",
    ]

    with log_file.open("w", encoding="utf-8") as handle:
        handle.write(f"Starting backup at {now_iso()}\n")
        handle.write(f"Repo: {config.restic_repository}\n")
        process = run_restic(config, cmd, capture_output=True, check=False)
        if process.stdout:
            print(process.stdout, end="")
            handle.write(process.stdout)
        if process.stderr:
            print(process.stderr, end="", file=sys.stderr)
            handle.write(process.stderr)
        if process.returncode != 0:
            raise SystemExit(process.returncode)
        handle.write(f"Backup completed at {now_iso()}\n")

    print(f"Backup completed at {now_iso()}")
    print(f"Log written: {log_file}")
    return 0


def main() -> int:
    args = build_parser().parse_args()
    return run(args.env_file)


if __name__ == "__main__":
    raise SystemExit(main())