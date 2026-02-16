from pathlib import Path
import runpy
import sys

import pytest

import scripts.cli as cli_module
import scripts.commands.factory as factory_module
from scripts.cli import CliApplication


def test_build_parser_accepts_all_actions(tmp_path: Path) -> None:
    app = CliApplication(tmp_path)
    parser = app.build_parser()

    for action in ["backup", "report", "forget-prune", "restore-smoke"]:
        args = parser.parse_args([action])
        assert args.action == action
        assert args.env_file is None


def test_build_parser_accepts_optional_env_file(tmp_path: Path) -> None:
    app = CliApplication(tmp_path)
    parser = app.build_parser()

    args = parser.parse_args(["backup", "/tmp/custom.env"])

    assert args.action == "backup"
    assert args.env_file == "/tmp/custom.env"


def test_run_invokes_factory_and_command(tmp_path: Path) -> None:
    app = CliApplication(tmp_path)
    observed: dict[str, str | None] = {}

    class FakeCommand:
        def run(self) -> int:
            return 9

    class FakeFactory:
        def create(self, action: str, env_file: str | None) -> FakeCommand:
            observed["action"] = action
            observed["env_file"] = env_file
            return FakeCommand()

    app._factory = FakeFactory()  # type: ignore[assignment]

    result = app.run(["report", "/tmp/integration.env"])

    assert result == 9
    assert observed == {"action": "report", "env_file": "/tmp/integration.env"}


def test_main_uses_project_root_and_runs(monkeypatch: pytest.MonkeyPatch) -> None:
    observed: dict[str, Path] = {}

    class AppStub:
        def __init__(self, project_root: Path) -> None:
            observed["project_root"] = project_root

        def run(self) -> int:
            return 13

    monkeypatch.setattr(cli_module, "CliApplication", AppStub)

    result = cli_module.main()

    assert result == 13
    assert observed["project_root"] == Path(
        cli_module.__file__).resolve().parent.parent


def test_cli_module_main_guard_executes(monkeypatch: pytest.MonkeyPatch) -> None:
    observed: dict[str, str | None] = {}

    class FakeCommand:
        def run(self) -> int:
            return 0

    class FactorySpy:
        def __init__(self, project_root: Path) -> None:
            _ = project_root

        def create(self, action: str, env_file: str | None) -> FakeCommand:
            observed["action"] = action
            observed["env_file"] = env_file
            return FakeCommand()

    monkeypatch.setattr(factory_module, "CommandFactory", FactorySpy)
    monkeypatch.setattr(sys, "argv", ["wsl-backup", "backup"])
    monkeypatch.delitem(sys.modules, "scripts.cli", raising=False)

    with pytest.raises(SystemExit) as exc:
        runpy.run_module("scripts.cli", run_name="__main__")

    assert exc.value.code == 0
    assert observed == {"action": "backup", "env_file": None}
