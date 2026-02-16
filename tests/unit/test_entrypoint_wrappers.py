from __future__ import annotations

import argparse
import runpy
import sys
from pathlib import Path
from types import ModuleType

import pytest

import scripts.backup as backup_module
import scripts.forget_prune as forget_prune_module
import scripts.report as report_module
import scripts.restore_smoke as restore_smoke_module
import scripts.commands.factory as factory_module


@pytest.mark.parametrize(
    ("module", "action"),
    [
        (backup_module, "backup"),
        (report_module, "report"),
        (forget_prune_module, "forget-prune"),
        (restore_smoke_module, "restore-smoke"),
    ],
)
def test_run_dispatches_expected_action(
    module: ModuleType,
    action: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    observed: dict[str, object] = {}

    class FakeCommand:
        def run(self) -> int:
            return 7

    class FactorySpy:
        def __init__(self, project_root: Path) -> None:
            observed["project_root"] = project_root

        def create(self, requested_action: str, env_file: str | None) -> FakeCommand:
            observed["action"] = requested_action
            observed["env_file"] = env_file
            return FakeCommand()

    monkeypatch.setattr(module, "CommandFactory", FactorySpy)

    result = module.run("/tmp/custom.env")
    module_file = module.__file__
    assert module_file is not None

    assert result == 7
    assert observed["action"] == action
    assert observed["env_file"] == "/tmp/custom.env"
    assert observed["project_root"] == Path(
        module_file).resolve().parent.parent


@pytest.mark.parametrize(
    "module",
    [backup_module, report_module, forget_prune_module, restore_smoke_module],
)
def test_build_parser_env_file_optional(module: ModuleType) -> None:
    parser = module.build_parser()

    args_default = parser.parse_args([])
    args_custom = parser.parse_args(["/tmp/example.env"])

    assert args_default.env_file is None
    assert args_custom.env_file == "/tmp/example.env"


@pytest.mark.parametrize(
    "module",
    [backup_module, report_module, forget_prune_module, restore_smoke_module],
)
def test_main_parses_and_delegates(module: ModuleType, monkeypatch: pytest.MonkeyPatch) -> None:
    observed: dict[str, str | None] = {}

    class ParserStub:
        def parse_args(self) -> argparse.Namespace:
            return argparse.Namespace(env_file="/tmp/from-main.env")

    def fake_build_parser() -> ParserStub:
        return ParserStub()

    def fake_run(env_file: str | None = None) -> int:
        observed["env_file"] = env_file
        return 11

    monkeypatch.setattr(module, "build_parser", fake_build_parser)
    monkeypatch.setattr(module, "run", fake_run)

    result = module.main()

    assert result == 11
    assert observed["env_file"] == "/tmp/from-main.env"


@pytest.mark.parametrize(
    ("module_name", "action"),
    [
        ("scripts.backup", "backup"),
        ("scripts.report", "report"),
        ("scripts.forget_prune", "forget-prune"),
        ("scripts.restore_smoke", "restore-smoke"),
    ],
)
def test_module_main_guard_executes(
    module_name: str,
    action: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    observed: dict[str, str | None] = {}

    class FakeCommand:
        def run(self) -> int:
            return 0

    class FactorySpy:
        def __init__(self, project_root: Path) -> None:
            _ = project_root

        def create(self, requested_action: str, env_file: str | None) -> FakeCommand:
            observed["action"] = requested_action
            observed["env_file"] = env_file
            return FakeCommand()

    monkeypatch.setattr(factory_module, "CommandFactory", FactorySpy)
    monkeypatch.setattr(sys, "argv", [module_name])
    monkeypatch.delitem(sys.modules, module_name, raising=False)

    with pytest.raises(SystemExit) as exc:
        runpy.run_module(module_name, run_name="__main__")

    assert exc.value.code == 0
    assert observed == {"action": action, "env_file": None}
