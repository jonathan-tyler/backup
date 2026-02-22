package backup

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type CadencePaths struct {
	Daily   []string `yaml:"daily"`
	Weekly  []string `yaml:"weekly"`
	Monthly []string `yaml:"monthly"`
}

type CadencePathFiles struct {
	Daily   string `yaml:"daily"`
	Weekly  string `yaml:"weekly"`
	Monthly string `yaml:"monthly"`
}

func defaultCadencePathFiles(profileName string, ruleType string) CadencePathFiles {
	return CadencePathFiles{
		Daily:   filepath.Join("rules", fmt.Sprintf("%s.%s.daily.txt", profileName, ruleType)),
		Weekly:  filepath.Join("rules", fmt.Sprintf("%s.%s.weekly.txt", profileName, ruleType)),
		Monthly: filepath.Join("rules", fmt.Sprintf("%s.%s.monthly.txt", profileName, ruleType)),
	}
}

func withCadencePathFileDefaults(override CadencePathFiles, defaults CadencePathFiles) CadencePathFiles {
	resolved := defaults
	if strings.TrimSpace(override.Daily) != "" {
		resolved.Daily = override.Daily
	}
	if strings.TrimSpace(override.Weekly) != "" {
		resolved.Weekly = override.Weekly
	}
	if strings.TrimSpace(override.Monthly) != "" {
		resolved.Monthly = override.Monthly
	}
	return resolved
}

func (paths CadencePaths) ForCadence(cadence string) []string {
	switch cadence {
	case "daily":
		return paths.Daily
	case "weekly":
		return paths.Weekly
	case "monthly":
		return paths.Monthly
	default:
		return nil
	}
}

func (paths *CadencePaths) setAll(values []string) {
	paths.Daily = append([]string{}, values...)
	paths.Weekly = append([]string{}, values...)
	paths.Monthly = append([]string{}, values...)
}

func (paths *CadencePaths) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.SequenceNode:
		var values []string
		if err := node.Decode(&values); err != nil {
			return err
		}
		paths.setAll(values)
		return nil
	case yaml.MappingNode:
		type alias CadencePaths
		var decoded alias
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		*paths = CadencePaths(decoded)
		return nil
	default:
		return fmt.Errorf("invalid cadence path format")
	}
}

type ProfileConfig struct {
	IncludeByCadence CadencePaths
	ExcludeByCadence CadencePaths
	UseFSSnapshot    bool
	RepositoryHint   string
}

type AppConfig struct {
	Path     string
	Exists   bool
	Profiles map[string]ProfileConfig
}

type fileProfileConfig struct {
	Repository    string           `yaml:"repository"`
	IncludePaths  CadencePaths     `yaml:"include"`
	ExcludePaths  CadencePaths     `yaml:"exclude"`
	IncludeFiles  CadencePathFiles `yaml:"include_files"`
	ExcludeFiles  CadencePathFiles `yaml:"exclude_files"`
	UseFSSnapshot bool             `yaml:"use_fs_snapshot"`
}

type fileAppConfig struct {
	Profiles map[string]fileProfileConfig `yaml:"profiles"`
}

func ResolveConfigPath(runtime Runtime) (string, error) {
	if override := os.Getenv("BACKUP_CONFIG"); override != "" {
		return override, nil
	}

	if runtime == RuntimeWindows {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "backup", "config.yaml"), nil
		}
	}

	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "backup", "config.yaml"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, ".config", "backup", "config.yaml"), nil
}

func defaultConfig(path string) AppConfig {
	return AppConfig{
		Path:   path,
		Exists: false,
		Profiles: map[string]ProfileConfig{
			"wsl": {
				IncludeByCadence: CadencePaths{Daily: []string{"$HOME"}, Weekly: []string{"$HOME"}, Monthly: []string{"$HOME"}},
				ExcludeByCadence: CadencePaths{Daily: []string{}, Weekly: []string{}, Monthly: []string{}},
				UseFSSnapshot:    false,
				RepositoryHint:   "configure per-environment",
			},
			"windows": {
				IncludeByCadence: CadencePaths{Daily: []string{"C:\\Users\\<user>"}, Weekly: []string{"C:\\Users\\<user>"}, Monthly: []string{"C:\\Users\\<user>"}},
				ExcludeByCadence: CadencePaths{Daily: []string{}, Weekly: []string{}, Monthly: []string{}},
				UseFSSnapshot:    true,
				RepositoryHint:   "configure per-environment",
			},
		},
	}
}

