package backup

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
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
	results := make([]ExecutionResult, len(invocations))
	errors := make([]error, len(invocations))

	var waitGroup sync.WaitGroup
	for invocationIndex := range invocations {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			invocation := invocations[index]
			output, err := executor.Run(invocation.Executable, invocation.Args...)
			if err != nil {
				errors[index] = err
				return
			}
			results[index] = ExecutionResult{Target: invocation.Target, Output: output}
		}(invocationIndex)
	}

	waitGroup.Wait()

	for invocationIndex := range errors {
		if errors[invocationIndex] != nil {
			return nil, fmt.Errorf("%s invocation failed: %w", invocations[invocationIndex].Target, errors[invocationIndex])
		}
	}

	return results, nil
}
