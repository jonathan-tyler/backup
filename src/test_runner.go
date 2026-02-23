package backup

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunCrossPlatformManualTests() error {
	rootDir, err := resolveRepoRootForManualTests()
	if err != nil {
		return err
	}

	linuxScript := filepath.Join(rootDir, "tests", "manual", "run_manual_integration_tests.sh")
	if err := runInteractiveCommand(rootDir, "bash", linuxScript); err != nil {
		return fmt.Errorf("linux manual integration tests failed: %w", err)
	}

	if err := pauseForPhaseTransition(); err != nil {
		return err
	}

	windowsScript := filepath.Join(rootDir, "tests", "manual", "run_manual_integration_tests.ps1")
	windowsScriptPath, err := convertToWindowsPath(windowsScript)
	if err != nil {
		return fmt.Errorf("resolve windows script path: %w", err)
	}

	if _, err := exec.LookPath("powershell.exe"); err != nil {
		return fmt.Errorf("powershell.exe not found; install/enable WSL Windows interop")
	}

	if err := runInteractiveCommand(rootDir, "powershell.exe", "-ExecutionPolicy", "Bypass", "-File", windowsScriptPath); err != nil {
		return fmt.Errorf("windows manual integration tests failed: %w", err)
	}

	return nil
}

func resolveRepoRootForManualTests() (string, error) {
	if override := strings.TrimSpace(os.Getenv("BACKUP_REPO_ROOT")); override != "" {
		if err := ensureManualScriptsExist(override); err != nil {
			return "", err
		}
		return override, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	if err := ensureManualScriptsExist(cwd); err != nil {
		return "", fmt.Errorf("manual test scripts not found in current directory; run from repo root or set BACKUP_REPO_ROOT: %w", err)
	}
	return cwd, nil
}

func ensureManualScriptsExist(rootDir string) error {
	linuxScript := filepath.Join(rootDir, "tests", "manual", "run_manual_integration_tests.sh")
	windowsScript := filepath.Join(rootDir, "tests", "manual", "run_manual_integration_tests.ps1")

	if _, err := os.Stat(linuxScript); err != nil {
		return err
	}
	if _, err := os.Stat(windowsScript); err != nil {
		return err
	}
	return nil
}

func runInteractiveCommand(dir string, name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Dir = dir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	command.Env = os.Environ()
	return command.Run()
}

func convertToWindowsPath(path string) (string, error) {
	command := exec.Command("wslpath", "-w", path)
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("wslpath -w failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func pauseForPhaseTransition() error {
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Linux phase complete. Press Enter to continue to Windows phase...")
	reader := bufio.NewReader(os.Stdin)
	_, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read pause input: %w", err)
	}
	return nil
}
