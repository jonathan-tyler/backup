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
			IncludeByCadence: backup.CadencePaths{
				Daily:   []string{`C:\\Users\\test`},
				Weekly:  []string{`C:\\Users\\test`},
				Monthly: []string{`C:\\Users\\test`},
			},
			ExcludeByCadence: backup.CadencePaths{},
			RepositoryHint:   `C:\\repo`,
			UseFSSnapshot:    true,
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

func TestBuildResticInvocationsUsesCadencePaths(t *testing.T) {
	t.Parallel()

	plan := backup.RunPlan{Cadence: "weekly", Targets: []string{"wsl"}}
	config := backup.AppConfig{Profiles: map[string]backup.ProfileConfig{
		"wsl": {
			IncludeByCadence: backup.CadencePaths{
				Daily:   []string{"/home/test/daily"},
				Weekly:  []string{"/home/test/daily", "/home/test/weekly"},
				Monthly: []string{"/home/test/daily", "/home/test/weekly", "/home/test/monthly"},
			},
			ExcludeByCadence: backup.CadencePaths{
				Daily:   []string{"/home/test/.cache"},
				Weekly:  []string{"/home/test/.cache", "/home/test/tmp"},
				Monthly: []string{"/home/test/.cache"},
			},
			RepositoryHint: "/repo",
		},
	}}

	invocations, err := backup.BuildResticInvocations(plan, config)
	if err != nil {
		t.Fatalf("BuildResticInvocations returned error: %v", err)
	}
	if len(invocations) != 1 {
		t.Fatalf("expected one invocation, got %d", len(invocations))
	}

	args := invocations[0].Args
	serialized := fmt.Sprintf("%q", args)
	if serialized == "" {
		t.Fatal("expected invocation args")
	}
	if args[len(args)-2] != "/home/test/daily" || args[len(args)-1] != "/home/test/weekly" {
		t.Fatalf("unexpected weekly include args: %#v", args)
	}
}

func TestBuildRestoreInvocationUsesLatestAndTarget(t *testing.T) {
	t.Parallel()

	plan := backup.RestorePlan{Target: "wsl", RestoreTarget: "/tmp/restore"}
	config := backup.AppConfig{Profiles: map[string]backup.ProfileConfig{
		"wsl": {
			RepositoryHint: "/repo",
		},
	}}

	invocation, err := backup.BuildRestoreInvocation(plan, config)
	if err != nil {
		t.Fatalf("BuildRestoreInvocation returned error: %v", err)
	}

	if invocation.Executable != "restic" {
		t.Fatalf("expected restic executable, got %q", invocation.Executable)
	}
	serialized := fmt.Sprintf("%q", invocation.Args)
	if serialized == "" {
		t.Fatal("expected invocation args")
	}
	if len(invocation.Args) < 6 {
		t.Fatalf("unexpected restore args: %#v", invocation.Args)
	}
	if invocation.Args[0] != "-r" || invocation.Args[2] != "restore" || invocation.Args[3] != "latest" {
		t.Fatalf("unexpected restore args: %#v", invocation.Args)
	}
	if invocation.Args[4] != "--target" || invocation.Args[5] != "/tmp/restore" {
		t.Fatalf("unexpected restore target args: %#v", invocation.Args)
	}
}
