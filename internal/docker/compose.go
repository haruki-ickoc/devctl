package docker

import (
    "os"
    "path/filepath"
    "strings"

    "github.com/haruki-ickoc/devctl/internal/ui"
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
    WorkDir            string
    ComposeFiles       []string
    CustomComposeFiles []string // CLI の --file / -f フラグ等で指定された追加・上書きファイル
    EnvFile            string
    Services           []string
    ExtraArgs          []string
}

// isOverrideFile はファイルパスが Compose override ファイルかどうかを判定します。
func isOverrideFile(filePath string) bool {
    base := filepath.Base(filePath)
    return strings.Contains(base, ".override.")
}

// findOverrideFile は作業ディレクトリ内でベース Compose ファイルに対応する override ファイルを探索します。
func findOverrideFile(workDir, baseFile string) string {
    if workDir == "" || baseFile == "" {
        return ""
    }
    baseName := filepath.Base(baseFile)
    candidateMap := map[string][]string{
        "compose.yaml":        {"compose.override.yaml", "compose.override.yml"},
        "compose.yml":         {"compose.override.yml", "compose.override.yaml"},
        "docker-compose.yaml": {"docker-compose.override.yaml", "docker-compose.override.yml"},
        "docker-compose.yml":  {"docker-compose.override.yml", "docker-compose.override.yaml"},
    }
    if list, ok := candidateMap[baseName]; ok {
        for _, c := range list {
            p := filepath.Join(workDir, c)
            if _, err := os.Stat(p); err == nil {
                return c
            }
        }
    }
    fallbacks := []string{
        "compose.override.yml",
        "compose.override.yaml",
        "docker-compose.override.yml",
        "docker-compose.override.yaml",
    }
    for _, fb := range fallbacks {
        p := filepath.Join(workDir, fb)
        if _, err := os.Stat(p); err == nil {
            return fb
        }
    }
    return ""
}

// buildBaseArgs は docker compose の基本コマンドと引数リストを構築します。
// CLI から CustomComposeFiles が指定された場合はベースファイルにそれらを重ね、
// 未指定の場合は通常通り作業ディレクトリ内の override ファイルを自動マージします。
func (c *ComposeClient) buildBaseArgs(opts ComposeOptions) (string, []string) {
    cmdParts := strings.Fields(c.composeCmd)
    bin := cmdParts[0]
    args := cmdParts[1:]

    if len(opts.CustomComposeFiles) > 0 {
        // CLI から CustomComposeFiles が指定されている場合:
        // ベースファイルから開発用 override を除外した上で、指定ファイルを順に付加
        for _, f := range opts.ComposeFiles {
            if isOverrideFile(f) {
                continue
            }
            args = append(args, "-f", f)
        }
        for _, f := range opts.CustomComposeFiles {
            args = append(args, "-f", f)
        }
    } else {
        // CustomComposeFiles が未指定の場合（通常時）:
        // 既存の ComposeFiles を適用し、override ファイルが含まれていなければ自動検出して追加
        hasOverride := false
        for _, f := range opts.ComposeFiles {
            if isOverrideFile(f) {
                hasOverride = true
            }
            args = append(args, "-f", f)
        }
        if !hasOverride && len(opts.ComposeFiles) > 0 && opts.WorkDir != "" {
            if ov := findOverrideFile(opts.WorkDir, opts.ComposeFiles[0]); ov != "" {
                args = append(args, "-f", ov)
            }
        }
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
