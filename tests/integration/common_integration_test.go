//go:build integration

package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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
	Repository    string              `yaml:"repository"`
	Include       backup.CadencePaths `yaml:"include"`
	Exclude       backup.CadencePaths `yaml:"exclude"`
	UseFSSnapshot bool                `yaml:"use_fs_snapshot"`
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

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	term := strings.TrimSpace(strings.ToLower(os.Getenv("TERM")))
	return term != "" && term != "dumb"
}

func colorize(text string, colorCode string) string {
	if !colorEnabled() {
		return text
	}
	return "\x1b[" + colorCode + "m" + text + "\x1b[0m"
}

func normalizePath(path string) string {
	normalized := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.TrimPrefix(normalized, "./")
}

func pathMatches(repoPath string, expected string) bool {
	normalizedRepoPath := normalizePath(repoPath)
	normalizedExpected := normalizePath(expected)
	return normalizedRepoPath == normalizedExpected || strings.HasSuffix(normalizedRepoPath, "/"+normalizedExpected)
}

type resticLSNode struct {
	StructType string `json:"struct_type"`
	Type       string `json:"type"`
	Path       string `json:"path"`
}

func compactFileCentricListingFromJSON(rawJSON string) string {
	if strings.TrimSpace(rawJSON) == "" {
		return "  (none)"
	}

	seen := make(map[string]struct{})
	paths := make([]string, 0)
	for _, line := range strings.Split(rawJSON, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var node resticLSNode
		if err := json.Unmarshal([]byte(line), &node); err != nil {
			continue
		}
		if node.StructType != "node" {
			continue
		}
		if node.Type != "file" && node.Type != "symlink" {
			continue
		}
		if node.Path == "" {
			continue
		}
		if _, exists := seen[node.Path]; exists {
			continue
		}
		seen[node.Path] = struct{}{}
		paths = append(paths, node.Path)
	}

	if len(paths) == 0 {
		return "  (none)"
	}

	sort.Strings(paths)
	return formatList(paths)
}

func collectFileCentricPathsFromJSON(rawJSON string) []string {
	seen := make(map[string]struct{})
	paths := make([]string, 0)
	for _, line := range strings.Split(rawJSON, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var node resticLSNode
		if err := json.Unmarshal([]byte(line), &node); err != nil {
			continue
		}
		if node.StructType != "node" {
			continue
		}
		if node.Type != "file" && node.Type != "symlink" {
			continue
		}
		if node.Path == "" {
			continue
		}
		if _, exists := seen[node.Path]; exists {
			continue
		}
		seen[node.Path] = struct{}{}
		paths = append(paths, node.Path)
	}
	sort.Strings(paths)
	return paths
}

func containsExpectedPath(paths []string, expected string) bool {
	for _, path := range paths {
		if pathMatches(path, expected) {
			return true
		}
	}
	return false
}

func renderExpectationLines(label string, items []string, paths []string, shouldExist bool, okColorCode string, failColorCode string) (string, int) {
	if len(items) == 0 {
		return fmt.Sprintf("  %s: (none)", label), 0
	}

	matched := 0
	lines := make([]string, 0, len(items)+1)
	for _, item := range items {
		exists := containsExpectedPath(paths, item)
		ok := (shouldExist && exists) || (!shouldExist && !exists)
		if ok {
			matched++
		}

		status := "✗"
		colorCode := failColorCode
		if ok {
			status = "✓"
			colorCode = okColorCode
		}

		line := fmt.Sprintf("  [%s] %s", status, item)
		lines = append(lines, colorize(line, colorCode))
	}
	return strings.Join(lines, "\n"), matched
}

func colorizeList(items []string, colorCode string) string {
	if len(items) == 0 {
		return "  (none)"
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, colorize("  - "+item, colorCode))
	}
	return strings.Join(lines, "\n")
}

