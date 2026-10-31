package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/haruki-ickoc/devctl/internal/docker"
    "github.com/haruki-ickoc/devctl/internal/ui"
)

var (
    coreBuild         bool
    coreRemoveVolumes bool
    coreFollowLogs    bool
    coreTailLogs      string
    coreBackupOutput  string
    coreRestoreYes    bool
)

var coreCmd = &cobra.Command{
    Use:   "core",
    Short: "共通基盤（リバースプロキシ、DB、Redis等）の統合管理",
    Long:  `各プロジェクトが横断して利用する共通インフラ基盤のコンテナ群を管理します。`,
}

var corePullCmd = &cobra.Command{
    Use:   "pull [services...]",
    Short: "共通基盤コンテナイメージを取得",
    Long: `共通基盤（Core）のコンテナイメージをレジストリから取得します（コンテナの起動・再作成は行いません）。
特定のサービスのみを対象にイメージを取得することも可能です。`,
    RunE: func(cmd *cobra.Command, args []string) error {
        if cfg.Core.WorkDir == "" {
            return fmt.Errorf("config.yaml に core.workdir が設定されていません")
        }

        if !dryRun {
            if _, err := os.Stat(cfg.Core.WorkDir); os.IsNotExist(err) {
                return fmt.Errorf("共通基盤の作業ディレクトリが存在しません: %s", cfg.Core.WorkDir)
            }
        }

        opts := docker.ComposeOptions{
            WorkDir:      cfg.Core.WorkDir,
            ComposeFiles: cfg.Core.ComposeFiles,
            EnvFile:      cfg.Core.EnvFile,
            Services:     args,
        }

        if err := composeClient.Pull(opts); err != nil {
            return fmt.Errorf("共通基盤のイメージ取得に失敗しました: %w", err)
        }

        ui.Success("共通基盤のイメージ取得が正常に完了しました。")
        return nil
    },
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
            if err := networkManager.Ensure(cfg.Network.Name, cfg.Network.Driver, cfg.Network.Subnet, cfg.Network.Gateway, cfg.Network.Attachable); err != nil {
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

var coreStartCmd = &cobra.Command{
    Use:   "start [services...]",
    Short: "停止済みの共通基盤コンテナを開始",
    Long: `停止済み（Exited）の共通基盤コンテナを既存状態のまま高速再開します（新規作成は行いません）。
特定のサービスのみを対象に開始することも可能です。`,
    RunE: func(cmd *cobra.Command, args []string) error {
        if cfg.Core.WorkDir == "" {
            return fmt.Errorf("config.yaml に core.workdir が設定されていません")
        }

        if cfg.Network.Name != "" {
            if err := networkManager.Ensure(cfg.Network.Name, cfg.Network.Driver, cfg.Network.Subnet, cfg.Network.Gateway, cfg.Network.Attachable); err != nil {
                return fmt.Errorf("共通ネットワークの準備に失敗しました: %w", err)
            }
        }

        opts := docker.ComposeOptions{
            WorkDir:      cfg.Core.WorkDir,
            ComposeFiles: cfg.Core.ComposeFiles,
            EnvFile:      cfg.Core.EnvFile,
            Services:     args,
        }

        if err := composeClient.Start(opts); err != nil {
            return fmt.Errorf("共通基盤の開始に失敗しました: %w (コンテナが未作成の場合は 'devctl core up' を実行してください)", err)
        }

        ui.Success("共通基盤を開始しました。")
        return nil
    },
}

var coreStopCmd = &cobra.Command{
    Use:   "stop [services...]",
    Short: "稼働中の共通基盤コンテナを一時停止",
    Long: `稼働中の共通基盤コンテナを一時停止（Exited）します（コンテナの削除は行いません）。
ボリュームの backup / restore を行う前の安全な一時停止に適しています。`,
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

        if err := composeClient.Stop(opts); err != nil {
            return fmt.Errorf("共通基盤の一時停止に失敗しました: %w", err)
        }

        ui.Success("共通基盤を一時停止しました。")
        return nil
    },
}

var coreBackupCmd = &cobra.Command{
    Use:   "backup [volumes...]",
    Short: "共通基盤の Named Volume をアーカイブに退避",
    Long: `共通基盤（Core）で定義されている Named Volume のデータを .tar.gz アーカイブとして保存します。
コンテナの削除は行いません。データの整合性を保つため、実行前に 'devctl core stop' でコンテナを一時停止してください。`,
    RunE: func(cmd *cobra.Command, args []string) error {
        if cfg.Core.WorkDir == "" {
            return fmt.Errorf("config.yaml に core.workdir が設定されていません")
        }

        if !dryRun {
            if _, err := os.Stat(cfg.Core.WorkDir); os.IsNotExist(err) {
                return fmt.Errorf("共通基盤の作業ディレクトリが存在しません: %s", cfg.Core.WorkDir)
            }
        }

        vopts := docker.VolumeBackupOptions{
            ComposeOpts: docker.ComposeOptions{
                WorkDir:      cfg.Core.WorkDir,
                ComposeFiles: cfg.Core.ComposeFiles,
                EnvFile:      cfg.Core.EnvFile,
            },
            ScopeType:     "core",
            ScopeName:     "core",
            TargetVolumes: args,
            ArchivePath:   coreBackupOutput,
        }

        if _, err := composeClient.BackupVolumes(vopts); err != nil {
            return fmt.Errorf("共通基盤のボリュームバックアップに失敗しました: %w", err)
        }
        return nil
    },
}

var coreRestoreCmd = &cobra.Command{
    Use:   "restore <backup> [volumes...]",
    Short: "バックアップアーカイブから共通基盤の Named Volume を復元",
    Long: `指定されたバックアップアーカイブ (.tar.gz) から共通基盤（Core）の Named Volume へデータを復元（上書き）します。
使用中の Volume への復元による破損を防ぐため、実行前に 'devctl core stop' でコンテナを一時停止してください。`,
    RunE: func(cmd *cobra.Command, args []string) error {
        if cfg.Core.WorkDir == "" {
            return fmt.Errorf("config.yaml に core.workdir が設定されていません")
        }

        if !dryRun {
            if _, err := os.Stat(cfg.Core.WorkDir); os.IsNotExist(err) {
                return fmt.Errorf("共通基盤の作業ディレクトリが存在しません: %s", cfg.Core.WorkDir)
            }
        }

        if len(args) < 1 {
            return fmt.Errorf("復元元のバックアップアーカイブファイルを指定してください (使用法: devctl core restore <backup> [volumes...])")
        }

        backupFile := args[0]
        targetVolumes := args[1:]

        vopts := docker.VolumeBackupOptions{
            ComposeOpts: docker.ComposeOptions{
                WorkDir:      cfg.Core.WorkDir,
                ComposeFiles: cfg.Core.ComposeFiles,
                EnvFile:      cfg.Core.EnvFile,
            },
            ScopeType:     "core",
            ScopeName:     "core",
            TargetVolumes: targetVolumes,
            ArchivePath:   backupFile,
            AssumeYes:     coreRestoreYes,
        }

        if err := composeClient.RestoreVolumes(vopts); err != nil {
            return fmt.Errorf("共通基盤のボリューム復元に失敗しました: %w", err)
        }
        return nil
    },
}

func init() {
    RootCmd.AddCommand(coreCmd)
    coreCmd.AddCommand(corePullCmd)
    coreCmd.AddCommand(coreUpCmd)
    coreCmd.AddCommand(coreDownCmd)
    coreCmd.AddCommand(coreStartCmd)
    coreCmd.AddCommand(coreStopCmd)
    coreCmd.AddCommand(coreRestartCmd)
    coreCmd.AddCommand(corePsCmd)
    coreCmd.AddCommand(coreLogsCmd)
    coreCmd.AddCommand(coreBackupCmd)
    coreCmd.AddCommand(coreRestoreCmd)

    coreUpCmd.Flags().BoolVarP(&coreBuild, "build", "b", false, "起動前にイメージをビルド")
    coreDownCmd.Flags().BoolVarP(&coreRemoveVolumes, "volumes", "V", false, "名前付きボリュームも同時に削除")
    coreLogsCmd.Flags().BoolVarP(&coreFollowLogs, "follow", "f", false, "ログ出力をリアルタイム追跡")
    coreLogsCmd.Flags().StringVarP(&coreTailLogs, "tail", "t", "100", "末尾から表示する行数")
    coreBackupCmd.Flags().StringVarP(&coreBackupOutput, "output", "o", "", "バックアップアーカイブの出力先ファイルパス (省略時: <workdir>/.devctl/backups/core-<timestamp>.tar.gz)")
    coreRestoreCmd.Flags().BoolVarP(&coreRestoreYes, "yes", "y", false, "上書き確認プロンプトをスキップして実行")
}
