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
	CaseName        string
	Target          string
	Profile         string
	UseFSSnapshot   bool
	ResticPath      string
	TempDir         string
	RepoDir         string
	DataDir         string
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

func resolveTargetAndRestic(t *testing.T) (string, string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		resticPath, err := exec.LookPath("restic.exe")
		if err != nil {
			t.Skip("restic.exe not installed")
		}
		return "windows", resticPath
	}

	if os.Getenv("WSL_DISTRO_NAME") == "" {
		t.Skip("WSL-only integration test")
	}

	resticPath, err := exec.LookPath("restic")
	if err != nil {
		t.Skip("restic not installed")
	}
	return "wsl", resticPath
}

func createBaseFixture(t *testing.T, caseName string, target string, resticPath string) integrationFixture {
	t.Helper()

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	dataDir := filepath.Join(tempDir, "data")
	otherDir := filepath.Join(tempDir, "other")

	mkdir := func(path string) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}
	write := func(path string, content string) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	mkdir(filepath.Join(dataDir, "docs"))
	mkdir(filepath.Join(dataDir, "nested", "level1"))
	mkdir(filepath.Join(dataDir, "nested", "level2", "cache"))
	mkdir(filepath.Join(dataDir, "images"))
	mkdir(filepath.Join(dataDir, "logs"))
	mkdir(filepath.Join(dataDir, "links"))
	mkdir(otherDir)

	write(filepath.Join(dataDir, "root.txt"), "root")
	write(filepath.Join(dataDir, "docs", "readme.md"), "readme")
	write(filepath.Join(dataDir, "nested", "level1", "keep.txt"), "keep")
	write(filepath.Join(dataDir, "nested", "level1", "ignore.tmp"), "ignore")
	write(filepath.Join(dataDir, "nested", "level2", "keep-two.txt"), "keep-two")
	write(filepath.Join(dataDir, "nested", "level2", "cache", "cache.bin"), "cache")
	write(filepath.Join(dataDir, "images", "icon.png"), "icon")
	write(filepath.Join(dataDir, "logs", "app.log"), "log")
	write(filepath.Join(dataDir, "top.tmp"), "tmp")
	write(filepath.Join(otherDir, "outside.txt"), "outside")

	profile := "wsl"
	useFSSnapshot := false
	if target == "windows" {
		profile = "windows"
		useFSSnapshot = true
	}

	return integrationFixture{
		CaseName:      caseName,
		Target:        target,
		Profile:       profile,
		UseFSSnapshot: useFSSnapshot,
		ResticPath:    resticPath,
		TempDir:       tempDir,
		RepoDir:       repoDir,
		DataDir:       dataDir,
		IncludePaths:  []string{dataDir},
		ExcludePatterns: []string{
			filepath.Join(dataDir, "*.tmp"),
			filepath.Join(dataDir, "nested", "level1", "*.tmp"),
			filepath.Join(dataDir, "nested", "level2", "cache", "*"),
			filepath.Join(dataDir, "logs", "*.log"),
		},
		BeforeList: []string{
			"data/root.txt",
			"data/docs/readme.md",
			"data/nested/level1/keep.txt",
			"data/nested/level1/ignore.tmp",
			"data/nested/level2/keep-two.txt",
			"data/nested/level2/cache/cache.bin",
			"data/images/icon.png",
			"data/logs/app.log",
			"data/top.tmp",
			"other/outside.txt",
		},
		ExpectedAfter: []string{
			"data/root.txt",
			"data/docs/readme.md",
			"data/nested/level1/keep.txt",
			"data/nested/level2/keep-two.txt",
			"data/images/icon.png",
		},
		ExpectedAbsent: []string{
			"data/top.tmp",
			"data/nested/level1/ignore.tmp",
			"data/nested/level2/cache/cache.bin",
			"data/logs/app.log",
			"outside.txt",
		},
	}
}

func addFileSymlinkAndHardlink(t *testing.T, fixture *integrationFixture) {
	t.Helper()
	originalPath := filepath.Join(fixture.DataDir, "nested", "level1", "keep.txt")
	symlinkPath := filepath.Join(fixture.DataDir, "links", "symlink-to-keep.txt")
	hardLinkPath := filepath.Join(fixture.DataDir, "links", "hardlink-to-keep.txt")

	if err := os.Symlink(originalPath, symlinkPath); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if err := os.Link(originalPath, hardLinkPath); err != nil {
		t.Skipf("hardlink creation unavailable: %v", err)
	}

	fixture.BeforeList = append(fixture.BeforeList, "data/links/symlink-to-keep.txt", "data/links/hardlink-to-keep.txt")
	fixture.ExpectedAfter = append(fixture.ExpectedAfter, "data/links/symlink-to-keep.txt", "data/links/hardlink-to-keep.txt")
}

func addDirectorySymlink(t *testing.T, fixture *integrationFixture) {
	t.Helper()
	targetDir := filepath.Join(fixture.DataDir, "nested", "level2")
	linkPath := filepath.Join(fixture.DataDir, "links", "dir-symlink-to-level2")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Skipf("directory symlink creation unavailable: %v", err)
	}
	fixture.BeforeList = append(fixture.BeforeList, "data/links/dir-symlink-to-level2")
	fixture.ExpectedAfter = append(fixture.ExpectedAfter, "data/links/dir-symlink-to-level2")
}

