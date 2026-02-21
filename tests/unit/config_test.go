package unit

import (
	"os"
	"path/filepath"
	"testing"

	backup "wsl-backup-cli/src"
)

func TestResolveConfigPathUsesOverride(t *testing.T) {
	t.Parallel()

	previous := os.Getenv("BACKUP_CONFIG")
	t.Cleanup(func() {
		if previous == "" {
			_ = os.Unsetenv("BACKUP_CONFIG")
			return
		}
		_ = os.Setenv("BACKUP_CONFIG", previous)
	})
	_ = os.Setenv("BACKUP_CONFIG", "/tmp/custom-backup-config.yaml")

	path, err := backup.ResolveConfigPath(backup.RuntimeWSL)
	if err != nil {
		t.Fatalf("ResolveConfigPath returned error: %v", err)
	}
	if path != "/tmp/custom-backup-config.yaml" {
		t.Fatalf("unexpected config path: %q", path)
	}
}

func TestLoadConfigParsesExistingYAMLFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	content := []byte("profiles:\n  wsl:\n    repository: /repo/wsl\n    include:\n      - /home/test\n    exclude:\n      - /home/test/.cache\n    use_fs_snapshot: false\n")
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previous := os.Getenv("BACKUP_CONFIG")
	t.Cleanup(func() {
		if previous == "" {
			_ = os.Unsetenv("BACKUP_CONFIG")
			return
		}
		_ = os.Setenv("BACKUP_CONFIG", previous)
	})
	_ = os.Setenv("BACKUP_CONFIG", configPath)

	config, err := backup.LoadConfig(backup.RuntimeWSL)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if !config.Exists {
		t.Fatal("expected config.Exists true")
	}
	if config.Profiles["wsl"].RepositoryHint != "/repo/wsl" {
		t.Fatalf("unexpected repository: %q", config.Profiles["wsl"].RepositoryHint)
	}
}
