package cmd

import (
    "fmt"
    "os"
    "sort"
    "text/tabwriter"

    "github.com/spf13/cobra"
    "github.com/haruki-ickoc/devctl/internal/config"
    "github.com/haruki-ickoc/devctl/internal/docker"
    "github.com/haruki-ickoc/devctl/internal/ui"
)

var (
    projectBuild         bool
    projectNoCache       bool
    projectPull          bool
    projectRemoveVolumes bool
    projectFollowLogs    bool
    projectTailLogs      string
    projectComposeFiles  []string
    projectBackupOutput  string
    projectRestoreYes    bool
)

var projectCmd = &cobra.Command{
    Use:     "project",
    Aliases: []string{"proj", "p"},
    Short:   "各プロジェクト環境の個別管理",
    Long:    `登録された各プロジェクトコンテナの起動、停止、状態確認、カスタムタスク実行を任意の場所から行います。`,
}

// resolveTargetProject は対象プロジェクトを判定します。
// コマンドライン引数を最優先とし、未指定時はカレントディレクトリから合致するプロジェクトを自動検出します。
func resolveTargetProject(args []string) (*config.ProjectItem, []string, error) {
    if len(args) > 0 {
        projectName := args[0]
        if proj, ok := cfg.Projects[projectName]; ok {
            return &proj, args[1:], nil
        }
        // 第1引数がプロジェクト名でない場合、カレントディレクトリがプロジェクトか確認
        if currentDir, err := os.Getwd(); err == nil {
            if proj, found := cfg.FindProjectByWorkingDir(currentDir); found {
                return proj, args, nil
            }
        }
        return nil, nil, fmt.Errorf("プロジェクト '%s' は設定ファイルに見つかりません", projectName)
    }

    // 引数が指定されていない場合: カレントディレクトリを確認
    currentDir, err := os.Getwd()
    if err == nil {
        if proj, found := cfg.FindProjectByWorkingDir(currentDir); found {
            ui.Info("カレントディレクトリ (%s) からプロジェクト '%s' を自動検出しました", currentDir, proj.Name)
            return proj, nil, nil
        }
    }

    return nil, nil, fmt.Errorf("プロジェクト名が指定されておらず、カレントディレクトリも登録プロジェクトではありません")
}

// prepareDependencies はプロジェクトが依存する共通ネットワークや Core サービスの起動状態を事前に検証・確保します。
func prepareDependencies(proj *config.ProjectItem) error {
    if proj.DependsOn.Network {
        if err := networkManager.Ensure(cfg.Network.Name, cfg.Network.Driver, cfg.Network.Subnet, cfg.Network.Gateway, cfg.Network.Attachable); err != nil {
            return fmt.Errorf("%s の共通ネットワーク準備に失敗しました: %w", proj.Name, err)
        }
    }

    if len(proj.DependsOn.Core) > 0 {
        if cfg.Core.WorkDir == "" {
            return fmt.Errorf("プロジェクト '%s' は共通基盤 (%v) に依存していますが、config.yaml に core.workdir が設定されていません", proj.Name, proj.DependsOn.Core)
        }
        if !dryRun {
            if _, err := os.Stat(cfg.Core.WorkDir); os.IsNotExist(err) {
                return fmt.Errorf("共通基盤の作業ディレクトリが存在しません: %s", cfg.Core.WorkDir)
            }
        }

        ui.Info("%s が依存する共通基盤サービスを検証中: %v", proj.Name, proj.DependsOn.Core)
        opts := docker.ComposeOptions{
            WorkDir:      cfg.Core.WorkDir,
            ComposeFiles: cfg.Core.ComposeFiles,
            EnvFile:      cfg.Core.EnvFile,
            Services:     proj.DependsOn.Core,
        }
        if err := composeClient.EnsureServicesRunning(opts); err != nil {
            return fmt.Errorf("必要な共通基盤サービスの準備に失敗しました: %w", err)
        }
    }
    return nil
}

