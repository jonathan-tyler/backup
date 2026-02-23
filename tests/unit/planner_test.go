package unit

import (
	"strings"
	"testing"

	backup "wsl-backup-cli/src"
)

func TestBuildRunPlanWSLTarget(t *testing.T) {
	t.Parallel()

	plan, err := backup.BuildRunPlan("daily", backup.RuntimeWSL)
	if err != nil {
		t.Fatalf("BuildRunPlan returned error: %v", err)
	}
	if len(plan.Targets) != 2 || plan.Targets[0] != "wsl" || plan.Targets[1] != "windows" {
		t.Fatalf("expected targets [wsl windows], got %#v", plan.Targets)
	}
}

func TestBuildRunPlanWindowsRejected(t *testing.T) {
	t.Parallel()

	_, err := backup.BuildRunPlan("daily", backup.RuntimeWindows)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "must run inside WSL") {
		t.Fatalf("unexpected error: %q", err.Error())
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

func TestBuildRestorePlanWindowsRejected(t *testing.T) {
	t.Parallel()

	_, err := backup.BuildRestorePlan(backup.RuntimeWindows, `C:\\restore`)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "must run inside WSL") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestBuildRestorePlanRequiresTarget(t *testing.T) {
	t.Parallel()

	_, err := backup.BuildRestorePlan(backup.RuntimeWSL, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindPlatformIncludeOverlapWarningsEquivalentWindowsAndWSLPaths(t *testing.T) {
	t.Parallel()

	plan := backup.RunPlan{Cadence: "daily", Targets: []string{"wsl", "windows"}}
	config := backup.AppConfig{Profiles: map[string]backup.ProfileConfig{
		"wsl": {
			IncludeByCadence: backup.CadencePaths{Daily: []string{"/mnt/c/Users/daily"}},
		},
		"windows": {
			IncludeByCadence: backup.CadencePaths{Daily: []string{`C:\Users\daily`}},
		},
	}}

	warnings := backup.FindPlatformIncludeOverlapWarnings(plan, config)
	if len(warnings) == 0 {
		t.Fatal("expected overlap warning")
	}

	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "platform include overlap detected") {
		t.Fatalf("expected overlap message, got %q", joined)
	}
	if !strings.Contains(joined, "wslpath -w") {
		t.Fatalf("expected wslpath translation tip, got %q", joined)
	}
}

func TestFindPlatformIncludeOverlapWarningsNoOverlap(t *testing.T) {
	t.Parallel()

	plan := backup.RunPlan{Cadence: "daily", Targets: []string{"wsl", "windows"}}
	config := backup.AppConfig{Profiles: map[string]backup.ProfileConfig{
		"wsl": {
			IncludeByCadence: backup.CadencePaths{Daily: []string{"/home/daily"}},
		},
		"windows": {
			IncludeByCadence: backup.CadencePaths{Daily: []string{`C:\Users\daily`}},
		},
	}}

	warnings := backup.FindPlatformIncludeOverlapWarnings(plan, config)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
}
