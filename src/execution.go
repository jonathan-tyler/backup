package backup

import (
	"fmt"
	"os/exec"
	"strings"
)

type Executor interface {
	Run(name string, args ...string) (string, error)
}

type SystemExecutor struct{}

func (executor SystemExecutor) Run(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

type ExecutionResult struct {
	Target string
	Output string
}

func ExecuteResticInvocations(invocations []ResticInvocation, executor Executor) ([]ExecutionResult, error) {
	results := make([]ExecutionResult, 0, len(invocations))
	for _, invocation := range invocations {
		output, err := executor.Run(invocation.Executable, invocation.Args...)
		if err != nil {
			return nil, fmt.Errorf("%s invocation failed: %w", invocation.Target, err)
		}
		results = append(results, ExecutionResult{Target: invocation.Target, Output: output})
	}
	return results, nil
}
