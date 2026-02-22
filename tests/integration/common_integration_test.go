//go:build integration

package integration

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	backup "wsl-backup-cli/src"

	"gopkg.in/yaml.v3"
)

type integrationFixture struct {
	Target          string
	Profile         string
	UseFSSnapshot   bool
	ResticPath      string
	TempDir         string
	RepoDir         string
	DataDir         string
	OtherDir        string
	IncludePaths    []string
	ExcludePatterns []string
	BeforeList      []string
	ExpectedAfter   []string
	ExpectedAbsent  []string
}

type testConfigFile struct {
	Profiles map[string]testConfigProfile `yaml:"profiles"`
}

type testConfigProfile struct {
	Repository    string   `yaml:"repository"`
	Include       []string `yaml:"include"`
	Exclude       []string `yaml:"exclude"`
	UseFSSnapshot bool     `yaml:"use_fs_snapshot"`
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "  (none)"
	}
	var builder strings.Builder
	for _, item := range items {
		builder.WriteString("  - ")
		builder.WriteString(item)
		builder.WriteString("\n")
	}
	return strings.TrimSuffix(builder.String(), "\n")
}

func listContainsPath(listing string, relPath string) bool {
	normalizedListing := strings.ToLower(strings.ReplaceAll(listing, "\\", "/"))
	normalizedRelPath := strings.ToLower(strings.ReplaceAll(relPath, "\\", "/"))
	return strings.Contains(normalizedListing, normalizedRelPath)
}

func writeTestConfig(path string, profile string, repository string, includes []string, excludes []string, useFSSnapshot bool) error {
	config := testConfigFile{
		Profiles: map[string]testConfigProfile{
			profile: {
				Repository:    repository,
				Include:       includes,
				Exclude:       excludes,
				UseFSSnapshot: useFSSnapshot,
			},
		},
	}

	content, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, content, 0o644)
}

func createCommonManifest(t *testing.T, target string, resticPath string) integrationFixture {
	t.Helper()

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	dataDir := filepath.Join(tempDir, "data")
	otherDir := filepath.Join(tempDir, "other")
	linksDir := filepath.Join(dataDir, "links")

	if err := os.MkdirAll(filepath.Join(dataDir, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "nested", "level1"), 0o755); err != nil {
		t.Fatalf("mkdir level1: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "nested", "level2", "cache"), 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "images"), 0o755); err != nil {
		t.Fatalf("mkdir images: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "logs"), 0o755); err != nil {
		t.Fatalf("mkdir logs: %v", err)
	}
	if err := os.MkdirAll(linksDir, 0o755); err != nil {
		t.Fatalf("mkdir links: %v", err)
	}
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatalf("mkdir other dir: %v", err)
	}

	writeFile := func(path string, content string) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write file %s: %v", path, err)
		}
	}

	writeFile(filepath.Join(dataDir, "root.txt"), "root")
	writeFile(filepath.Join(dataDir, "docs", "readme.md"), "readme")
	writeFile(filepath.Join(dataDir, "nested", "level1", "keep.txt"), "keep")
	writeFile(filepath.Join(dataDir, "nested", "level1", "ignore.tmp"), "ignore")
	writeFile(filepath.Join(dataDir, "nested", "level2", "keep-two.txt"), "keep-two")
	writeFile(filepath.Join(dataDir, "nested", "level2", "cache", "cache.bin"), "cache")
	writeFile(filepath.Join(dataDir, "images", "icon.png"), "icon")
	writeFile(filepath.Join(dataDir, "logs", "app.log"), "log")
	writeFile(filepath.Join(dataDir, "top.tmp"), "tmp")
	writeFile(filepath.Join(otherDir, "outside.txt"), "outside")

	originalPath := filepath.Join(dataDir, "nested", "level1", "keep.txt")
	symlinkPath := filepath.Join(linksDir, "symlink-to-keep.txt")
	hardLinkPath := filepath.Join(linksDir, "hardlink-to-keep.txt")

	if err := os.Symlink(originalPath, symlinkPath); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if err := os.Link(originalPath, hardLinkPath); err != nil {
		t.Skipf("hardlink creation unavailable: %v", err)
	}

	expectedAfter := []string{
		"data/root.txt",
		"data/docs/readme.md",
		"data/nested/level1/keep.txt",
		"data/nested/level2/keep-two.txt",
		"data/images/icon.png",
		"data/links/symlink-to-keep.txt",
		"data/links/hardlink-to-keep.txt",
	}
	beforeList := []string{
		"data/root.txt",
		"data/docs/readme.md",
		"data/nested/level1/keep.txt",
		"data/nested/level1/ignore.tmp",
		"data/nested/level2/keep-two.txt",
		"data/nested/level2/cache/cache.bin",
		"data/images/icon.png",
		"data/logs/app.log",
		"data/top.tmp",
		"data/links/symlink-to-keep.txt",
		"data/links/hardlink-to-keep.txt",
		"other/outside.txt",
	}
	expectedAbsent := []string{
		"data/top.tmp",
		"data/nested/level1/ignore.tmp",
		"data/nested/level2/cache/cache.bin",
		"data/logs/app.log",
		"outside.txt",
	}

	if target == "windows" {
		junctionPath := filepath.Join(linksDir, "junction-to-level2")
		junctionTarget := filepath.Join(dataDir, "nested", "level2")
		junctionCmd := exec.Command("cmd", "/C", "mklink", "/J", junctionPath, junctionTarget)
		if output, err := junctionCmd.CombinedOutput(); err != nil {
			t.Skipf("junction creation unavailable: %v (%s)", err, strings.TrimSpace(string(output)))
		}
		expectedAfter = append(expectedAfter, "data/links/junction-to-level2")
		beforeList = append(beforeList, "data/links/junction-to-level2")
	}

	profile := "wsl"
	useFSSnapshot := false
	if target == "windows" {
		profile = "windows"
		useFSSnapshot = true
	}

	return integrationFixture{
		Target:        target,
		Profile:       profile,
		UseFSSnapshot: useFSSnapshot,
		ResticPath:    resticPath,
		TempDir:       tempDir,
		RepoDir:       repoDir,
		DataDir:       dataDir,
		OtherDir:      otherDir,
		IncludePaths:  []string{dataDir},
		ExcludePatterns: []string{
			filepath.Join(dataDir, "*.tmp"),
			filepath.Join(dataDir, "nested", "level1", "*.tmp"),
			filepath.Join(dataDir, "nested", "level2", "cache", "*"),
			filepath.Join(dataDir, "logs", "*.log"),
		},
		BeforeList:     beforeList,
		ExpectedAfter:  expectedAfter,
		ExpectedAbsent: expectedAbsent,
	}
}

