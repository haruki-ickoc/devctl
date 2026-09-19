package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/skyou/devctl/internal/config"
	"github.com/skyou/devctl/internal/docker"
	"github.com/skyou/devctl/internal/ui"
)

var (
	cfgFile string
	verbose bool
	dryRun  bool

	cfg            *config.Config
	runner         *docker.Runner
	composeClient  *docker.ComposeClient
	networkManager *docker.NetworkManager
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "devctl",
	Short: "devctl is an integrated multi-project and core infra Docker management CLI",
	Long: `devctl simplifies managing multi-project Docker Compose environments and
shared infrastructure (reverse proxy, database, networks, logging, etc.).
It acts as a single control plane replacing scattered project Makefiles.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize runner
		runner = docker.NewRunner(dryRun, verbose)

		// Commands that do not require an existing config file
		switch cmd.Name() {
		case "init", "version", "help":
			return nil
		}

		// Load config
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			// If config check or view is called, pass the error along gently
			if cmd.Parent() != nil && cmd.Parent().Name() == "config" {
				return nil
			}
			return fmt.Errorf("configuration error: %w", err)
		}

		// Initialize docker clients
		composeClient = docker.NewComposeClient(runner, cfg.Defaults.ComposeCmd)
		networkManager = docker.NewNetworkManager(runner)

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ~/.config/devctl/config.yaml)")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	RootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "preview commands without executing them")
}
