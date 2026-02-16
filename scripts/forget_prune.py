#!/usr/bin/env python3

from __future__ import annotations

import argparse
from pathlib import Path

from .commands.factory import CommandFactory


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
    project_root = Path(__file__).resolve().parent.parent
    factory = CommandFactory(project_root)
    command = factory.create("forget-prune", env_file)
    return command.run()


def main() -> int:
    args = build_parser().parse_args()
    return run(args.env_file)


if __name__ == "__main__":
    raise SystemExit(main())