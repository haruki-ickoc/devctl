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

// RootCmd はサブコマンド未指定時に呼び出されるベースコマンドです。
var RootCmd = &cobra.Command{
	Use:   "devctl",
	Short: "複数プロジェクトと共通基盤を統合管理する Docker CLI ツール",
	Long: `devctl は、複数の Docker Compose プロジェクト環境および
共通インフラ基盤（リバースプロキシ、データベース、ネットワーク、ログ等）を
単一のコマンド体系から統合管理する CLI ツールです。
各リポジトリに散在しがちな Makefile の処理を config.yaml へ集約・代替します。`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// ランナーの初期化
		runner = docker.NewRunner(dryRun, verbose)

		// 既存設定ファイルを必須としないコマンド群
		switch cmd.Name() {
		case "init", "version", "help":
			return nil
		}

		// 設定ファイルのロード
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			// config コマンド配下の実行時はエラーをそのまま渡す
			if cmd.Parent() != nil && cmd.Parent().Name() == "config" {
				return nil
			}
			return fmt.Errorf("設定エラー: %w", err)
		}

		// Docker クライアント群の初期化
		composeClient = docker.NewComposeClient(runner, cfg.Defaults.ComposeCmd)
		networkManager = docker.NewNetworkManager(runner)

		return nil
	},
}

// Execute はルートコマンドを実行し、フラグを適切に設定します。
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "設定ファイルパスを指定 (デフォルト: ~/.config/devctl/config.yaml)")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "詳細ログ出力を有効化")
	RootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "コマンドを実行せずにプレビュー表示")
}
