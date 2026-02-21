package backup

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ProfileConfig struct {
	IncludePaths   []string
	ExcludePaths   []string
	UseFSSnapshot  bool
	RepositoryHint string
}

type AppConfig struct {
	Path     string
	Exists   bool
	Profiles map[string]ProfileConfig
}

type fileProfileConfig struct {
	Repository    string   `yaml:"repository"`
	IncludePaths  []string `yaml:"include"`
	ExcludePaths  []string `yaml:"exclude"`
	UseFSSnapshot bool     `yaml:"use_fs_snapshot"`
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
				IncludePaths:   []string{"$HOME"},
				ExcludePaths:   []string{},
				UseFSSnapshot:  false,
				RepositoryHint: "configure per-environment",
			},
			"windows": {
				IncludePaths:   []string{"C:\\Users\\<user>"},
				ExcludePaths:   []string{},
				UseFSSnapshot:  true,
				RepositoryHint: "configure per-environment",
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
		for profileName, profile := range parsed.Profiles {
			loadedProfiles[profileName] = ProfileConfig{
				IncludePaths:   profile.IncludePaths,
				ExcludePaths:   profile.ExcludePaths,
				UseFSSnapshot:  profile.UseFSSnapshot,
				RepositoryHint: profile.Repository,
			}
		}

		return AppConfig{Path: path, Exists: true, Profiles: loadedProfiles}, nil
	} else if !os.IsNotExist(err) {
		return AppConfig{}, fmt.Errorf("read config: %w", err)
	}

	return config, nil
}

func ValidatePlanConfig(plan RunPlan, config AppConfig) error {
	for _, target := range plan.Targets {
		if _, ok := config.Profiles[target]; !ok {
			return fmt.Errorf("missing profile config: %s", target)
		}
	}
	return nil
}
