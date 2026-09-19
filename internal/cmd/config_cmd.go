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
	Short: "devctl 設定ファイルの確認、初期化、構文およびパス検証",
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "現在読み込まれている設定ファイルのパスを表示",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg != nil && cfg.LoadedPath != "" {
			fmt.Println(cfg.LoadedPath)
			return nil
		}
		defaultPath, _ := config.DefaultConfigPath()
		fmt.Printf("既定のパス: %s (現在は未作成)\n", defaultPath)
		return nil
	},
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "現在有効な設定内容を YAML 形式でダンプ出力",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("有効な設定が読み込まれていません")
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		ui.Info("読み込み元: %s の有効な設定内容:\n", cfg.LoadedPath)
		fmt.Println(string(data))
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "~/.config/devctl/config.yaml にデフォルト設定ファイルを生成",
	RunE: func(cmd *cobra.Command, args []string) error {
		destPath, err := config.DefaultConfigPath()
		if err != nil {
			return err
		}

		if _, err := os.Stat(destPath); err == nil {
			ui.Warn("設定ファイルは既に存在します: %s", destPath)
			ui.Info("上書きする場合は、既存のファイルを手動で削除または編集してください。")
			return nil
		}

		dir := filepath.Dir(destPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("ディレクトリ %s の作成に失敗しました: %w", dir, err)
		}

		if err := os.WriteFile(destPath, []byte(config.DefaultConfigYAML), 0644); err != nil {
			return fmt.Errorf("デフォルト設定ファイルの書き込みに失敗しました: %w", err)
		}

		ui.Success("デフォルト設定ファイルを初期化しました: %s", destPath)
		ui.Info("環境に合わせてこのファイルを編集してください。")
		return nil
	},
}

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "設定ファイルの構文と指定ディレクトリ・Compose ファイルの実在性を検証",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("設定ファイルを読み込めませんでした。'devctl config init' を実行してください")
		}

		ui.Success("設定ファイルの構文は正常です: %s", cfg.LoadedPath)
		fmt.Println()

		// 共通ネットワークの検証
		ui.Step("共通ネットワーク設定の検証:")
		ui.Info("  ネットワーク名: %s", cfg.Network.Name)
		ui.Info("  ドライバ:       %s", cfg.Network.Driver)

		// 共通基盤 (Core) の検証
		ui.Step("共通基盤 (Core) の検証:")
		if cfg.Core.WorkDir == "" {
			ui.Warn("  共通基盤の作業ディレクトリ (workdir) が未設定です")
		} else {
			if info, err := os.Stat(cfg.Core.WorkDir); err != nil || !info.IsDir() {
				ui.Warn("  共通基盤の作業ディレクトリが存在しません: %s", cfg.Core.WorkDir)
			} else {
				ui.Success("  共通基盤の作業ディレクトリが存在します: %s", cfg.Core.WorkDir)
				for _, f := range cfg.Core.ComposeFiles {
					fullPath := filepath.Join(cfg.Core.WorkDir, f)
					if _, err := os.Stat(fullPath); err != nil {
						ui.Warn("    Compose ファイルが見つかりません: %s", fullPath)
					} else {
						ui.Success("    Compose ファイルを確認: %s", f)
					}
				}
			}
		}

		// 管理対象プロジェクトの検証
		ui.Step("管理対象プロジェクトの検証 (%d 件登録):", len(cfg.Projects))
		for name, proj := range cfg.Projects {
			fmt.Printf("  [%s]:\n", name)
			if info, err := os.Stat(proj.WorkDir); err != nil || !info.IsDir() {
				ui.Warn("    作業ディレクトリが存在しません: %s", proj.WorkDir)
			} else {
				ui.Success("    作業ディレクトリが存在します: %s", proj.WorkDir)
				for _, f := range proj.ComposeFiles {
					fullPath := filepath.Join(proj.WorkDir, f)
					if _, err := os.Stat(fullPath); err != nil {
						ui.Warn("      Compose ファイルが見つかりません: %s", fullPath)
					} else {
						ui.Success("      Compose ファイルを確認: %s", f)
					}
				}
			}
			if len(proj.Tasks) > 0 {
				ui.Info("    登録タスク数: %d", len(proj.Tasks))
			}
		}

		fmt.Println()
		ui.Success("設定ファイルの検証が完了しました。")
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
