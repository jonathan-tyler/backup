package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Command struct {
	Name    string
	Cadence string
	Target  string
}

func usage() string {
	return strings.Join([]string{
		"Usage:",
		"  wsl-backup-cli run <daily|weekly|monthly>",
		"  wsl-backup-cli report <daily|weekly|monthly>",
		"  wsl-backup-cli restore <target>",
		"  wsl-backup-cli help",
		"  wsl-backup-cli --help",
	}, "\n")
}

func isValidCadence(cadence string) bool {
	switch cadence {
	case "daily", "weekly", "monthly":
		return true
	default:
		return false
	}
}

func parseArgs(args []string) (Command, error) {
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
		return Command{Name: command, Cadence: cadence}, nil
	case "restore":
		if len(args) < 2 {
			return Command{}, fmt.Errorf("missing target")
		}
		return Command{Name: command, Target: args[1]}, nil
	default:
		return Command{}, fmt.Errorf("Unknown command: %s", command)
	}
}

func run(command Command) (string, error) {
	switch command.Name {
	case "help":
		return usage(), nil
	case "run":
		return fmt.Sprintf("%s backup run is not implemented yet.", command.Cadence), nil
	case "report":
		return fmt.Sprintf("%s backup report is not implemented yet.", command.Cadence), nil
	case "restore":
		return fmt.Sprintf("restore is not implemented yet (target: %s).", command.Target), nil
	default:
		return "", fmt.Errorf("Unknown command: %s", command.Name)
	}
}

func runCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	command, err := parseArgs(args)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	output, err := run(command)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, output)
	return 0
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}