var projectListCmd = &cobra.Command{
    Use:     "list",
    Aliases: []string{"ls"},
    Short:   "登録済みプロジェクト一覧を表示",
    RunE: func(cmd *cobra.Command, args []string) error {
        if len(cfg.Projects) == 0 {
            ui.Warn("設定ファイルにプロジェクトが登録されていません。")
            return nil
        }

        w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
        fmt.Fprintln(w, "PROJECT\tWORKDIR\tTASKS\tDESCRIPTION")

        var names []string
        for name := range cfg.Projects {
            names = append(names, name)
        }
        sort.Strings(names)

        for _, name := range names {
            p := cfg.Projects[name]
            taskCount := len(p.Tasks)
            fmt.Fprintf(w, "%s\t%s\t%d 件\t%s\n", p.Name, p.WorkDir, taskCount, p.Description)
        }
        w.Flush()
        return nil
    },
}

var projectBuildCmd = &cobra.Command{
    Use:   "build [project-name] [services...]",
    Short: "プロジェクトコンテナイメージをビルド",
    Long: `指定されたプロジェクトのコンテナイメージをビルドします（コンテナは起動しません）。
特定のサービスのみを対象にビルドすることも可能です。`,
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        opts := docker.ComposeOptions{
            WorkDir:            proj.WorkDir,
            ComposeFiles:       proj.ComposeFiles,
            CustomComposeFiles: projectComposeFiles,
            EnvFile:            proj.EnvFile,
            Services:           remainingArgs,
        }

        if err := composeClient.Build(opts, projectNoCache, projectPull); err != nil {
            return fmt.Errorf("プロジェクト '%s' のイメージビルドに失敗しました: %w", proj.Name, err)
        }

        ui.Success("プロジェクト '%s' のイメージビルドが正常に完了しました。", proj.Name)
        return nil
    },
}

var projectPullCmd = &cobra.Command{
    Use:   "pull [project-name] [services...]",
    Short: "プロジェクトコンテナイメージを取得",
    Long: `指定されたプロジェクトのコンテナイメージをレジストリから取得します（コンテナの起動・再作成は行いません）。
特定のサービスのみを対象にイメージを取得することも可能です。`,
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        opts := docker.ComposeOptions{
            WorkDir:            proj.WorkDir,
            ComposeFiles:       proj.ComposeFiles,
            CustomComposeFiles: projectComposeFiles,
            EnvFile:            proj.EnvFile,
            Services:           remainingArgs,
        }

        if err := composeClient.Pull(opts); err != nil {
            return fmt.Errorf("プロジェクト '%s' のイメージ取得に失敗しました: %w", proj.Name, err)
        }

        ui.Success("プロジェクト '%s' のイメージ取得が正常に完了しました。", proj.Name)
        return nil
    },
}

var projectUpCmd = &cobra.Command{
    Use:   "up [project-name] [services...]",
    Short: "プロジェクトコンテナを起動",
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        if err := prepareDependencies(proj); err != nil {
            return err
        }

        opts := docker.ComposeOptions{
            WorkDir:            proj.WorkDir,
            ComposeFiles:       proj.ComposeFiles,
            CustomComposeFiles: projectComposeFiles,
            EnvFile:            proj.EnvFile,
            Services:           remainingArgs,
        }

        if err := composeClient.Up(opts, projectBuild); err != nil {
            return fmt.Errorf("プロジェクト '%s' の起動に失敗しました: %w", proj.Name, err)
        }

        ui.Success("プロジェクト '%s' が正常に起動しました。", proj.Name)
        return nil
    },
}

var projectDownCmd = &cobra.Command{
    Use:   "down [project-name] [services...]",
    Short: "プロジェクトコンテナを停止",
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        opts := docker.ComposeOptions{
            WorkDir:            proj.WorkDir,
            ComposeFiles:       proj.ComposeFiles,
            CustomComposeFiles: projectComposeFiles,
            EnvFile:            proj.EnvFile,
            Services:           remainingArgs,
        }

        if err := composeClient.Down(opts, projectRemoveVolumes); err != nil {
            return fmt.Errorf("プロジェクト '%s' の停止に失敗しました: %w", proj.Name, err)
        }

        ui.Success("プロジェクト '%s' を停止しました。", proj.Name)
        return nil
    },
}

