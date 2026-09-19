package cmd

import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/skyou/devctl/internal/ui"
)

var networkCmd = &cobra.Command{
    Use:     "network",
    Aliases: []string{"net"},
    Short:   "共通 Docker ネットワークの管理",
    Long:    `config.yaml で定義された共通 Docker ネットワークの存在確認、手動作成、削除を行います。`,
}

var networkCheckCmd = &cobra.Command{
    Use:   "check",
    Short: "共通ネットワークの存在・設定確認",
    RunE: func(cmd *cobra.Command, args []string) error {
        netName := cfg.Network.Name
        ui.Info("ネットワークを確認中: %s", netName)
        exists, err := networkManager.Exists(netName)
        if err != nil {
            return err
        }
        if exists {
            ui.Success("ネットワーク '%s' は存在します。", netName)
            subnet, gateway, err := networkManager.InspectIPAM(netName)
            if err == nil {
                if subnet != "" || gateway != "" {
                    ui.Info("  実環境構成: Subnet=%s, Gateway=%s", subnet, gateway)
                }
                if cfg.Network.Subnet != "" || cfg.Network.Gateway != "" {
                    ui.Info("  設定ファイル: Subnet=%s, Gateway=%s", cfg.Network.Subnet, cfg.Network.Gateway)
                    var mismatches []string
                    if cfg.Network.Subnet != "" && subnet != "" && cfg.Network.Subnet != subnet {
                        mismatches = append(mismatches, fmt.Sprintf("Subnet (設定: %s, 実環境: %s)", cfg.Network.Subnet, subnet))
                    }
                    if cfg.Network.Gateway != "" && gateway != "" && cfg.Network.Gateway != gateway {
                        mismatches = append(mismatches, fmt.Sprintf("Gateway (設定: %s, 実環境: %s)", cfg.Network.Gateway, gateway))
                    }
                    if len(mismatches) > 0 {
                        ui.Warn("▲ 警告: 設定ファイルと実環境のネットワーク構成が一致していません:")
                        for _, m := range mismatches {
                            ui.Warn("    - %s", m)
                        }
                        ui.Warn("  再作成する場合は 'devctl network rm' を実行後に 'devctl network create' を実行してください。")
                    } else {
                        ui.Success("  Subnet / Gateway の設定は正常に一致しています。")
                    }
                }
            }
        } else {
            ui.Warn("ネットワーク '%s' は存在しません。'devctl network create' で作成できます。", netName)
            if cfg.Network.Subnet != "" || cfg.Network.Gateway != "" {
                ui.Info("  作成予定構成: Subnet=%s, Gateway=%s", cfg.Network.Subnet, cfg.Network.Gateway)
            }
        }
        return nil
    },
}

var networkCreateCmd = &cobra.Command{
    Use:   "create",
    Short: "共通ネットワークを作成（未存在時）",
    RunE: func(cmd *cobra.Command, args []string) error {
        return networkManager.Ensure(cfg.Network.Name, cfg.Network.Driver, cfg.Network.Subnet, cfg.Network.Gateway, cfg.Network.Attachable)
    },
}

var networkRmCmd = &cobra.Command{
    Use:     "rm",
    Aliases: []string{"remove", "delete"},
    Short:   "共通ネットワークを削除",
    RunE: func(cmd *cobra.Command, args []string) error {
        ui.Warn("ネットワーク '%s' を削除します（接続コンテナがある場合は注意してください）", cfg.Network.Name)
        return networkManager.Remove(cfg.Network.Name)
    },
}

func init() {
    RootCmd.AddCommand(networkCmd)
    networkCmd.AddCommand(networkCheckCmd)
    networkCmd.AddCommand(networkCreateCmd)
    networkCmd.AddCommand(networkRmCmd)
}
