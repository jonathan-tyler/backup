package unit

import (
	"fmt"
	"testing"

	backup "wsl-backup-cli/src"
)

type fakeExecutor struct {
	calls []string
}

func (executor *fakeExecutor) Run(name string, args ...string) (string, error) {
	executor.calls = append(executor.calls, name)
	if name == "fail" {
		return "", fmt.Errorf("boom")
	}
	return "ok", nil
}

func TestBuildResticInvocationsWindowsUsesExe(t *testing.T) {
	t.Parallel()

	plan := backup.RunPlan{Cadence: "daily", Targets: []string{"windows"}}
	config := backup.AppConfig{Profiles: map[string]backup.ProfileConfig{
		"windows": {
			IncludePaths:   []string{`C:\\Users\\test`},
			RepositoryHint: `C:\\repo`,
			UseFSSnapshot:  true,
		},
	}}

	invocations, err := backup.BuildResticInvocations(plan, config)
	if err != nil {
		t.Fatalf("BuildResticInvocations returned error: %v", err)
	}
	if invocations[0].Executable != "restic.exe" {
		t.Fatalf("expected restic.exe, got %q", invocations[0].Executable)
	}
}

func TestExecuteResticInvocationsRunsExecutable(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{}
	invocations := []backup.ResticInvocation{{Target: "wsl", Executable: "restic", Args: []string{"snapshots"}}}

	_, err := backup.ExecuteResticInvocations(invocations, executor)
	if err != nil {
		t.Fatalf("ExecuteResticInvocations returned error: %v", err)
	}
	if len(executor.calls) != 1 || executor.calls[0] != "restic" {
		t.Fatalf("unexpected calls: %#v", executor.calls)
	}
}
