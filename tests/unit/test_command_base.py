from __future__ import annotations

from collections.abc import Callable
from typing import cast

import pytest

from scripts.commands.base import Command


class ConcreteCommand(Command):
    def run(self) -> int:
        return 0


class DelegatingCommand(Command):
    def run(self) -> int:
        base_run = cast(Callable[[Command], int], Command.__dict__["run"])
        return base_run(self)


def test_concrete_command_run_returns_value() -> None:
    assert ConcreteCommand().run() == 0


def test_base_run_raises_not_implemented_error() -> None:
    with pytest.raises(NotImplementedError):
        DelegatingCommand().run()
