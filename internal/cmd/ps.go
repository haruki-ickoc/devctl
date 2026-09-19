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
    Short:   "共通基盤および全プロジェクトの稼働ステータスを一覧表示",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. 共通ネットワークのステータス
        ui.Step("共通ネットワーク: %s", cfg.Network.Name)
        exists, err := networkManager.Exists(cfg.Network.Name)
        if err != nil {
            ui.Warn("ネットワークの確認に失敗しました: %v", err)
        } else if exists {
            ui.Success("ステータス: 稼働中 (ACTIVE)")
        } else {
            ui.Warn("ステータス: 未作成 (NOT CREATED)")
        }
        fmt.Println()

        // 2. 共通基盤 (Core) のステータス
        if cfg.Core.WorkDir != "" {
            ui.Step("共通基盤 (%s):", cfg.Core.WorkDir)
            if _, err := os.Stat(cfg.Core.WorkDir); err == nil {
                opts := docker.ComposeOptions{
                    WorkDir:      cfg.Core.WorkDir,
                    ComposeFiles: cfg.Core.ComposeFiles,
                    EnvFile:      cfg.Core.EnvFile,
                }
                _ = composeClient.Ps(opts)
            } else {
                ui.Warn("共通基盤の作業ディレクトリが見つかりません: %s", cfg.Core.WorkDir)
            }
            fmt.Println()
        }

        // 3. 各プロジェクトのステータス
        for name, proj := range cfg.Projects {
            ui.Step("プロジェクト: %s (%s)", name, proj.WorkDir)
            if _, err := os.Stat(proj.WorkDir); err == nil {
                opts := docker.ComposeOptions{
                    WorkDir:      proj.WorkDir,
                    ComposeFiles: proj.ComposeFiles,
                    EnvFile:      proj.EnvFile,
                }
                _ = composeClient.Ps(opts)
            } else {
                ui.Warn("プロジェクトの作業ディレクトリが見つかりません: %s", proj.WorkDir)
            }
            fmt.Println()
        }

        return nil
    },
}

func init() {
    RootCmd.AddCommand(psCmd)
}