func addBrokenSymlink(t *testing.T, fixture *integrationFixture) {
	t.Helper()
	brokenTarget := filepath.Join(fixture.DataDir, "missing-target.txt")
	linkPath := filepath.Join(fixture.DataDir, "links", "broken-symlink.txt")
	if err := os.Symlink(brokenTarget, linkPath); err != nil {
		t.Skipf("broken symlink creation unavailable: %v", err)
	}
	fixture.BeforeList = append(fixture.BeforeList, "data/links/broken-symlink.txt")
	fixture.ExpectedAfter = append(fixture.ExpectedAfter, "data/links/broken-symlink.txt")
}

func addSymlinkLoop(t *testing.T, fixture *integrationFixture) {
	t.Helper()
	loopA := filepath.Join(fixture.DataDir, "links", "loop-a")
	loopB := filepath.Join(fixture.DataDir, "links", "loop-b")
	if err := os.Symlink(loopB, loopA); err != nil {
		t.Skipf("loop symlink A creation unavailable: %v", err)
	}
	if err := os.Symlink(loopA, loopB); err != nil {
		t.Skipf("loop symlink B creation unavailable: %v", err)
	}
	fixture.BeforeList = append(fixture.BeforeList, "data/links/loop-a", "data/links/loop-b")
	fixture.ExpectedAfter = append(fixture.ExpectedAfter, "data/links/loop-a", "data/links/loop-b")
}

func addWindowsJunction(t *testing.T, fixture *integrationFixture) {
	t.Helper()
	if fixture.Target != "windows" {
		t.Skip("windows junction case is windows-only")
	}
	junctionPath := filepath.Join(fixture.DataDir, "links", "junction-to-level2")
	junctionTarget := filepath.Join(fixture.DataDir, "nested", "level2")
	junctionCmd := exec.Command("cmd", "/C", "mklink", "/J", junctionPath, junctionTarget)
	if output, err := junctionCmd.CombinedOutput(); err != nil {
		t.Skipf("junction creation unavailable: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	fixture.BeforeList = append(fixture.BeforeList, "data/links/junction-to-level2")
	fixture.ExpectedAfter = append(fixture.ExpectedAfter, "data/links/junction-to-level2")
}

func runFixtureCase(t *testing.T, fixture integrationFixture, allowPause bool) {
	t.Helper()

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
	t.Logf("case: %s", fixture.CaseName)
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

	if allowPause && os.Getenv("BACKUP_ITEST_PAUSE") == "1" {
		fmt.Printf("integration pause enabled\ncase: %s\nrepo: %s\ndata: %s\n", fixture.CaseName, fixture.RepoDir, fixture.DataDir)
		fmt.Println("manual inspect commands:")
		fmt.Printf("  RESTIC_PASSWORD=integration-test-password %s -r %q snapshots\n", fixture.ResticPath, fixture.RepoDir)
		fmt.Printf("  RESTIC_PASSWORD=integration-test-password %s -r %q ls latest\n", fixture.ResticPath, fixture.RepoDir)
		fmt.Println("press Enter to continue cleanup...")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
}

func runCase(t *testing.T, caseName string, mutate func(t *testing.T, fixture *integrationFixture), allowPause bool) {
	t.Helper()
	target, resticPath := resolveTargetAndRestic(t)
	fixture := createBaseFixture(t, caseName, target, resticPath)
	if mutate != nil {
		mutate(t, &fixture)
	}
	runFixtureCase(t, fixture, allowPause)
}

func TestIntegrationCaseIncludeExclude(t *testing.T) {
	runCase(t, "include-exclude", nil, false)
}

func TestIntegrationCaseFileSymlinkAndHardlink(t *testing.T) {
	runCase(t, "file-symlink-hardlink", addFileSymlinkAndHardlink, false)
}

func TestIntegrationCaseDirectorySymlink(t *testing.T) {
	runCase(t, "directory-symlink", addDirectorySymlink, false)
}

func TestIntegrationCaseBrokenSymlink(t *testing.T) {
	runCase(t, "broken-symlink", addBrokenSymlink, false)
}

func TestIntegrationCaseSymlinkLoop(t *testing.T) {
	runCase(t, "symlink-loop", addSymlinkLoop, false)
}

func TestIntegrationCaseWindowsJunction(t *testing.T) {
	runCase(t, "windows-junction", addWindowsJunction, false)
}

func TestIntegrationManifestAllCases(t *testing.T) {
	t.Run("include-exclude", func(t *testing.T) {
		runCase(t, "include-exclude", nil, os.Getenv("BACKUP_ITEST_PAUSE") == "1")
	})
	t.Run("file-symlink-hardlink", func(t *testing.T) {
		runCase(t, "file-symlink-hardlink", addFileSymlinkAndHardlink, false)
	})
	t.Run("directory-symlink", func(t *testing.T) {
		runCase(t, "directory-symlink", addDirectorySymlink, false)
	})
	t.Run("broken-symlink", func(t *testing.T) {
		runCase(t, "broken-symlink", addBrokenSymlink, false)
	})
	t.Run("symlink-loop", func(t *testing.T) {
		runCase(t, "symlink-loop", addSymlinkLoop, false)
	})
	t.Run("windows-junction", func(t *testing.T) {
		runCase(t, "windows-junction", addWindowsJunction, false)
	})
}