var projectStartCmd = &cobra.Command{
    Use:   "start [project-name] [services...]",
    Short: "停止済みのプロジェクトコンテナを開始",
    Long: `停止済み（Exited）のプロジェクトコンテナを既存状態のまま高速再開します（新規作成は行いません）。
特定のサービスのみを対象に開始することも可能です。`,
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        if err := prepareDependencies(proj); err != nil {
            return err
        }

        opts := docker.ComposeOptions{
            WorkDir:            proj.WorkDir,
            ComposeFiles:       proj.ComposeFiles,
            CustomComposeFiles: projectComposeFiles,
            EnvFile:            proj.EnvFile,
            Services:           remainingArgs,
        }

        if err := composeClient.Start(opts); err != nil {
            return fmt.Errorf("プロジェクト '%s' の開始に失敗しました: %w (コンテナが未作成の場合は 'devctl up %s' を実行してください)", proj.Name, err, proj.Name)
        }

        ui.Success("プロジェクト '%s' を開始しました。", proj.Name)
        return nil
    },
}

var projectStopCmd = &cobra.Command{
    Use:   "stop [project-name] [services...]",
    Short: "稼働中のプロジェクトコンテナを一時停止",
    Long: `稼働中のプロジェクトコンテナを一時停止（Exited）します（コンテナの削除は行いません）。
特定のサービスのみを対象に停止することも可能です。`,
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        opts := docker.ComposeOptions{
            WorkDir:            proj.WorkDir,
            ComposeFiles:       proj.ComposeFiles,
            CustomComposeFiles: projectComposeFiles,
            EnvFile:            proj.EnvFile,
            Services:           remainingArgs,
        }

        if err := composeClient.Stop(opts); err != nil {
            return fmt.Errorf("プロジェクト '%s' の一時停止に失敗しました: %w", proj.Name, err)
        }

        ui.Success("プロジェクト '%s' を一時停止しました。", proj.Name)
        return nil
    },
}

var projectRestartCmd = &cobra.Command{
    Use:   "restart [project-name] [services...]",
    Short: "プロジェクトコンテナを再起動",
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        if err := prepareDependencies(proj); err != nil {
            return err
        }

        opts := docker.ComposeOptions{
            WorkDir:            proj.WorkDir,
            ComposeFiles:       proj.ComposeFiles,
            CustomComposeFiles: projectComposeFiles,
            EnvFile:            proj.EnvFile,
            Services:           remainingArgs,
        }

        return composeClient.Restart(opts)
    },
}

var projectPsCmd = &cobra.Command{
    Use:   "ps [project-name]",
    Short: "プロジェクトコンテナの稼働状況を表示",
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        opts := docker.ComposeOptions{
            WorkDir:            proj.WorkDir,
            ComposeFiles:       proj.ComposeFiles,
            CustomComposeFiles: projectComposeFiles,
            EnvFile:            proj.EnvFile,
            Services:           remainingArgs,
        }

        return composeClient.Ps(opts)
    },
}

var projectLogsCmd = &cobra.Command{
    Use:   "logs [project-name] [services...]",
    Short: "プロジェクトコンテナのログを表示",
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        opts := docker.ComposeOptions{
            WorkDir:            proj.WorkDir,
            ComposeFiles:       proj.ComposeFiles,
            CustomComposeFiles: projectComposeFiles,
            EnvFile:            proj.EnvFile,
            Services:           remainingArgs,
        }

        return composeClient.Logs(opts, projectFollowLogs, projectTailLogs)
    },
}

