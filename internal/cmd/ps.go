package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/skyou/devctl/internal/docker"
	"github.com/skyou/devctl/internal/ui"
)

var psCmd = &cobra.Command{
	Use:     "ps",
	Aliases: []string{"status"},
	Short:   "Show status of core infrastructure and all managed projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Shared Network Status
		ui.Step("Shared Network: %s", cfg.Network.Name)
		exists, err := networkManager.Exists(cfg.Network.Name)
		if err != nil {
			ui.Warn("Could not check network: %v", err)
		} else if exists {
			ui.Success("Status: ACTIVE")
		} else {
			ui.Warn("Status: NOT CREATED")
		}
		fmt.Println()

		// 2. Core Infra Status
		if cfg.Core.WorkDir != "" {
			ui.Step("Core Infrastructure (%s):", cfg.Core.WorkDir)
			if _, err := os.Stat(cfg.Core.WorkDir); err == nil {
				opts := docker.ComposeOptions{
					WorkDir:      cfg.Core.WorkDir,
					ComposeFiles: cfg.Core.ComposeFiles,
					EnvFile:      cfg.Core.EnvFile,
				}
				_ = composeClient.Ps(opts)
			} else {
				ui.Warn("Core workdir not found at %s", cfg.Core.WorkDir)
			}
			fmt.Println()
		}

		// 3. Projects Status
		for name, proj := range cfg.Projects {
			ui.Step("Project: %s (%s)", name, proj.WorkDir)
			if _, err := os.Stat(proj.WorkDir); err == nil {
				opts := docker.ComposeOptions{
					WorkDir:      proj.WorkDir,
					ComposeFiles: proj.ComposeFiles,
					EnvFile:      proj.EnvFile,
				}
				_ = composeClient.Ps(opts)
			} else {
				ui.Warn("Project workdir not found at %s", proj.WorkDir)
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(psCmd)
}