func runDailyManifestFlow(t *testing.T, target string, resticPath string) {
	t.Helper()

	fixture := createCommonManifest(t, target, resticPath)

	_ = os.Setenv("RESTIC_PASSWORD", "integration-test-password")
	defer func() { _ = os.Unsetenv("RESTIC_PASSWORD") }()

	_, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     fixture.Target,
		Executable: fixture.ResticPath,
		Args:       []string{"-r", fixture.RepoDir, "init"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic init failed: %v", err)
	}

	backupBinary := os.Getenv("BACKUP_BINARY")
	if backupBinary != "" {
		configPath := filepath.Join(fixture.TempDir, "config.yaml")
		if writeErr := writeTestConfig(configPath, fixture.Profile, fixture.RepoDir, fixture.IncludePaths, fixture.ExcludePatterns, fixture.UseFSSnapshot); writeErr != nil {
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
		args := []string{"-r", fixture.RepoDir, "backup"}
		for _, pattern := range fixture.ExcludePatterns {
			args = append(args, "--exclude", pattern)
		}
		args = append(args, fixture.IncludePaths...)

		_, err = backup.ExecuteResticInvocations([]backup.ResticInvocation{{
			Target:     fixture.Target,
			Executable: fixture.ResticPath,
			Args:       args,
		}}, backup.SystemExecutor{})
		if err != nil {
			t.Fatalf("restic backup failed: %v", err)
		}
	}

	snapshotsResults, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     fixture.Target,
		Executable: fixture.ResticPath,
		Args:       []string{"-r", fixture.RepoDir, "snapshots"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic snapshots failed: %v", err)
	}

	lsResults, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     fixture.Target,
		Executable: fixture.ResticPath,
		Args:       []string{"-r", fixture.RepoDir, "ls", "latest"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic ls latest failed: %v", err)
	}

	lsOutput := lsResults[0].Output
	t.Logf("before list:\n%s", formatList(fixture.BeforeList))
	t.Logf("expected after list:\n%s", formatList(fixture.ExpectedAfter))
	t.Logf("expected excluded list:\n%s", formatList(fixture.ExpectedAbsent))
	t.Logf("repo snapshots output:\n%s", snapshotsResults[0].Output)
	t.Logf("repo ls latest output:\n%s", lsOutput)

	for _, expected := range fixture.ExpectedAfter {
		if !listContainsPath(lsOutput, expected) {
			t.Fatalf("expected path missing from repository listing: %s", expected)
		}
	}

	for _, excluded := range fixture.ExpectedAbsent {
		if listContainsPath(lsOutput, excluded) {
			t.Fatalf("excluded path found in repository listing: %s", excluded)
		}
	}

	if os.Getenv("BACKUP_ITEST_PAUSE") == "1" {
		fmt.Printf("integration pause enabled\nrepo: %s\ndata: %s\n", fixture.RepoDir, fixture.DataDir)
		fmt.Println("manual inspect commands:")
		fmt.Printf("  RESTIC_PASSWORD=integration-test-password %s -r %q snapshots\n", fixture.ResticPath, fixture.RepoDir)
		fmt.Printf("  RESTIC_PASSWORD=integration-test-password %s -r %q ls latest\n", fixture.ResticPath, fixture.RepoDir)
		fmt.Println("press Enter to continue cleanup...")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
}

func TestIntegrationDailyManifestFlow(t *testing.T) {
	if runtime.GOOS == "windows" {
		resticPath, err := exec.LookPath("restic.exe")
		if err != nil {
			t.Skip("restic.exe not installed")
		}
		runDailyManifestFlow(t, "windows", resticPath)
		return
	}

	if os.Getenv("WSL_DISTRO_NAME") == "" {
		t.Skip("WSL-only integration test")
	}

	resticPath, err := exec.LookPath("restic")
	if err != nil {
		t.Skip("restic not installed")
	}

	runDailyManifestFlow(t, "wsl", resticPath)
}