var projectExecCmd = &cobra.Command{
    Use:   "exec [project-name] <service> <command...>",
    Short: "プロジェクトコンテナ内でコマンドを実行",
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        if len(remainingArgs) < 2 {
            return fmt.Errorf("使用法: devctl project exec [project-name] <service> <command...>")
        }

        service := remainingArgs[0]
        execCmd := remainingArgs[1:]

        opts := docker.ComposeOptions{
            WorkDir:            proj.WorkDir,
            ComposeFiles:       proj.ComposeFiles,
            CustomComposeFiles: projectComposeFiles,
            EnvFile:            proj.EnvFile,
        }

        return composeClient.Exec(opts, service, execCmd)
    },
}

var projectRunCmd = &cobra.Command{
    Use:   "run [project-name] <task-name>",
    Short: "事前定義されたプロジェクトタスクを実行 (Makefile代替)",
    Long: `config.yaml のプロジェクト 'tasks' セクションで定義されたコマンドを実行します。
使用例:
  devctl run catchUper logs
  devctl project run catchUper backend-sh`,
    RunE: func(cmd *cobra.Command, args []string) error {
        if len(args) == 0 {
            return fmt.Errorf("タスク名またはプロジェクト名が指定されていません")
        }

        var proj *config.ProjectItem
        var taskName string

        // パターン1: devctl run <proj> <task>
        if p, ok := cfg.Projects[args[0]]; ok {
            if len(args) < 2 {
                // プロジェクト内で定義済みのタスク一覧を表示
                ui.Info("プロジェクト '%s' の利用可能タスク一覧:", p.Name)
                for tName, tItem := range p.Tasks {
                    fmt.Printf("  %-15s %s\n", tName, tItem.Summary())
                }
                return fmt.Errorf("実行するタスク名を指定してください")
            }
            proj = &p
            taskName = args[1]
        } else {
            // パターン2: devctl run <task> (カレントディレクトリがプロジェクト内の場合)
            currentDir, err := os.Getwd()
            if err == nil {
                if detected, found := cfg.FindProjectByWorkingDir(currentDir); found {
                    proj = detected
                    taskName = args[0]
                }
            }
        }

        if proj == nil {
            return fmt.Errorf("タスクを実行する対象プロジェクトを判定できませんでした。プロジェクト名を指定してください: devctl run <project> <task>")
        }

        task, ok := proj.Tasks[taskName]
        if !ok {
            ui.Warn("タスク '%s' はプロジェクト '%s' に定義されていません。利用可能タスク一覧:", taskName, proj.Name)
            for tName, tItem := range proj.Tasks {
                fmt.Printf("  %-15s %s\n", tName, tItem.Summary())
            }
            return fmt.Errorf("タスクが未定義です")
        }

        cmds := task.GetCommands()
        if len(cmds) == 0 {
            return fmt.Errorf("タスク '%s' に実行可能なコマンドが定義されていません", taskName)
        }

        ui.Info("プロジェクト '%s' でタスク '%s' を実行中...", proj.Name, taskName)
        return composeClient.RunShellTasks(proj.WorkDir, cmds)
    },
}

var projectBackupCmd = &cobra.Command{
    Use:   "backup [project-name] [volumes...]",
    Short: "プロジェクトの Named Volume をアーカイブに退避",
    Long: `指定されたプロジェクトで定義されている Named Volume のデータを .tar.gz アーカイブとして保存します。
コンテナの削除は行いません。データの整合性を保つため、実行前に 'devctl stop <project>' でコンテナを一時停止してください。`,
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        vopts := docker.VolumeBackupOptions{
            ComposeOpts: docker.ComposeOptions{
                WorkDir:            proj.WorkDir,
                ComposeFiles:       proj.ComposeFiles,
                CustomComposeFiles: projectComposeFiles,
                EnvFile:            proj.EnvFile,
            },
            ScopeType:     "project",
            ScopeName:     proj.Name,
            TargetVolumes: remainingArgs,
            ArchivePath:   projectBackupOutput,
        }

        if _, err := composeClient.BackupVolumes(vopts); err != nil {
            return fmt.Errorf("プロジェクト '%s' のボリュームバックアップに失敗しました: %w", proj.Name, err)
        }
        return nil
    },
}

