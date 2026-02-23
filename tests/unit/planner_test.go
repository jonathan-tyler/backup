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

func TestBuildRestorePlanWSLTarget(t *testing.T) {
	t.Parallel()

	plan, err := backup.BuildRestorePlan(backup.RuntimeWSL, "/tmp/restore")
	if err != nil {
		t.Fatalf("BuildRestorePlan returned error: %v", err)
	}
	if plan.Target != "wsl" {
		t.Fatalf("expected target wsl, got %q", plan.Target)
	}
}

func TestBuildRestorePlanWindowsTarget(t *testing.T) {
	t.Parallel()

	plan, err := backup.BuildRestorePlan(backup.RuntimeWindows, `C:\\restore`)
	if err != nil {
		t.Fatalf("BuildRestorePlan returned error: %v", err)
	}
	if plan.Target != "windows" {
		t.Fatalf("expected target windows, got %q", plan.Target)
	}
}

func TestBuildRestorePlanRequiresTarget(t *testing.T) {
	t.Parallel()

	_, err := backup.BuildRestorePlan(backup.RuntimeWSL, "")
	if err == nil {
		t.Fatal("expected error")
	}
}
