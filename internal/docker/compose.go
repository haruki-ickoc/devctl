package docker

import (
	"strings"

	"github.com/skyou/devctl/internal/ui"
)

// ComposeClient は共通基盤およびプロジェクトの Docker Compose コマンド実行を担当します。
type ComposeClient struct {
	runner     *Runner
	composeCmd string
}

// NewComposeClient は新しい Docker Compose クライアントを生成します。
func NewComposeClient(runner *Runner, composeCmd string) *ComposeClient {
	if composeCmd == "" {
		composeCmd = "docker compose"
	}
	return &ComposeClient{
		runner:     runner,
		composeCmd: composeCmd,
	}
}

// ComposeOptions は Compose コマンド実行時のオプションパラメータです。
type ComposeOptions struct {
	WorkDir      string
	ComposeFiles []string
	EnvFile      string
	Services     []string
	ExtraArgs    []string
}

// buildBaseArgs は docker compose の基本コマンドと引数リストを構築します。
func (c *ComposeClient) buildBaseArgs(opts ComposeOptions) (string, []string) {
	cmdParts := strings.Fields(c.composeCmd)
	bin := cmdParts[0]
	args := cmdParts[1:]

	for _, f := range opts.ComposeFiles {
		args = append(args, "-f", f)
	}

	if opts.EnvFile != "" {
		args = append(args, "--env-file", opts.EnvFile)
	}

	return bin, args
}

// Up はサービスをバックグラウンド（デタッチドモード）で起動します。
func (c *ComposeClient) Up(opts ComposeOptions, build bool) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "up", "-d")
	if build {
		args = append(args, "--build")
	}
	args = append(args, opts.Services...)
	args = append(args, opts.ExtraArgs...)

	ui.Step("%s でコンテナを起動中...", opts.WorkDir)
	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Down はコンテナおよびリソースを停止・削除します。
func (c *ComposeClient) Down(opts ComposeOptions, removeVolumes bool) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "down")
	if removeVolumes {
		args = append(args, "-v")
	}
	args = append(args, opts.ExtraArgs...)

	ui.Step("%s でコンテナを停止中...", opts.WorkDir)
	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Restart はサービスを再起動します。
func (c *ComposeClient) Restart(opts ComposeOptions) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "restart")
	args = append(args, opts.Services...)
	args = append(args, opts.ExtraArgs...)

	ui.Step("%s でコンテナを再起動中...", opts.WorkDir)
	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Logs はサービスのログを表示またはリアルタイム追跡します。
func (c *ComposeClient) Logs(opts ComposeOptions, follow bool, tail string) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "logs")
	if follow {
		args = append(args, "-f")
	}
	if tail != "" {
		args = append(args, "--tail", tail)
	}
	args = append(args, opts.Services...)
	args = append(args, opts.ExtraArgs...)

	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Ps は指定された Compose 環境で稼働中のコンテナ一覧を表示します。
func (c *ComposeClient) Ps(opts ComposeOptions) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "ps")
	args = append(args, opts.ExtraArgs...)

	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Exec は稼働中のコンテナ内でコマンドを実行します。
func (c *ComposeClient) Exec(opts ComposeOptions, service string, cmd []string) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "exec", service)
	args = append(args, cmd...)

	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Run はサービス上でワンオフコマンドを実行します。
func (c *ComposeClient) Run(opts ComposeOptions, service string, cmd []string) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "run", "--rm", service)
	args = append(args, cmd...)

	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// RunShellTask は設定ファイルで定義されたカスタムタスク（Makefile代替処理）を実行します。
func (c *ComposeClient) RunShellTask(workDir, shellCommand string) error {
	ui.Step("カスタムタスクを実行中: %s", shellCommand)
	return c.runner.RunShell(workDir, shellCommand)
}
