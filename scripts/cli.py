#!/usr/bin/env python3

from __future__ import annotations

import argparse
from pathlib import Path

from .commands.factory import CommandFactory


class CliApplication:
    def __init__(self, project_root: Path) -> None:
        self._factory = CommandFactory(project_root)

    def build_parser(self) -> argparse.ArgumentParser:
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

    def run(self, argv: list[str] | None = None) -> int:
        args = self.build_parser().parse_args(argv)
        command = self._factory.create(args.action, args.env_file)
        return command.run()


def main() -> int:
    project_root = Path(__file__).resolve().parent.parent
    app = CliApplication(project_root)
    return app.run()


if __name__ == "__main__":
    raise SystemExit(main())