func LoadConfig(runtime Runtime) (AppConfig, error) {
	path, err := ResolveConfigPath(runtime)
	if err != nil {
		return AppConfig{}, err
	}

	config := defaultConfig(path)
	if _, err := os.Stat(path); err == nil {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return AppConfig{}, fmt.Errorf("read config: %w", readErr)
		}

		var parsed fileAppConfig
		if unmarshalErr := yaml.Unmarshal(data, &parsed); unmarshalErr != nil {
			return AppConfig{}, fmt.Errorf("parse config: %w", unmarshalErr)
		}

		loadedProfiles := map[string]ProfileConfig{}
		configDir := filepath.Dir(path)
		for profileName, profile := range parsed.Profiles {
			includeFiles := withCadencePathFileDefaults(profile.IncludeFiles, defaultCadencePathFiles(profileName, "include"))
			excludeFiles := withCadencePathFileDefaults(profile.ExcludeFiles, defaultCadencePathFiles(profileName, "exclude"))

			includeFromFiles, loadIncludeErr := loadCadencePathsFromFiles(includeFiles, configDir)
			if loadIncludeErr != nil {
				return AppConfig{}, fmt.Errorf("load include files for profile %s: %w", profileName, loadIncludeErr)
			}

			excludeFromFiles, loadExcludeErr := loadCadencePathsFromFiles(excludeFiles, configDir)
			if loadExcludeErr != nil {
				return AppConfig{}, fmt.Errorf("load exclude files for profile %s: %w", profileName, loadExcludeErr)
			}

			loadedProfiles[profileName] = ProfileConfig{
				IncludeByCadence: mergeCadencePaths(profile.IncludePaths, includeFromFiles),
				ExcludeByCadence: mergeCadencePaths(profile.ExcludePaths, excludeFromFiles),
				UseFSSnapshot:    profile.UseFSSnapshot,
				RepositoryHint:   profile.Repository,
			}
		}

		return AppConfig{Path: path, Exists: true, Profiles: loadedProfiles}, nil
	} else if !os.IsNotExist(err) {
		return AppConfig{}, fmt.Errorf("read config: %w", err)
	}

	return config, nil
}

func mergeCadencePaths(inline CadencePaths, fromFiles CadencePaths) CadencePaths {
	return CadencePaths{
		Daily:   append(append([]string{}, inline.Daily...), fromFiles.Daily...),
		Weekly:  append(append([]string{}, inline.Weekly...), fromFiles.Weekly...),
		Monthly: append(append([]string{}, inline.Monthly...), fromFiles.Monthly...),
	}
}

func loadCadencePathsFromFiles(files CadencePathFiles, configDir string) (CadencePaths, error) {
	daily, err := loadPathListFile(files.Daily, configDir)
	if err != nil {
		return CadencePaths{}, err
	}

	weekly, err := loadPathListFile(files.Weekly, configDir)
	if err != nil {
		return CadencePaths{}, err
	}

	monthly, err := loadPathListFile(files.Monthly, configDir)
	if err != nil {
		return CadencePaths{}, err
	}

	return CadencePaths{Daily: daily, Weekly: weekly, Monthly: monthly}, nil
}

func loadPathListFile(path string, configDir string) ([]string, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return []string{}, nil
	}

	resolvedPath := trimmedPath
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(configDir, resolvedPath)
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("open path list file %s: %w", resolvedPath, err)
	}
	defer file.Close()

	paths := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		paths = append(paths, line)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("read path list file %s: %w", resolvedPath, scanErr)
	}

	return paths, nil
}

func ValidatePlanConfig(plan RunPlan, config AppConfig) error {
	for _, target := range plan.Targets {
		if _, ok := config.Profiles[target]; !ok {
			return fmt.Errorf("missing profile config: %s", target)
		}
	}
	return nil
}
