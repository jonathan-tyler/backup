#!/usr/bin/env python3

from __future__ import annotations

import argparse

from .backup import run as backup_run
from .forget_prune import run as forget_prune_run
from .report import run as report_run
from .restore_smoke import run as restore_smoke_run


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="wsl-backup",
        description="WSL backup operations CLI",
    )
    parser.add_argument(
        "action",
        choices=["backup", "report", "forget-prune", "restore-smoke"],
    )
    parser.add_argument(
        "env_file",
        nargs="?",
        default=None,
        help="Optional path to env file (default: config/backup.env)",
    )
    return parser


def main() -> int:
    args = build_parser().parse_args()

    action_map = {
        "backup": backup_run,
        "report": report_run,
        "forget-prune": forget_prune_run,
        "restore-smoke": restore_smoke_run,
    }
    handler = action_map[args.action]
    return handler(args.env_file)


if __name__ == "__main__":
    raise SystemExit(main())