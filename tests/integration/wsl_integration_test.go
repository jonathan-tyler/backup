//go:build integration && linux

package integration

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	backup "wsl-backup-cli/src"
)

func TestIntegrationWSLDailyResticFlow(t *testing.T) {
	if os.Getenv("WSL_DISTRO_NAME") == "" {
		t.Skip("WSL-only integration test")
	}
	if _, err := exec.LookPath("restic"); err != nil {
		t.Skip("restic not installed")
	}

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	dataDir := filepath.Join(tempDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write sample file: %v", err)
	}

	_ = os.Setenv("RESTIC_PASSWORD", "integration-test-password")
	defer func() { _ = os.Unsetenv("RESTIC_PASSWORD") }()

	_, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     "wsl",
		Executable: "restic",
		Args:       []string{"-r", repoDir, "init"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic init failed: %v", err)
	}

	backupBinary := os.Getenv("BACKUP_BINARY")
	if backupBinary != "" {
		configPath := filepath.Join(tempDir, "config.yaml")
		configContent := fmt.Sprintf("profiles:\n  wsl:\n    repository: %q\n    include:\n      - %q\n    exclude: []\n    use_fs_snapshot: false\n", repoDir, dataDir)
		if writeErr := os.WriteFile(configPath, []byte(configContent), 0o644); writeErr != nil {
			t.Fatalf("write config file: %v", writeErr)
		}

		previousConfig := os.Getenv("BACKUP_CONFIG")
		_ = os.Setenv("BACKUP_CONFIG", configPath)
		defer func() {
			if previousConfig == "" {
				_ = os.Unsetenv("BACKUP_CONFIG")
				return
			}
			_ = os.Setenv("BACKUP_CONFIG", previousConfig)
		}()

		command := exec.Command(backupBinary, "run", "daily")
		commandOutput, commandErr := command.CombinedOutput()
		if commandErr != nil {
			t.Fatalf("backup binary run daily failed: %v (%s)", commandErr, strings.TrimSpace(string(commandOutput)))
		}
	} else {
		_, err = backup.ExecuteResticInvocations([]backup.ResticInvocation{{
			Target:     "wsl",
			Executable: "restic",
			Args:       []string{"-r", repoDir, "backup", dataDir},
		}}, backup.SystemExecutor{})
		if err != nil {
			t.Fatalf("restic backup failed: %v", err)
		}
	}

	results, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     "wsl",
		Executable: "restic",
		Args:       []string{"-r", repoDir, "snapshots"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic snapshots failed: %v", err)
	}
	if len(results) != 1 || results[0].Output == "" {
		t.Fatalf("expected snapshot output, got %#v", results)
	}

	if os.Getenv("BACKUP_ITEST_PAUSE") == "1" {
		fmt.Printf("integration pause enabled\nrepo: %s\ndata: %s\n", repoDir, dataDir)
		fmt.Println("\nInspect source files:")
		fmt.Printf("  find %q -maxdepth 3 -type f -print\n", dataDir)
		fmt.Printf("  find %q -maxdepth 3 -type f -print -exec cat {} \\;\n", dataDir)
		fmt.Println("\nInspect restic repository:")
		fmt.Printf("  RESTIC_PASSWORD=integration-test-password restic -r %q snapshots\n", repoDir)
		fmt.Printf("  RESTIC_PASSWORD=integration-test-password restic -r %q ls latest\n", repoDir)
		fmt.Printf("  RESTIC_PASSWORD=integration-test-password restic -r %q stats\n", repoDir)
		fmt.Println("\nPress Enter to continue cleanup...")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
}

func TestIntegrationWSLLinksFlow(t *testing.T) {
	if os.Getenv("WSL_DISTRO_NAME") == "" {
		t.Skip("WSL-only integration test")
	}
	if _, err := exec.LookPath("restic"); err != nil {
		t.Skip("restic not installed")
	}

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	dataDir := filepath.Join(tempDir, "data")
	restoreDir := filepath.Join(tempDir, "restore")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.MkdirAll(restoreDir, 0o755); err != nil {
		t.Fatalf("mkdir restore dir: %v", err)
	}

	originalPath := filepath.Join(dataDir, "original.txt")
	symlinkPath := filepath.Join(dataDir, "symlink-to-original.txt")
	hardLinkPath := filepath.Join(dataDir, "hardlink-to-original.txt")

	if err := os.WriteFile(originalPath, []byte("hello-links"), 0o644); err != nil {
		t.Fatalf("write original file: %v", err)
	}
	if err := os.Symlink(originalPath, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	if err := os.Link(originalPath, hardLinkPath); err != nil {
		t.Fatalf("create hardlink: %v", err)
	}

	_ = os.Setenv("RESTIC_PASSWORD", "integration-test-password")
	defer func() { _ = os.Unsetenv("RESTIC_PASSWORD") }()

	_, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{ {
		Target:     "wsl",
		Executable: "restic",
		Args:       []string{"-r", repoDir, "init"},
	} }, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic init failed: %v", err)
	}

	_, err = backup.ExecuteResticInvocations([]backup.ResticInvocation{ {
		Target:     "wsl",
		Executable: "restic",
		Args:       []string{"-r", repoDir, "backup", dataDir},
	} }, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic backup failed: %v", err)
	}

	lsResults, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{ {
		Target:     "wsl",
		Executable: "restic",
		Args:       []string{"-r", repoDir, "ls", "latest"},
	} }, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic ls latest failed: %v", err)
	}
	if !strings.Contains(lsResults[0].Output, "symlink-to-original.txt") {
		t.Fatalf("snapshot listing missing symlink: %s", lsResults[0].Output)
	}
	if !strings.Contains(lsResults[0].Output, "hardlink-to-original.txt") {
		t.Fatalf("snapshot listing missing hardlink: %s", lsResults[0].Output)
	}

	_, err = backup.ExecuteResticInvocations([]backup.ResticInvocation{ {
		Target:     "wsl",
		Executable: "restic",
		Args:       []string{"-r", repoDir, "restore", "latest", "--target", restoreDir},
	} }, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic restore failed: %v", err)
	}

	restoredOriginal := filepath.Join(restoreDir, strings.TrimPrefix(originalPath, string(os.PathSeparator)))
	restoredSymlink := filepath.Join(restoreDir, strings.TrimPrefix(symlinkPath, string(os.PathSeparator)))
	restoredHardLink := filepath.Join(restoreDir, strings.TrimPrefix(hardLinkPath, string(os.PathSeparator)))

	symlinkInfo, err := os.Lstat(restoredSymlink)
	if err != nil {
		t.Fatalf("lstat restored symlink: %v", err)
	}
	if symlinkInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected restored symlink, got mode %v", symlinkInfo.Mode())
	}

	originalInfo, err := os.Stat(restoredOriginal)
	if err != nil {
		t.Fatalf("stat restored original: %v", err)
	}
	hardLinkInfo, err := os.Stat(restoredHardLink)
	if err != nil {
		t.Fatalf("stat restored hardlink: %v", err)
	}
	if !os.SameFile(originalInfo, hardLinkInfo) {
		t.Fatal("restored hardlink does not reference same file")
	}
}
