package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/skyou/devctl/internal/config"
	"github.com/skyou/devctl/internal/ui"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect, initialize, and validate devctl configuration",
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show path to the loaded config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg != nil && cfg.LoadedPath != "" {
			fmt.Println(cfg.LoadedPath)
			return nil
		}
		defaultPath, _ := config.DefaultConfigPath()
		fmt.Printf("Default path: %s (not currently created)\n", defaultPath)
		return nil
	},
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Print the current active configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no valid configuration loaded")
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		ui.Info("Active configuration from %s:\n", cfg.LoadedPath)
		fmt.Println(string(data))
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate default configuration file at ~/.config/devctl/config.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		destPath, err := config.DefaultConfigPath()
		if err != nil {
			return err
		}

		if _, err := os.Stat(destPath); err == nil {
			ui.Warn("Configuration file already exists at: %s", destPath)
			ui.Info("To overwrite, manually remove or edit the file.")
			return nil
		}

		dir := filepath.Dir(destPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if err := os.WriteFile(destPath, []byte(config.DefaultConfigYAML), 0644); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}

		ui.Success("Default configuration initialized at: %s", destPath)
		ui.Info("Edit this file to match your local repositories and services.")
		return nil
	},
}

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate configuration syntax and inspect paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("failed to load configuration. Run 'devctl config init'")
		}

		ui.Success("Configuration file syntax is valid: %s", cfg.LoadedPath)
		fmt.Println()

		// Check network
		ui.Step("Checking Shared Network Configuration:")
		ui.Info("  Network Name: %s", cfg.Network.Name)
		ui.Info("  Driver:       %s", cfg.Network.Driver)

		// Check Core
		ui.Step("Checking Core Infrastructure:")
		if cfg.Core.WorkDir == "" {
			ui.Warn("  Core workdir is empty")
		} else {
			if info, err := os.Stat(cfg.Core.WorkDir); err != nil || !info.IsDir() {
				ui.Warn("  Core workdir does not exist: %s", cfg.Core.WorkDir)
			} else {
				ui.Success("  Core workdir exists: %s", cfg.Core.WorkDir)
				for _, f := range cfg.Core.ComposeFiles {
					fullPath := filepath.Join(cfg.Core.WorkDir, f)
					if _, err := os.Stat(fullPath); err != nil {
						ui.Warn("    compose file missing: %s", fullPath)
					} else {
						ui.Success("    compose file found: %s", f)
					}
				}
			}
		}

		// Check Projects
		ui.Step("Checking Managed Projects (%d registered):", len(cfg.Projects))
		for name, proj := range cfg.Projects {
			fmt.Printf("  [%s]:\n", name)
			if info, err := os.Stat(proj.WorkDir); err != nil || !info.IsDir() {
				ui.Warn("    Workdir does not exist: %s", proj.WorkDir)
			} else {
				ui.Success("    Workdir exists: %s", proj.WorkDir)
				for _, f := range proj.ComposeFiles {
					fullPath := filepath.Join(proj.WorkDir, f)
					if _, err := os.Stat(fullPath); err != nil {
						ui.Warn("      compose file missing: %s", fullPath)
					} else {
						ui.Success("      compose file found: %s", f)
					}
				}
			}
			if len(proj.Tasks) > 0 {
				ui.Info("    Registered tasks: %d", len(proj.Tasks))
			}
		}

		fmt.Println()
		ui.Success("Configuration check complete.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configCheckCmd)
}
