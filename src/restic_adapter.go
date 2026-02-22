package backup

import "fmt"

type ResticInvocation struct {
	Target     string
	Executable string
	Args       []string
}

func BuildResticInvocations(plan RunPlan, config AppConfig) ([]ResticInvocation, error) {
	invocations := make([]ResticInvocation, 0, len(plan.Targets))

	for _, target := range plan.Targets {
		profile, ok := config.Profiles[target]
		if !ok {
			return nil, fmt.Errorf("missing profile config: %s", target)
		}
		includePaths := profile.IncludeByCadence.ForCadence(plan.Cadence)
		excludePaths := profile.ExcludeByCadence.ForCadence(plan.Cadence)

		if len(includePaths) == 0 {
			return nil, fmt.Errorf("missing include paths for target: %s", target)
		}
		if profile.RepositoryHint == "" {
			return nil, fmt.Errorf("missing repository for target: %s", target)
		}

		args := []string{"-r", profile.RepositoryHint, "backup"}
		if profile.UseFSSnapshot {
			args = append(args, "--use-fs-snapshot")
		}
		for _, excludePath := range excludePaths {
			args = append(args, "--exclude", excludePath)
		}
		args = append(args, includePaths...)

		executable := "restic"
		if target == "windows" {
			executable = "restic.exe"
		}

		invocations = append(invocations, ResticInvocation{
			Target:     target,
			Executable: executable,
			Args:       args,
		})
	}

	return invocations, nil
}
