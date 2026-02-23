package backup

import (
	"fmt"
	"os"
	"strings"
)

type Runtime string

const (
	RuntimeWSL     Runtime = "wsl"
	RuntimeWindows Runtime = "windows"
	RuntimeLinux   Runtime = "linux"
)

type RunPlan struct {
	Cadence string
	Targets []string
}

type RestorePlan struct {
	Target        string
	RestoreTarget string
}

func DetectRuntime() Runtime {
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return RuntimeWSL
	}
	if os.Getenv("OS") == "Windows_NT" {
		return RuntimeWindows
	}
	if data, err := os.ReadFile("/proc/version"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "microsoft") || strings.Contains(content, "wsl") {
			return RuntimeWSL
		}
	}
	return RuntimeLinux
}

func BuildRunPlan(cadence string, runtime Runtime) (RunPlan, error) {
	if cadence == "" {
		return RunPlan{}, fmt.Errorf("missing cadence")
	}

	switch runtime {
	case RuntimeWindows:
		return RunPlan{Cadence: cadence, Targets: []string{"windows"}}, nil
	case RuntimeWSL, RuntimeLinux:
		return RunPlan{Cadence: cadence, Targets: []string{"wsl"}}, nil
	default:
		return RunPlan{}, fmt.Errorf("unknown runtime: %s", runtime)
	}
}

func BuildRestorePlan(runtime Runtime, restoreTarget string) (RestorePlan, error) {
	if strings.TrimSpace(restoreTarget) == "" {
		return RestorePlan{}, fmt.Errorf("missing target")
	}

	switch runtime {
	case RuntimeWindows:
		return RestorePlan{Target: "windows", RestoreTarget: restoreTarget}, nil
	case RuntimeWSL, RuntimeLinux:
		return RestorePlan{Target: "wsl", RestoreTarget: restoreTarget}, nil
	default:
		return RestorePlan{}, fmt.Errorf("unknown runtime: %s", runtime)
	}
}