var projectRestoreCmd = &cobra.Command{
    Use:   "restore [project-name] <backup> [volumes...]",
    Short: "バックアップアーカイブからプロジェクトの Named Volume を復元",
    Long: `指定されたバックアップアーカイブ (.tar.gz) からプロジェクトの Named Volume へデータを復元（上書き）します。
使用中の Volume への復元による破損を防ぐため、実行前に 'devctl stop <project>' でコンテナを一時停止してください。`,
    RunE: func(cmd *cobra.Command, args []string) error {
        proj, remainingArgs, err := resolveTargetProject(args)
        if err != nil {
            return err
        }

        if len(remainingArgs) < 1 {
            return fmt.Errorf("復元元のバックアップアーカイブファイルを指定してください (使用法: devctl project restore [project-name] <backup> [volumes...])")
        }

        backupFile := remainingArgs[0]
        targetVolumes := remainingArgs[1:]

        vopts := docker.VolumeBackupOptions{
            ComposeOpts: docker.ComposeOptions{
                WorkDir:            proj.WorkDir,
                ComposeFiles:       proj.ComposeFiles,
                CustomComposeFiles: projectComposeFiles,
                EnvFile:            proj.EnvFile,
            },
            ScopeType:     "project",
            ScopeName:     proj.Name,
            TargetVolumes: targetVolumes,
            ArchivePath:   backupFile,
            AssumeYes:     projectRestoreYes,
        }

        if err := composeClient.RestoreVolumes(vopts); err != nil {
            return fmt.Errorf("プロジェクト '%s' のボリューム復元に失敗しました: %w", proj.Name, err)
        }
        return nil
    },
}

// 日常の開発体験を快適にするトップレベルエイリアスコマンド群
var (
    topBuildCmd = &cobra.Command{
        Use:   "build [project-name] [services...]",
        Short: "'project build' の短縮エイリアス",
        RunE:  projectBuildCmd.RunE,
    }
    topPullCmd = &cobra.Command{
        Use:   "pull [project-name] [services...]",
        Short: "'project pull' の短縮エイリアス",
        RunE:  projectPullCmd.RunE,
    }
    topUpCmd = &cobra.Command{
        Use:   "up [project-name] [services...]",
        Short: "'project up' の短縮エイリアス",
        RunE:  projectUpCmd.RunE,
    }
    topDownCmd = &cobra.Command{
        Use:   "down [project-name] [services...]",
        Short: "'project down' の短縮エイリアス",
        RunE:  projectDownCmd.RunE,
    }
    topStartCmd = &cobra.Command{
        Use:   "start [project-name] [services...]",
        Short: "'project start' の短縮エイリアス",
        RunE:  projectStartCmd.RunE,
    }
    topStopCmd = &cobra.Command{
        Use:   "stop [project-name] [services...]",
        Short: "'project stop' の短縮エイリアス",
        RunE:  projectStopCmd.RunE,
    }
    topRestartCmd = &cobra.Command{
        Use:   "restart [project-name] [services...]",
        Short: "'project restart' の短縮エイリアス",
        RunE:  projectRestartCmd.RunE,
    }
    topLogsCmd = &cobra.Command{
        Use:   "logs [project-name] [services...]",
        Short: "'project logs' の短縮エイリアス",
        RunE:  projectLogsCmd.RunE,
    }
    topRunCmd = &cobra.Command{
        Use:   "run [project-name] <task-name>",
        Short: "'project run' の短縮エイリアス",
        RunE:  projectRunCmd.RunE,
    }
    topListCmd = &cobra.Command{
        Use:   "list",
        Short: "'project list' の短縮エイリアス",
        RunE:  projectListCmd.RunE,
    }
)

