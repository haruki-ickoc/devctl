package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/skyou/devctl/internal/docker"
	"github.com/skyou/devctl/internal/ui"
)

var (
	coreBuild         bool
	coreRemoveVolumes bool
	coreFollowLogs    bool
	coreTailLogs      string
)

var coreCmd = &cobra.Command{
	Use:   "core",
	Short: "Manage shared core infrastructure (reverse proxy, db, redis, etc.)",
	Long:  `Manage shared core infrastructure containers across all projects.`,
}

var coreUpCmd = &cobra.Command{
	Use:   "up [services...]",
	Short: "Start core infrastructure services",
	Long:  `Ensures the shared network exists, then starts core infrastructure compose services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Core.WorkDir == "" {
			return fmt.Errorf("core.workdir is not configured in config.yaml")
		}

		if _, err := os.Stat(cfg.Core.WorkDir); os.IsNotExist(err) {
			return fmt.Errorf("core workdir does not exist: %s", cfg.Core.WorkDir)
		}

		// Ensure network exists first
		if cfg.Network.Name != "" {
			if err := networkManager.Ensure(cfg.Network.Name, cfg.Network.Driver, cfg.Network.Attachable); err != nil {
				return fmt.Errorf("failed to prepare shared network: %w", err)
			}
		}

		opts := docker.ComposeOptions{
			WorkDir:      cfg.Core.WorkDir,
			ComposeFiles: cfg.Core.ComposeFiles,
			EnvFile:      cfg.Core.EnvFile,
			Services:     args,
		}

		if err := composeClient.Up(opts, coreBuild); err != nil {
			return fmt.Errorf("failed to start core infrastructure: %w", err)
		}

		ui.Success("Core infrastructure started successfully.")
		return nil
	},
}

var coreDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop core infrastructure services",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Core.WorkDir == "" {
			return fmt.Errorf("core.workdir is not configured in config.yaml")
		}

		opts := docker.ComposeOptions{
			WorkDir:      cfg.Core.WorkDir,
			ComposeFiles: cfg.Core.ComposeFiles,
			EnvFile:      cfg.Core.EnvFile,
			Services:     args,
		}

		if err := composeClient.Down(opts, coreRemoveVolumes); err != nil {
			return fmt.Errorf("failed to stop core infrastructure: %w", err)
		}

		ui.Success("Core infrastructure stopped.")
		return nil
	},
}

var coreRestartCmd = &cobra.Command{
	Use:   "restart [services...]",
	Short: "Restart core infrastructure services",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Core.WorkDir == "" {
			return fmt.Errorf("core.workdir is not configured in config.yaml")
		}

		opts := docker.ComposeOptions{
			WorkDir:      cfg.Core.WorkDir,
			ComposeFiles: cfg.Core.ComposeFiles,
			EnvFile:      cfg.Core.EnvFile,
			Services:     args,
		}

		return composeClient.Restart(opts)
	},
}

var corePsCmd = &cobra.Command{
	Use:     "ps",
	Aliases: []string{"status"},
	Short:   "Show status of core infrastructure containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Core.WorkDir == "" {
			return fmt.Errorf("core.workdir is not configured in config.yaml")
		}

		opts := docker.ComposeOptions{
			WorkDir:      cfg.Core.WorkDir,
			ComposeFiles: cfg.Core.ComposeFiles,
			EnvFile:      cfg.Core.EnvFile,
			Services:     args,
		}

		return composeClient.Ps(opts)
	},
}

var coreLogsCmd = &cobra.Command{
	Use:   "logs [services...]",
	Short: "View logs of core infrastructure containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Core.WorkDir == "" {
			return fmt.Errorf("core.workdir is not configured in config.yaml")
		}

		opts := docker.ComposeOptions{
			WorkDir:      cfg.Core.WorkDir,
			ComposeFiles: cfg.Core.ComposeFiles,
			EnvFile:      cfg.Core.EnvFile,
			Services:     args,
		}

		return composeClient.Logs(opts, coreFollowLogs, coreTailLogs)
	},
}

func init() {
	RootCmd.AddCommand(coreCmd)
	coreCmd.AddCommand(coreUpCmd)
	coreCmd.AddCommand(coreDownCmd)
	coreCmd.AddCommand(coreRestartCmd)
	coreCmd.AddCommand(corePsCmd)
	coreCmd.AddCommand(coreLogsCmd)

	coreUpCmd.Flags().BoolVarP(&coreBuild, "build", "b", false, "build images before starting")
	coreDownCmd.Flags().BoolVarP(&coreRemoveVolumes, "volumes", "v", false, "remove named volumes")
	coreLogsCmd.Flags().BoolVarP(&coreFollowLogs, "follow", "f", false, "follow log output")
	coreLogsCmd.Flags().StringVarP(&coreTailLogs, "tail", "t", "100", "number of lines to show from the end of the logs")
}