func renderBeforeListWithExpectations(before []string, expectedAfter []string, expectedAbsent []string) string {
	if len(before) == 0 {
		return "  (none)"
	}

	expectedIncluded := make(map[string]struct{}, len(expectedAfter))
	for _, path := range expectedAfter {
		expectedIncluded[normalizePath(path)] = struct{}{}
	}

	expectedExcluded := make(map[string]struct{}, len(expectedAbsent))
	for _, path := range expectedAbsent {
		expectedExcluded[normalizePath(path)] = struct{}{}
	}

	lines := make([]string, 0, len(before))
	for _, item := range before {
		normalized := normalizePath(item)
		line := "  - " + item
		if _, ok := expectedIncluded[normalized]; ok {
			lines = append(lines, colorize(line, "38;5;108"))
			continue
		}
		if _, ok := expectedExcluded[normalized]; ok {
			lines = append(lines, colorize(line, "38;5;131"))
			continue
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func writeTestConfig(path string, profile string, repository string, includes []string, excludes []string, useFSSnapshot bool) error {
	config := testConfigFile{
		Profiles: map[string]testConfigProfile{
			profile: {
				Repository: repository,
				Include: backup.CadencePaths{
					Daily:   includes,
					Weekly:  includes,
					Monthly: includes,
				},
				Exclude: backup.CadencePaths{
					Daily:   excludes,
					Weekly:  excludes,
					Monthly: excludes,
				},
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

	if runtime.GOOS != "linux" {
		t.Skip("integration tests currently support Windows and Linux/WSL")
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

func addAllCasesForSingleManifestRun(t *testing.T, fixture *integrationFixture) {
	t.Helper()
	addFileSymlinkAndHardlink(t, fixture)
	addDirectorySymlink(t, fixture)
	addBrokenSymlink(t, fixture)
	addSymlinkLoop(t, fixture)
	if fixture.Target == "windows" {
		addWindowsJunction(t, fixture)
	}
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

	lsJSONResults, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     fixture.Target,
		Executable: fixture.ResticPath,
		Args:       []string{"-r", fixture.RepoDir, "ls", "latest", "--json"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic ls latest --json failed: %v", err)
	}

	lsOutput := lsResults[0].Output
	repoPaths := collectFileCentricPathsFromJSON(lsJSONResults[0].Output)
	compactListing := compactFileCentricListingFromJSON(lsJSONResults[0].Output)
	includedRendered, includedMatched := renderExpectationLines("included", fixture.ExpectedAfter, repoPaths, true, "38;5;108", "38;5;131")
	excludedRendered, excludedMatched := renderExpectationLines("excluded", fixture.ExpectedAbsent, repoPaths, false, "38;5;131", "38;5;108")
	beforeRendered := renderBeforeListWithExpectations(fixture.BeforeList, fixture.ExpectedAfter, fixture.ExpectedAbsent)
	t.Logf("case: %s", fixture.CaseName)
	t.Logf("include rules:\n%s", colorizeList(fixture.IncludePaths, "38;5;108"))
	t.Logf("exclude rules:\n%s", colorizeList(fixture.ExcludePatterns, "38;5;131"))
	t.Logf("before list (green=expected include, red=expected exclude):\n%s", beforeRendered)
	t.Logf("expected excluded list:\n%s", colorizeList(fixture.ExpectedAbsent, "38;5;131"))
	t.Logf("assert include matches: %d/%d\n%s", includedMatched, len(fixture.ExpectedAfter), includedRendered)
	t.Logf("assert exclude matches: %d/%d\n%s", excludedMatched, len(fixture.ExpectedAbsent), excludedRendered)
	t.Logf("repo file/symlink count: %d", len(repoPaths))
	t.Logf("repo snapshots output:\n%s", snapshotsResults[0].Output)
	t.Logf("repo ls latest file-centric output:\n%s", compactListing)

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
		fmt.Printf("  RESTIC_PASSWORD=integration-test-password %s -r %q ls latest --json | jq -r 'select(.struct_type==\"node\" and .type==\"file\") | .path'\n", fixture.ResticPath, fixture.RepoDir)
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

func directoryHasFiles(path string) (bool, error) {
	found := false
	err := filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == path {
			return nil
		}
		if !entry.IsDir() {
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil && err != filepath.SkipDir {
		return false, err
	}
	return found, nil
}

func TestIntegrationRestoreLatest(t *testing.T) {
	target, resticPath := resolveTargetAndRestic(t)
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	dataDir := filepath.Join(tempDir, "data")
	restoreDir := filepath.Join(tempDir, "restore-target")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "restore-me.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.MkdirAll(restoreDir, 0o755); err != nil {
		t.Fatalf("mkdir restore dir: %v", err)
	}

	_ = os.Setenv("RESTIC_PASSWORD", "integration-test-password")
	defer func() { _ = os.Unsetenv("RESTIC_PASSWORD") }()

	_, err := backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     target,
		Executable: resticPath,
		Args:       []string{"-r", repoDir, "init"},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic init failed: %v", err)
	}

	_, err = backup.ExecuteResticInvocations([]backup.ResticInvocation{{
		Target:     target,
		Executable: resticPath,
		Args:       []string{"-r", repoDir, "backup", dataDir},
	}}, backup.SystemExecutor{})
	if err != nil {
		t.Fatalf("restic backup failed: %v", err)
	}

	profile := "wsl"
	if target == "windows" {
		profile = "windows"
	}
	configPath := filepath.Join(tempDir, "config.yaml")
	if writeErr := writeTestConfig(configPath, profile, repoDir, []string{dataDir}, []string{}, false); writeErr != nil {
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

	backupBinary := os.Getenv("BACKUP_BINARY")
	if backupBinary != "" {
		command := exec.Command(backupBinary, "restore", restoreDir)
		commandOutput, commandErr := command.CombinedOutput()
		if commandErr != nil {
			t.Fatalf("backup binary restore failed: %v (%s)", commandErr, strings.TrimSpace(string(commandOutput)))
		}
	} else {
		_, runErr := backup.Run(backup.Command{Name: "restore", Target: restoreDir}, backup.SystemExecutor{})
		if runErr != nil {
			t.Fatalf("backup restore run failed: %v", runErr)
		}
	}

	hasFiles, scanErr := directoryHasFiles(restoreDir)
	if scanErr != nil {
		t.Fatalf("scan restore output failed: %v", scanErr)
	}
	if !hasFiles {
		t.Fatal("restore output did not contain any files")
	}
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
	runCase(t, "manual-all-cases-single-run", addAllCasesForSingleManifestRun, os.Getenv("BACKUP_ITEST_PAUSE") == "1")
}
