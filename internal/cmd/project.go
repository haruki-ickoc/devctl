package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/skyou/devctl/internal/config"
	"github.com/skyou/devctl/internal/docker"
	"github.com/skyou/devctl/internal/ui"
)

var (
	projectBuild         bool
	projectRemoveVolumes bool
	projectFollowLogs    bool
	projectTailLogs      string
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

// prepareDependencies はプロジェクトが依存する共通ネットワークや Core サービスの起動状態を事前に確保します。
func prepareDependencies(proj *config.ProjectItem) error {
	if proj.DependsOn.Network {
		if err := networkManager.Ensure(cfg.Network.Name, cfg.Network.Driver, cfg.Network.Attachable); err != nil {
			return fmt.Errorf("%s の共通ネットワーク準備に失敗しました: %w", proj.Name, err)
		}
	}

	if len(proj.DependsOn.Core) > 0 {
		ui.Info("%s が依存する共通基盤サービスを準備中: %v", proj.Name, proj.DependsOn.Core)
		opts := docker.ComposeOptions{
			WorkDir:      cfg.Core.WorkDir,
			ComposeFiles: cfg.Core.ComposeFiles,
			EnvFile:      cfg.Core.EnvFile,
			Services:     proj.DependsOn.Core,
		}
		if err := composeClient.Up(opts, false); err != nil {
			return fmt.Errorf("必要な共通基盤サービスの起動に失敗しました: %w", err)
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
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
			Services:     remainingArgs,
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
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
			Services:     remainingArgs,
		}

		if err := composeClient.Down(opts, projectRemoveVolumes); err != nil {
			return fmt.Errorf("プロジェクト '%s' の停止に失敗しました: %w", proj.Name, err)
		}

		ui.Success("プロジェクト '%s' を停止しました。", proj.Name)
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

		opts := docker.ComposeOptions{
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
			Services:     remainingArgs,
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
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
			Services:     remainingArgs,
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
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
			Services:     remainingArgs,
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
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
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
					fmt.Printf("  %-15s %s\n", tName, tItem.Description)
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
				fmt.Printf("  %-15s %s\n", tName, tItem.Description)
			}
			return fmt.Errorf("タスクが未定義です")
		}

		ui.Info("プロジェクト '%s' でタスク '%s' を実行中...", proj.Name, taskName)
		return composeClient.RunShellTask(proj.WorkDir, task.Command)
	},
}

// 日常の開発体験を快適にするトップレベルエイリアスコマンド群
var (
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
	projectCmd.AddCommand(projectUpCmd)
	projectCmd.AddCommand(projectDownCmd)
	projectCmd.AddCommand(projectRestartCmd)
	projectCmd.AddCommand(projectPsCmd)
	projectCmd.AddCommand(projectLogsCmd)
	projectCmd.AddCommand(projectExecCmd)
	projectCmd.AddCommand(projectRunCmd)

	// トップレベルエイリアスをルートに追加
	RootCmd.AddCommand(topUpCmd)
	RootCmd.AddCommand(topDownCmd)
	RootCmd.AddCommand(topRestartCmd)
	RootCmd.AddCommand(topLogsCmd)
	RootCmd.AddCommand(topRunCmd)
	RootCmd.AddCommand(topListCmd)

	// フラグ定義
	flags := []*cobra.Command{projectUpCmd, topUpCmd}
	for _, f := range flags {
		f.Flags().BoolVarP(&projectBuild, "build", "b", false, "起動前にイメージをビルド")
	}

	downFlags := []*cobra.Command{projectDownCmd, topDownCmd}
	for _, f := range downFlags {
		f.Flags().BoolVarP(&projectRemoveVolumes, "volumes", "v", false, "名前付きボリュームも同時に削除")
	}

	logFlags := []*cobra.Command{projectLogsCmd, topLogsCmd}
	for _, f := range logFlags {
		f.Flags().BoolVarP(&projectFollowLogs, "follow", "f", false, "ログ出力をリアルタイム追跡")
		f.Flags().StringVarP(&projectTailLogs, "tail", "t", "100", "末尾から表示する行数")
	}
}
