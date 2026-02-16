#!/usr/bin/env python3

from __future__ import annotations

import argparse

try:
    from .common import load_config, now_iso, run_restic
except ImportError:
    from common import load_config, now_iso, run_restic


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Run forget/prune retention")
    parser.add_argument(
        "env_file",
        nargs="?",
        default=None,
        help="Path to env file (default: config/backup.env)",
    )
    return parser


def run(env_file: str | None = None) -> int:
    config = load_config(env_file)

    print(f"Running forget/prune at {now_iso()}")
    print(
        "Policy: "
        f"daily={config.keep_daily} "
        f"weekly={config.keep_weekly} "
        f"monthly={config.keep_monthly}"
    )

    run_restic(
        config,
        [
            "forget",
            "--prune",
            "--keep-daily",
            str(config.keep_daily),
            "--keep-weekly",
            str(config.keep_weekly),
            "--keep-monthly",
            str(config.keep_monthly),
        ],
    )

    print(f"Forget/prune completed at {now_iso()}")
    return 0


def main() -> int:
    args = build_parser().parse_args()
    return run(args.env_file)


if __name__ == "__main__":
    raise SystemExit(main())