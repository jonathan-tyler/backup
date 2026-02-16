from __future__ import annotations

import subprocess

from scripts.commands.forget_prune_command import ForgetPruneCommand


class ResticStub:
    def __init__(self) -> None:
        self.calls: list[list[str]] = []

    def run(self, args, *, capture_output=False, check=True):
        current = list(args)
        self.calls.append(current)
        return subprocess.CompletedProcess(["restic", *args], 0, stdout="", stderr="")

    def ensure_initialized(self) -> None:
        raise NotImplementedError

    def snapshots_json(self) -> list[dict]:
        raise NotImplementedError


def test_forget_prune_command_runs_with_retention_values(sample_config, fixed_clock) -> None:
    restic = ResticStub()
    command = ForgetPruneCommand(sample_config, restic, fixed_clock)

    result = command.run()

    assert result == 0
    assert restic.calls == [
        [
            "forget",
            "--prune",
            "--keep-daily",
            "30",
            "--keep-weekly",
            "12",
            "--keep-monthly",
            "12",
        ]
    ]
