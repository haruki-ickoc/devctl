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
	Short: "共通基盤（リバースプロキシ、DB、Redis等）の統合管理",
	Long:  `各プロジェクトが横断して利用する共通インフラ基盤のコンテナ群を管理します。`,
}

var coreUpCmd = &cobra.Command{
	Use:   "up [services...]",
	Short: "共通基盤コンテナを起動",
	Long:  `共通ネットワークの存在を確認・自動作成した上で、共通基盤の Compose サービスを起動します。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Core.WorkDir == "" {
			return fmt.Errorf("config.yaml に core.workdir が設定されていません")
		}

		if _, err := os.Stat(cfg.Core.WorkDir); os.IsNotExist(err) {
			return fmt.Errorf("共通基盤の作業ディレクトリが存在しません: %s", cfg.Core.WorkDir)
		}

		// 共通ネットワークの存在を事前に担保
		if cfg.Network.Name != "" {
			if err := networkManager.Ensure(cfg.Network.Name, cfg.Network.Driver, cfg.Network.Attachable); err != nil {
				return fmt.Errorf("共通ネットワークの準備に失敗しました: %w", err)
			}
		}

		opts := docker.ComposeOptions{
			WorkDir:      cfg.Core.WorkDir,
			ComposeFiles: cfg.Core.ComposeFiles,
			EnvFile:      cfg.Core.EnvFile,
			Services:     args,
		}

		if err := composeClient.Up(opts, coreBuild); err != nil {
			return fmt.Errorf("共通基盤の起動に失敗しました: %w", err)
		}

		ui.Success("共通基盤が正常に起動しました。")
		return nil
	},
}

var coreDownCmd = &cobra.Command{
	Use:   "down",
	Short: "共通基盤コンテナを停止",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Core.WorkDir == "" {
			return fmt.Errorf("config.yaml に core.workdir が設定されていません")
		}

		opts := docker.ComposeOptions{
			WorkDir:      cfg.Core.WorkDir,
			ComposeFiles: cfg.Core.ComposeFiles,
			EnvFile:      cfg.Core.EnvFile,
			Services:     args,
		}

		if err := composeClient.Down(opts, coreRemoveVolumes); err != nil {
			return fmt.Errorf("共通基盤の停止に失敗しました: %w", err)
		}

		ui.Success("共通基盤を停止しました。")
		return nil
	},
}

var coreRestartCmd = &cobra.Command{
	Use:   "restart [services...]",
	Short: "共通基盤コンテナを再起動",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Core.WorkDir == "" {
			return fmt.Errorf("config.yaml に core.workdir が設定されていません")
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
	Short:   "共通基盤コンテナの稼働状況を表示",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Core.WorkDir == "" {
			return fmt.Errorf("config.yaml に core.workdir が設定されていません")
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
	Short: "共通基盤コンテナのログを表示",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Core.WorkDir == "" {
			return fmt.Errorf("config.yaml に core.workdir が設定されていません")
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

	coreUpCmd.Flags().BoolVarP(&coreBuild, "build", "b", false, "起動前にイメージをビルド")
	coreDownCmd.Flags().BoolVarP(&coreRemoveVolumes, "volumes", "v", false, "名前付きボリュームも同時に削除")
	coreLogsCmd.Flags().BoolVarP(&coreFollowLogs, "follow", "f", false, "ログ出力をリアルタイム追跡")
	coreLogsCmd.Flags().StringVarP(&coreTailLogs, "tail", "t", "100", "末尾から表示する行数")
}
