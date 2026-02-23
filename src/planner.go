package backup

import (
	"fmt"
	"os"
	"path"
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
	if runtime != RuntimeWSL {
		return RunPlan{}, fmt.Errorf("backup CLI must run inside WSL")
	}

	switch runtime {
	case RuntimeWSL:
		return RunPlan{Cadence: cadence, Targets: []string{"wsl", "windows"}}, nil
	default:
		return RunPlan{}, fmt.Errorf("unknown platform: %s", runtime)
	}
}

func BuildRestorePlan(runtime Runtime, restoreTarget string) (RestorePlan, error) {
	if strings.TrimSpace(restoreTarget) == "" {
		return RestorePlan{}, fmt.Errorf("missing target")
	}
	if runtime != RuntimeWSL {
		return RestorePlan{}, fmt.Errorf("backup CLI must run inside WSL")
	}

	switch runtime {
	case RuntimeWSL:
		return RestorePlan{Target: "wsl", RestoreTarget: restoreTarget}, nil
	default:
		return RestorePlan{}, fmt.Errorf("unknown platform: %s", runtime)
	}
}

func FindPlatformIncludeOverlapWarnings(plan RunPlan, config AppConfig) []string {
	if len(plan.Targets) < 2 {
		return nil
	}

	type includeItem struct {
		target     string
		rawPath    string
		normalized string
	}

	items := make([]includeItem, 0)
	for _, target := range plan.Targets {
		profile, ok := config.Profiles[target]
		if !ok {
			continue
		}
		for _, includePath := range profile.IncludeByCadence.ForCadence(plan.Cadence) {
			normalized := normalizePlatformPathForOverlap(includePath)
			if normalized == "" {
				continue
			}
			items = append(items, includeItem{target: target, rawPath: includePath, normalized: normalized})
		}
	}

	warnings := make([]string, 0)
	seen := map[string]struct{}{}
	pairKey := func(left includeItem, right includeItem) string {
		first := left.target + "|" + left.normalized + "|" + left.rawPath
		second := right.target + "|" + right.normalized + "|" + right.rawPath
		if first <= second {
			return first + "::" + second
		}
		return second + "::" + first
	}

	for leftIndex := 0; leftIndex < len(items); leftIndex++ {
		for rightIndex := leftIndex + 1; rightIndex < len(items); rightIndex++ {
			left := items[leftIndex]
			right := items[rightIndex]
			if left.target == right.target {
				continue
			}
			if !pathsOverlap(left.normalized, right.normalized) {
				continue
			}

			key := pairKey(left, right)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}

			warnings = append(warnings,
				fmt.Sprintf("warning: platform include overlap detected: %s=%s overlaps %s=%s", left.target, left.rawPath, right.target, right.rawPath),
			)
		}
	}

	if len(warnings) > 0 {
		warnings = append(warnings,
			"warning: path translation tip: use 'wslpath <path>' and 'wslpath -w <path>' to compare equivalents.",
		)
	}

	return warnings
}

func normalizePlatformPathForOverlap(rawPath string) string {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return ""
	}

	normalizedSlashes := strings.ReplaceAll(trimmed, "\\", "/")
	if len(normalizedSlashes) >= 3 && normalizedSlashes[1] == ':' && normalizedSlashes[2] == '/' {
		drive := strings.ToLower(normalizedSlashes[:1])
		rest := normalizedSlashes[2:]
		normalizedSlashes = "/mnt/" + drive + rest
	}

	normalizedSlashes = strings.ToLower(normalizedSlashes)
	cleaned := path.Clean(normalizedSlashes)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func pathsOverlap(left string, right string) bool {
	return isSameOrParentPath(left, right) || isSameOrParentPath(right, left)
}

func isSameOrParentPath(parent string, child string) bool {
	if parent == child {
		return true
	}
	if parent == "/" {
		return true
	}
	return strings.HasPrefix(child, parent+"/")
}
