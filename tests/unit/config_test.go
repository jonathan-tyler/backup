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
	if len(config.Profiles["wsl"].IncludeByCadence.Daily) != 1 || config.Profiles["wsl"].IncludeByCadence.Daily[0] != "/home/test" {
		t.Fatalf("unexpected daily include paths: %#v", config.Profiles["wsl"].IncludeByCadence.Daily)
	}
	if len(config.Profiles["wsl"].IncludeByCadence.Weekly) != 1 || config.Profiles["wsl"].IncludeByCadence.Weekly[0] != "/home/test" {
		t.Fatalf("unexpected weekly include paths: %#v", config.Profiles["wsl"].IncludeByCadence.Weekly)
	}
	if len(config.Profiles["wsl"].IncludeByCadence.Monthly) != 1 || config.Profiles["wsl"].IncludeByCadence.Monthly[0] != "/home/test" {
		t.Fatalf("unexpected monthly include paths: %#v", config.Profiles["wsl"].IncludeByCadence.Monthly)
	}
}

func TestLoadConfigParsesCadenceIncludeExcludeMaps(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	content := []byte("profiles:\n  wsl:\n    repository: /repo/wsl\n    include:\n      daily:\n        - /home/test/daily\n      weekly:\n        - /home/test/daily\n        - /home/test/weekly\n      monthly:\n        - /home/test/daily\n        - /home/test/weekly\n        - /home/test/monthly\n    exclude:\n      daily:\n        - /home/test/.cache\n      weekly:\n        - /home/test/.cache\n      monthly:\n        - /home/test/.cache\n    use_fs_snapshot: false\n")
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

	profile := config.Profiles["wsl"]
	if len(profile.IncludeByCadence.Daily) != 1 || profile.IncludeByCadence.Daily[0] != "/home/test/daily" {
		t.Fatalf("unexpected daily include paths: %#v", profile.IncludeByCadence.Daily)
	}
	if len(profile.IncludeByCadence.Weekly) != 2 {
		t.Fatalf("unexpected weekly include paths: %#v", profile.IncludeByCadence.Weekly)
	}
	if len(profile.IncludeByCadence.Monthly) != 3 {
		t.Fatalf("unexpected monthly include paths: %#v", profile.IncludeByCadence.Monthly)
	}
	if len(profile.ExcludeByCadence.Daily) != 1 || profile.ExcludeByCadence.Daily[0] != "/home/test/.cache" {
		t.Fatalf("unexpected daily exclude paths: %#v", profile.ExcludeByCadence.Daily)
	}
}

func TestLoadConfigParsesCadencePathsFromTextFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	rulesDir := filepath.Join(tempDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules dir: %v", err)
	}

	write := func(path string, content string) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	write(filepath.Join(rulesDir, "wsl.include.daily.txt"), "# daily includes\n/home/test/daily\n")
	write(filepath.Join(rulesDir, "wsl.include.weekly.txt"), "/home/test/daily\n/home/test/weekly\n")
	write(filepath.Join(rulesDir, "wsl.include.monthly.txt"), "/home/test/daily\n/home/test/weekly\n/home/test/monthly\n")
	write(filepath.Join(rulesDir, "wsl.exclude.daily.txt"), "/home/test/.cache\n")
	write(filepath.Join(rulesDir, "wsl.exclude.weekly.txt"), "/home/test/.cache\n")
	write(filepath.Join(rulesDir, "wsl.exclude.monthly.txt"), "/home/test/.cache\n")

	configPath := filepath.Join(tempDir, "config.yaml")
	content := []byte("profiles:\n  wsl:\n    repository: /repo/wsl\n    include_files:\n      daily: rules/wsl.include.daily.txt\n      weekly: rules/wsl.include.weekly.txt\n      monthly: rules/wsl.include.monthly.txt\n    exclude_files:\n      daily: rules/wsl.exclude.daily.txt\n      weekly: rules/wsl.exclude.weekly.txt\n      monthly: rules/wsl.exclude.monthly.txt\n    use_fs_snapshot: false\n")
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

	profile := config.Profiles["wsl"]
	if len(profile.IncludeByCadence.Daily) != 1 || profile.IncludeByCadence.Daily[0] != "/home/test/daily" {
		t.Fatalf("unexpected daily include paths: %#v", profile.IncludeByCadence.Daily)
	}
	if len(profile.IncludeByCadence.Weekly) != 2 {
		t.Fatalf("unexpected weekly include paths: %#v", profile.IncludeByCadence.Weekly)
	}
	if len(profile.IncludeByCadence.Monthly) != 3 {
		t.Fatalf("unexpected monthly include paths: %#v", profile.IncludeByCadence.Monthly)
	}
	if len(profile.ExcludeByCadence.Daily) != 1 || profile.ExcludeByCadence.Daily[0] != "/home/test/.cache" {
		t.Fatalf("unexpected daily exclude paths: %#v", profile.ExcludeByCadence.Daily)
	}
}

func TestLoadConfigAutoDiscoversDefaultRuleFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	rulesDir := filepath.Join(tempDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules dir: %v", err)
	}

	write := func(path string, content string) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	write(filepath.Join(rulesDir, "wsl.include.daily.txt"), "/home/test/daily\n")
	write(filepath.Join(rulesDir, "wsl.include.weekly.txt"), "/home/test/daily\n/home/test/weekly\n")
	write(filepath.Join(rulesDir, "wsl.include.monthly.txt"), "/home/test/daily\n/home/test/weekly\n/home/test/monthly\n")
	write(filepath.Join(rulesDir, "wsl.exclude.daily.txt"), "/home/test/.cache\n")
	write(filepath.Join(rulesDir, "wsl.exclude.weekly.txt"), "/home/test/.cache\n")
	write(filepath.Join(rulesDir, "wsl.exclude.monthly.txt"), "/home/test/.cache\n")

	configPath := filepath.Join(tempDir, "config.yaml")
	content := []byte("profiles:\n  wsl:\n    repository: /repo/wsl\n    use_fs_snapshot: false\n")
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

	profile := config.Profiles["wsl"]
	if len(profile.IncludeByCadence.Weekly) != 2 {
		t.Fatalf("unexpected weekly include paths: %#v", profile.IncludeByCadence.Weekly)
	}
	if len(profile.ExcludeByCadence.Daily) != 1 || profile.ExcludeByCadence.Daily[0] != "/home/test/.cache" {
		t.Fatalf("unexpected daily exclude paths: %#v", profile.ExcludeByCadence.Daily)
	}
}
