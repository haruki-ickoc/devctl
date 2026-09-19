package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version はビルド時に ldflags で設定されるバージョン情報です
	Version   = "0.0.2"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "devctl のバージョン・ビルド情報を表示",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("devctl version %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
