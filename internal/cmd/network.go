package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/skyou/devctl/internal/ui"
)

var networkCmd = &cobra.Command{
	Use:     "network",
	Aliases: []string{"net"},
	Short:   "Manage shared Docker network",
	Long:    `Inspect, create, or remove the shared Docker network specified in config.yaml.`,
}

var networkCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if the shared network exists",
	RunE: func(cmd *cobra.Command, args []string) error {
		netName := cfg.Network.Name
		ui.Info("Checking network: %s", netName)
		exists, err := networkManager.Exists(netName)
		if err != nil {
			return err
		}
		if exists {
			ui.Success("Network '%s' exists.", netName)
		} else {
			ui.Warn("Network '%s' DOES NOT exist. Run 'devctl network create' to create it.", netName)
		}
		return nil
	},
}

var networkCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create the shared network if missing",
	RunE: func(cmd *cobra.Command, args []string) error {
		return networkManager.Ensure(cfg.Network.Name, cfg.Network.Driver, cfg.Network.Attachable)
	},
}

var networkRmCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"remove", "delete"},
	Short:   "Remove the shared network",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Are you sure you want to remove network '%s'? (use caution if containers are connected)\n", cfg.Network.Name)
		return networkManager.Remove(cfg.Network.Name)
	},
}

func init() {
	RootCmd.AddCommand(networkCmd)
	networkCmd.AddCommand(networkCheckCmd)
	networkCmd.AddCommand(networkCreateCmd)
	networkCmd.AddCommand(networkRmCmd)
}
