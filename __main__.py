from __future__ import annotations

from argparse import ArgumentParser, Namespace


def register_args(parser: ArgumentParser) -> None:
    subparsers = parser.add_subparsers(dest="command", required=True)

    run_parser = subparsers.add_parser(
        "run",
        help="Run a backup for a given cadence.",
    )
    run_parser.add_argument(
        "cadence",
        choices=("daily", "weekly", "monthly"),
        help="Backup cadence to run.",
    )

    report_parser = subparsers.add_parser(
        "report",
        help="Run a backup report for a given cadence.",
    )
    report_parser.add_argument(
        "cadence",
        choices=("daily", "weekly", "monthly"),
        help="Backup cadence to report on.",
    )

    restore_parser = subparsers.add_parser(
        "restore",
        help="Restore from a backup target.",
    )
    restore_parser.add_argument(
        "target",
        type=str,
        help="Backup target name/path to restore from.",
    )


def run(args: Namespace) -> None:
    if args.command == "run":
        print(f"{args.cadence} backup run is not implemented yet.")
        return

    if args.command == "report":
        print(f"{args.cadence} backup report is not implemented yet.")
        return

    if args.command == "restore":
        print(f"restore is not implemented yet (target: {args.target}).")
        return

    raise SystemExit(f"Unknown command: {args.command}")


if __name__ == "__main__":
    parser = ArgumentParser(description="Backup CLI")
    register_args(parser)
    run(parser.parse_args())
