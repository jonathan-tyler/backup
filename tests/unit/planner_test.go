package unit

import (
	"testing"

	backup "wsl-backup-cli/src"
)

func TestBuildRunPlanWSLTarget(t *testing.T) {
	t.Parallel()

	plan, err := backup.BuildRunPlan("daily", backup.RuntimeWSL)
	if err != nil {
		t.Fatalf("BuildRunPlan returned error: %v", err)
	}
	if len(plan.Targets) != 1 || plan.Targets[0] != "wsl" {
		t.Fatalf("expected target wsl, got %#v", plan.Targets)
	}
}

func TestBuildRunPlanWindowsTarget(t *testing.T) {
	t.Parallel()

	plan, err := backup.BuildRunPlan("daily", backup.RuntimeWindows)
	if err != nil {
		t.Fatalf("BuildRunPlan returned error: %v", err)
	}
	if len(plan.Targets) != 1 || plan.Targets[0] != "windows" {
		t.Fatalf("expected target windows, got %#v", plan.Targets)
	}
}
