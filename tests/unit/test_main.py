from __future__ import annotations

import importlib.util
import runpy
from argparse import ArgumentParser, Namespace
from pathlib import Path

import pytest

MODULE_PATH = Path(__file__).resolve().parents[2] / "__main__.py"
SPEC = importlib.util.spec_from_file_location("backup_cli_main", MODULE_PATH)
if SPEC is None or SPEC.loader is None:
	raise RuntimeError("Unable to load backup CLI module for tests.")
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)

register_args = MODULE.register_args
run = MODULE.run


@pytest.fixture()
def parser() -> ArgumentParser:
	backup_parser = ArgumentParser(description="Backup CLI")
	register_args(backup_parser)
	return backup_parser


@pytest.mark.parametrize("cadence", ["daily", "weekly", "monthly"])
def test_register_args_parses_run_command(parser: ArgumentParser, cadence: str) -> None:
	args = parser.parse_args(["run", cadence])
	assert args.command == "run"
	assert args.cadence == cadence


@pytest.mark.parametrize("cadence", ["daily", "weekly", "monthly"])
def test_register_args_parses_report_command(parser: ArgumentParser, cadence: str) -> None:
	args = parser.parse_args(["report", cadence])
	assert args.command == "report"
	assert args.cadence == cadence


def test_register_args_parses_restore_command(parser: ArgumentParser) -> None:
	args = parser.parse_args(["restore", "target-dir"])
	assert args.command == "restore"
	assert args.target == "target-dir"


# def test_run_prints_run_message(capsys: pytest.CaptureFixture[str]) -> None:
# 	run(Namespace(command="run", cadence="weekly"))
# 	captured = capsys.readouterr()
# 	assert captured.out.strip() == "weekly backup run is not implemented yet."


# def test_run_prints_report_message(capsys: pytest.CaptureFixture[str]) -> None:
# 	run(Namespace(command="report", cadence="monthly"))
# 	captured = capsys.readouterr()
# 	assert captured.out.strip() == "monthly backup report is not implemented yet."


# def test_run_prints_restore_message(capsys: pytest.CaptureFixture[str]) -> None:
# 	run(Namespace(command="restore", target="repo/snapshot"))
# 	captured = capsys.readouterr()
# 	assert captured.out.strip() == "restore is not implemented yet (target: repo/snapshot)."


def test_run_unknown_command_raises_system_exit() -> None:
	with pytest.raises(SystemExit, match="Unknown command: invalid"):
		run(Namespace(command="invalid"))


def test_module_executes_main_block(
	monkeypatch: pytest.MonkeyPatch,
	capsys: pytest.CaptureFixture[str],
) -> None:
	class FakeCommandParser:
		def add_argument(self, *_args: object, **_kwargs: object) -> None:
			return None

	class FakeSubparsers:
		def add_parser(self, *_args: object, **_kwargs: object) -> FakeCommandParser:
			return FakeCommandParser()

	class FakeParser:
		def __init__(self, description: str):
			self.description = description

		def add_subparsers(self, **_kwargs: object) -> FakeSubparsers:
			return FakeSubparsers()

		def parse_args(self) -> Namespace:
			return Namespace(command="run", cadence="daily")

	captured: dict[str, object] = {}

	def fake_argument_parser(*, description: str) -> FakeParser:
		captured["description"] = description
		parser_instance = FakeParser(description=description)
		captured["parser"] = parser_instance
		return parser_instance

	monkeypatch.setattr("argparse.ArgumentParser", fake_argument_parser)

	runpy.run_path("__main__.py", run_name="__main__")

	assert captured["description"] == "Backup CLI"
	assert captured["parser"].description == "Backup CLI"
	printed = capsys.readouterr().out.strip()
	assert printed == "daily backup run is not implemented yet."

