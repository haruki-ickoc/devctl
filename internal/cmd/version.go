package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is set via ldflags during build
	Version   = "0.1.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print devctl version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("devctl version %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
