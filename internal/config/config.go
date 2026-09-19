package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the devctl configuration structure.
type Config struct {
	Version  string                 `yaml:"version"`
	Defaults DefaultsConfig         `yaml:"defaults"`
	Network  NetworkConfig          `yaml:"network"`
	Core     CoreConfig             `yaml:"core"`
	Projects map[string]ProjectItem `yaml:"projects"`

	// LoadedPath stores the path of the loaded config file
	LoadedPath string `yaml:"-"`
}

// DefaultsConfig contains global fallback settings.
type DefaultsConfig struct {
	ComposeCmd string `yaml:"compose_cmd"` // e.g. "docker compose" or "docker-compose"
	EnvFile    string `yaml:"env_file"`    // Default env file name
}

// NetworkConfig defines the shared Docker network.
type NetworkConfig struct {
	Name       string `yaml:"name"`
	Driver     string `yaml:"driver"`
	Attachable bool   `yaml:"attachable"`
}

// CoreConfig defines the shared infrastructure services.
type CoreConfig struct {
	WorkDir      string                 `yaml:"workdir"`
	ComposeFiles []string               `yaml:"compose_files"`
	EnvFile      string                 `yaml:"env_file"`
	Services     map[string]ServiceItem `yaml:"services"`
}

// ServiceItem defines metadata for a service inside Core.
type ServiceItem struct {
	Description string `yaml:"description"`
}

// ProjectItem represents an individual managed project.
type ProjectItem struct {
	Name         string            `yaml:"-"` // Key in map
	Description  string            `yaml:"description"`
	WorkDir      string            `yaml:"workdir"`
	ComposeFiles []string          `yaml:"compose_files"`
	EnvFile      string            `yaml:"env_file"`
	DependsOn    DependsOnConfig   `yaml:"depends_on"`
	Tasks        map[string]TaskItem `yaml:"tasks"`
}

// DependsOnConfig defines prerequisite infrastructure for a project.
type DependsOnConfig struct {
	Network bool     `yaml:"network"` // Requires shared network
	Core    []string `yaml:"core"`    // Specific core services required (or empty for all)
}

// TaskItem represents custom project commands (absorbing Makefiles).
type TaskItem struct {
	Description string `yaml:"description"`
	Command     string `yaml:"command"`
}

// Load finds and parses the configuration file.
// If configPath is empty, it searches default locations.
func Load(customPath string) (*Config, error) {
	targetPath, err := resolveConfigPath(customPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", targetPath, err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config: %w", err)
	}

	cfg.LoadedPath = targetPath
	cfg.applyDefaultsAndResolvePaths()

	return &cfg, nil
}

// resolveConfigPath determines the config path to load.
func resolveConfigPath(customPath string) (string, error) {
	if customPath != "" {
		return ExpandPath(customPath), nil
	}

	// 1. Check DEVCTL_CONFIG env var
	if envPath := os.Getenv("DEVCTL_CONFIG"); envPath != "" {
		return ExpandPath(envPath), nil
	}

	// 2. Check current directory
	candidates := []string{
		".devctl.yaml",
		".devctl.yml",
		"devctl.yaml",
		"devctl.yml",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs, nil
		}
	}

	// 3. Check ~/.config/devctl/config.yaml
	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultPath := filepath.Join(homeDir, ".config", "devctl", "config.yaml")
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath, nil
		}
		// Also check config.yml
		altPath := filepath.Join(homeDir, ".config", "devctl", "config.yml")
		if _, err := os.Stat(altPath); err == nil {
			return altPath, nil
		}
		return defaultPath, fmt.Errorf("config file not found. Run 'devctl config init' to create one at %s", defaultPath)
	}

	return "", fmt.Errorf("configuration file not found. Please provide --config or create ~/.config/devctl/config.yaml")
}

// DefaultConfigPath returns the canonical default configuration file path (~/.config/devctl/config.yaml).
func DefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "devctl", "config.yaml"), nil
}

func (c *Config) applyDefaultsAndResolvePaths() {
	if c.Defaults.ComposeCmd == "" {
		c.Defaults.ComposeCmd = "docker compose"
	}
	if c.Network.Name == "" {
		c.Network.Name = "dev-network"
	}
	if c.Network.Driver == "" {
		c.Network.Driver = "bridge"
	}

	// Resolve Core WorkDir
	if c.Core.WorkDir != "" {
		c.Core.WorkDir = ExpandPath(c.Core.WorkDir)
	}
	if len(c.Core.ComposeFiles) == 0 {
		c.Core.ComposeFiles = []string{"docker-compose.yml"}
	}

	// Resolve Project WorkDirs
	for name, proj := range c.Projects {
		proj.Name = name
		if proj.WorkDir != "" {
			proj.WorkDir = ExpandPath(proj.WorkDir)
		}
		if len(proj.ComposeFiles) == 0 {
			proj.ComposeFiles = []string{"docker-compose.yml"}
		}
		c.Projects[name] = proj
	}
}

// FindProjectByWorkingDir inspects the current directory and matches it to a registered project if possible.
func (c *Config) FindProjectByWorkingDir(currentDir string) (*ProjectItem, bool) {
	cleanCurrent, err := filepath.Abs(currentDir)
	if err != nil {
		return nil, false
	}

	for _, p := range c.Projects {
		if p.WorkDir == "" {
			continue
		}
		cleanProj, err := filepath.Abs(p.WorkDir)
		if err != nil {
			continue
		}
		// Exact match or currentDir is a subpath of project WorkDir
		if cleanCurrent == cleanProj || strings.HasPrefix(cleanCurrent, cleanProj+string(filepath.Separator)) {
			projCopy := p
			return &projCopy, true
		}
	}
	return nil, false
}

// ExpandPath expands ~ to the user's home directory and makes the path absolute.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return homeDir
			}
			return filepath.Join(homeDir, path[2:])
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
