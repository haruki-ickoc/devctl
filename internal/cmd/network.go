package cmd

import (
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
    Short: "共通ネットワークの存在確認",
    RunE: func(cmd *cobra.Command, args []string) error {
        netName := cfg.Network.Name
        ui.Info("ネットワークを確認中: %s", netName)
        exists, err := networkManager.Exists(netName)
        if err != nil {
            return err
        }
        if exists {
            ui.Success("ネットワーク '%s' は存在します。", netName)
        } else {
            ui.Warn("ネットワーク '%s' は存在しません。'devctl network create' で作成できます。", netName)
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