func init() {
    RootCmd.AddCommand(projectCmd)
    projectCmd.AddCommand(projectListCmd)
    projectCmd.AddCommand(projectBuildCmd)
    projectCmd.AddCommand(projectPullCmd)
    projectCmd.AddCommand(projectUpCmd)
    projectCmd.AddCommand(projectDownCmd)
    projectCmd.AddCommand(projectStartCmd)
    projectCmd.AddCommand(projectStopCmd)
    projectCmd.AddCommand(projectRestartCmd)
    projectCmd.AddCommand(projectPsCmd)
    projectCmd.AddCommand(projectLogsCmd)
    projectCmd.AddCommand(projectExecCmd)
    projectCmd.AddCommand(projectRunCmd)
    projectCmd.AddCommand(projectBackupCmd)
    projectCmd.AddCommand(projectRestoreCmd)

    // トップレベルエイリアスをルートに追加
    RootCmd.AddCommand(topBuildCmd)
    RootCmd.AddCommand(topPullCmd)
    RootCmd.AddCommand(topUpCmd)
    RootCmd.AddCommand(topDownCmd)
    RootCmd.AddCommand(topStartCmd)
    RootCmd.AddCommand(topStopCmd)
    RootCmd.AddCommand(topRestartCmd)
    RootCmd.AddCommand(topLogsCmd)
    RootCmd.AddCommand(topRunCmd)
    RootCmd.AddCommand(topListCmd)

    // フラグ定義
    buildFlags := []*cobra.Command{projectBuildCmd, topBuildCmd}
    for _, f := range buildFlags {
        f.Flags().BoolVar(&projectNoCache, "no-cache", false, "キャッシュを使用せずに最初からイメージをビルド")
        f.Flags().BoolVar(&projectPull, "pull", false, "常に最新のベースイメージをプルしてビルド")
    }

    flags := []*cobra.Command{projectUpCmd, topUpCmd}
    for _, f := range flags {
        f.Flags().BoolVarP(&projectBuild, "build", "b", false, "起動前にイメージをビルド")
    }

    downFlags := []*cobra.Command{projectDownCmd, topDownCmd}
    for _, f := range downFlags {
        f.Flags().BoolVarP(&projectRemoveVolumes, "volumes", "V", false, "名前付きボリュームも同時に削除")
    }

    logFlags := []*cobra.Command{projectLogsCmd, topLogsCmd}
    for _, f := range logFlags {
        f.Flags().BoolVarP(&projectFollowLogs, "follow", "f", false, "ログ出力をリアルタイム追跡")
        f.Flags().StringVarP(&projectTailLogs, "tail", "t", "100", "末尾から表示する行数")
        f.Flags().StringSliceVar(&projectComposeFiles, "file", nil, "追加または上書きで適用する Compose ファイル (例: compose.prod.yml)")
    }

    // Compose ファイルの追加・上書きフラグ (-f / --file)
    composeFileFlags := []*cobra.Command{
        projectBuildCmd, topBuildCmd,
        projectPullCmd, topPullCmd,
        projectUpCmd, topUpCmd,
        projectDownCmd, topDownCmd,
        projectStartCmd, topStartCmd,
        projectStopCmd, topStopCmd,
        projectRestartCmd, topRestartCmd,
        projectPsCmd,
        projectExecCmd,
        projectBackupCmd,
        projectRestoreCmd,
    }
    for _, f := range composeFileFlags {
        f.Flags().StringSliceVarP(&projectComposeFiles, "file", "f", nil, "追加または上書きで適用する Compose ファイル (例: compose.prod.yml)")
    }

    projectBackupCmd.Flags().StringVarP(&projectBackupOutput, "output", "o", "", "バックアップアーカイブの出力先ファイルパス (省略時: <workdir>/.devctl/backups/<project>-<timestamp>.tar.gz)")
    projectRestoreCmd.Flags().BoolVarP(&projectRestoreYes, "yes", "y", false, "上書き確認プロンプトをスキップして実行")
}
