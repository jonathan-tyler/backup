package backup

import (
	"fmt"
	"io"
	"strings"
)

type Command struct {
	Name    string
	Cadence string
	Target  string
	Report  string
}

func Usage() string {
	return strings.Join([]string{
		"Usage:",
		"  backup run <daily|weekly|monthly>",
		"  backup report <daily|weekly|monthly> [new|excluded]",
		"  backup restore <target>",
		"  backup help",
		"  backup --help",
		"",
		"Report options:",
		"  new       Show items newly selected for backup (include/exclude diff)",
		"  excluded  Show items currently excluded from backup",
		"",
		"As wsl-sys-cli extension:",
		"  sys backup run <daily|weekly|monthly>",
		"  sys backup report <daily|weekly|monthly> [new|excluded]",
		"  sys backup restore <target>",
		"  sys backup --help",
	}, "\n")
}

func parseReportOption(args []string) (string, error) {
	if len(args) == 0 {
		return "default", nil
	}
	if len(args) > 1 {
		return "", fmt.Errorf("too many report options")
	}

	switch args[0] {
	case "new":
		return "new", nil
	case "excluded":
		return "excluded", nil
	default:
		return "", fmt.Errorf("unknown report option: %s", args[0])
	}
}

func isValidCadence(cadence string) bool {
	switch cadence {
	case "daily", "weekly", "monthly":
		return true
	default:
		return false
	}
}

func ParseArgs(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{Name: "help"}, nil
	}

	command := args[0]
	switch command {
	case "help", "-h", "--help":
		return Command{Name: "help"}, nil
	case "run", "report":
		if len(args) < 2 {
			return Command{}, fmt.Errorf("missing cadence")
		}
		cadence := args[1]
		if !isValidCadence(cadence) {
			return Command{}, fmt.Errorf("invalid cadence: %s", cadence)
		}

		if command == "run" {
			if len(args) > 2 {
				return Command{}, fmt.Errorf("run does not accept options")
			}
			return Command{Name: command, Cadence: cadence}, nil
		}

		reportOption, err := parseReportOption(args[2:])
		if err != nil {
			return Command{}, err
		}
		return Command{Name: command, Cadence: cadence, Report: reportOption}, nil
	case "restore":
		if len(args) < 2 {
			return Command{}, fmt.Errorf("missing target")
		}
		return Command{Name: command, Target: args[1]}, nil
	default:
		return Command{}, fmt.Errorf("unknown command: %s", command)
	}
}

func Run(command Command, executor Executor) (string, error) {
	switch command.Name {
	case "help":
		return Usage(), nil
	case "run":
		runtime := DetectRuntime()
		plan, err := BuildRunPlan(command.Cadence, runtime)
		if err != nil {
			return "", err
		}
		config, err := LoadConfig(runtime)
		if err != nil {
			return "", err
		}
		if err := ValidatePlanConfig(plan, config); err != nil {
			return "", err
		}
		invocations, err := BuildResticInvocations(plan, config)
		if err != nil {
			return "", err
		}
		if command.Cadence == "daily" && config.Exists {
			results, err := ExecuteResticInvocations(invocations, executor)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("daily backup run executed for targets=%s (steps=%d).", strings.Join(plan.Targets, ","), len(results)), nil
		}
		return fmt.Sprintf("scaffold only: %s backup run targets=%s (execution not implemented yet).", plan.Cadence, strings.Join(plan.Targets, ",")), nil
	case "report":
		switch command.Report {
		case "new":
			return fmt.Sprintf("%s backup report (new) is not implemented yet.", command.Cadence), nil
		case "excluded":
			return fmt.Sprintf("%s backup report (excluded) is not implemented yet.", command.Cadence), nil
		default:
			return fmt.Sprintf("%s backup report is not implemented yet.", command.Cadence), nil
		}
	case "restore":
		return fmt.Sprintf("restore is not implemented yet (target: %s).", command.Target), nil
	default:
		return "", fmt.Errorf("unknown command: %s", command.Name)
	}
}

func RunCLI(args []string, stdout io.Writer, stderr io.Writer, executor Executor) int {
	command, err := ParseArgs(args)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	output, err := Run(command, executor)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, output)
	return 0
}
