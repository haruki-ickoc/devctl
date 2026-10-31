package docker

import (
    "fmt"
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

// isValidBuildEnv は BUILD_ENV の値が安全な環境識別子（英数字・ハイフン・アンダースコア）かどうかを判定します。
func isValidBuildEnv(val string) bool {
    if val == "" {
        return false
    }
    for _, r := range val {
        if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
            return false
        }
    }
    return true
}

// parseBuildEnvFromContent は .env ファイルの内容から BUILD_ENV の値を抽出します。
func parseBuildEnvFromContent(content string) string {
    var result string
    for _, rawLine := range strings.Split(content, "\n") {
        line := strings.TrimSpace(rawLine)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        if strings.HasPrefix(line, "export ") {
            line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
        }
        idx := strings.IndexByte(line, '=')
        if idx < 0 {
            continue
        }
        key := strings.TrimSpace(line[:idx])
        if key != "BUILD_ENV" {
            continue
        }
        val := strings.TrimSpace(line[idx+1:])
        if commentIdx := strings.Index(val, " #"); commentIdx >= 0 {
            val = strings.TrimSpace(val[:commentIdx])
        }
        if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
            val = val[1 : len(val)-1]
        }
        val = strings.TrimSpace(val)
        if isValidBuildEnv(val) {
            result = val
        }
    }
    return result
}

// resolveBuildEnv は対象プロジェクトの .env ファイル群から BUILD_ENV を読み取ります。
func resolveBuildEnv(workDir, envFile, baseFile string) string {
    if workDir == "" {
        return ""
    }

    var candidates []string
    addCandidate := func(p string) {
        if p == "" {
            return
        }
        full := p
        if !filepath.IsAbs(full) {
            full = filepath.Join(workDir, p)
        }
        for _, existing := range candidates {
            if existing == full {
                return
            }
        }
        candidates = append(candidates, full)
    }

    if envFile != "" {
        addCandidate(envFile)
    }
    addCandidate(".env")
    if baseFile != "" {
        baseDir := filepath.Dir(baseFile)
        if baseDir != "." && baseDir != "" {
            addCandidate(filepath.Join(baseDir, ".env"))
        }
    }

    for _, path := range candidates {
        data, err := os.ReadFile(path)
        if err != nil {
            continue
        }
        if val := parseBuildEnvFromContent(string(data)); val != "" {
            return val
        }
    }
    return ""
}

// findEnvOverrideFile はベース Compose ファイルと同じディレクトリ内で BUILD_ENV に対応する override ファイルを探索します。
func findEnvOverrideFile(workDir, baseFile, buildEnv string) string {
    if workDir == "" || baseFile == "" || !isValidBuildEnv(buildEnv) {
        return ""
    }

    baseDir := filepath.Dir(baseFile)
    baseName := filepath.Base(baseFile)

    candidateMap := map[string][]string{
        "compose.yaml":        {"compose." + buildEnv + ".yaml", "compose." + buildEnv + ".yml"},
        "compose.yml":         {"compose." + buildEnv + ".yml", "compose." + buildEnv + ".yaml"},
        "docker-compose.yaml": {"docker-compose." + buildEnv + ".yaml", "docker-compose." + buildEnv + ".yml"},
        "docker-compose.yml":  {"docker-compose." + buildEnv + ".yml", "docker-compose." + buildEnv + ".yaml"},
    }

    var candidates []string
    if list, ok := candidateMap[baseName]; ok {
        candidates = append(candidates, list...)
    }
    fallbacks := []string{
        "compose." + buildEnv + ".yaml",
        "compose." + buildEnv + ".yml",
        "docker-compose." + buildEnv + ".yaml",
        "docker-compose." + buildEnv + ".yml",
    }
    for _, fb := range fallbacks {
        exists := false
        for _, c := range candidates {
            if c == fb {
                exists = true
                break
            }
        }
        if !exists {
            candidates = append(candidates, fb)
        }
    }

    searchDirs := []string{baseDir}
    if baseDir == "." || baseDir == "" {
        searchDirs = append(searchDirs, "_docker")
    } else {
        searchDirs = append(searchDirs, ".")
    }

    for _, dir := range searchDirs {
        for _, c := range candidates {
            relPath := c
            if dir != "." && dir != "" {
                relPath = filepath.Join(dir, c)
            }
            fullPath := relPath
            if !filepath.IsAbs(fullPath) {
                fullPath = filepath.Join(workDir, relPath)
            }
            if _, err := os.Stat(fullPath); err == nil {
                return relPath
            }
        }
    }
    return ""
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
        var baseFile string
        for _, f := range opts.ComposeFiles {
            if !isOverrideFile(f) {
                baseFile = f
                break
            }
        }

        buildEnv := resolveBuildEnv(opts.WorkDir, opts.EnvFile, baseFile)
        if buildEnv != "" {
            // .env に BUILD_ENV が指定されている場合:
            // ベースファイルから標準の .override. を除外し、BUILD_ENV に対応する override のみを適用
            envOverride := findEnvOverrideFile(opts.WorkDir, baseFile, buildEnv)
            hasEnvOverride := false
            for _, f := range opts.ComposeFiles {
                if isOverrideFile(f) {
                    continue
                }
                if envOverride != "" && f == envOverride {
                    hasEnvOverride = true
                }
                args = append(args, "-f", f)
            }
            if envOverride != "" && !hasEnvOverride {
                args = append(args, "-f", envOverride)
            }
        } else {
            // CustomComposeFiles も BUILD_ENV も未指定の場合（通常時）:
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
    }

    if opts.EnvFile != "" {
        args = append(args, "--env-file", opts.EnvFile)
    }

    return bin, args
}

// Build は指定されたサービスのコンテナイメージをビルドします（コンテナは起動しません）。
func (c *ComposeClient) Build(opts ComposeOptions, noCache, pull bool) error {
    bin, baseArgs := c.buildBaseArgs(opts)
    args := append(baseArgs, "build")
    if noCache {
        args = append(args, "--no-cache")
    }
    if pull {
        args = append(args, "--pull")
    }
    args = append(args, opts.Services...)
    args = append(args, opts.ExtraArgs...)

    ui.Step("%s でイメージをビルド中...", opts.WorkDir)
    return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Pull は指定されたサービスのコンテナイメージを取得します（コンテナの起動・再作成は行いません）。
func (c *ComposeClient) Pull(opts ComposeOptions) error {
    bin, baseArgs := c.buildBaseArgs(opts)
    args := append(baseArgs, "pull")
    args = append(args, opts.Services...)
    args = append(args, opts.ExtraArgs...)

    ui.Step("%s でイメージを取得中...", opts.WorkDir)
    return c.runner.RunCommand(opts.WorkDir, bin, args...)
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

// Start は停止済み（Exited）のコンテナを開始します（新規作成は行いません）。
func (c *ComposeClient) Start(opts ComposeOptions) error {
    bin, baseArgs := c.buildBaseArgs(opts)
    args := append(baseArgs, "start")
    args = append(args, opts.Services...)
    args = append(args, opts.ExtraArgs...)

    ui.Step("%s で停止済みコンテナを開始中...", opts.WorkDir)
    return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Stop は稼働中のコンテナを一時停止します（コンテナの削除は行いません）。
func (c *ComposeClient) Stop(opts ComposeOptions) error {
    bin, baseArgs := c.buildBaseArgs(opts)
    args := append(baseArgs, "stop")
    args = append(args, opts.Services...)
    args = append(args, opts.ExtraArgs...)

    ui.Step("%s で稼働中コンテナを一時停止中...", opts.WorkDir)
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

// RunShellTasks は設定ファイルで定義された1つまたは複数のカスタムタスクコマンドを定義順に実行します。
// 途中でコマンドが失敗した場合は後続のコマンドを実行せず即座に中断（Fail-Fast）します。
func (c *ComposeClient) RunShellTasks(workDir string, commands []string) error {
    if len(commands) == 0 {
        return fmt.Errorf("実行するコマンドが定義されていません")
    }
    if len(commands) == 1 {
        return c.RunShellTask(workDir, commands[0])
    }

    total := len(commands)
    for i, cmdStr := range commands {
        ui.Step("[%d/%d] カスタムタスクを実行中: %s", i+1, total, cmdStr)
        if err := c.runner.RunShell(workDir, cmdStr); err != nil {
            return fmt.Errorf("タスクのステップ [%d/%d] (%s) の実行に失敗しました: %w", i+1, total, cmdStr, err)
        }
    }
    return nil
}

// findMissingServices は要求されたサービス一覧のうち、稼働中サービス一覧に含まれていないものを抽出します。
func findMissingServices(required, running []string) []string {
    runningSet := make(map[string]bool, len(running))
    for _, s := range running {
        trimmed := strings.TrimSpace(s)
        if trimmed != "" {
            runningSet[trimmed] = true
        }
    }

    var missing []string
    for _, req := range required {
        if !runningSet[req] {
            missing = append(missing, req)
        }
    }
    return missing
}

// RunningServices は指定された Compose 環境で現在 running 状態のサービス名一覧を取得します。
func (c *ComposeClient) RunningServices(opts ComposeOptions) ([]string, error) {
    if c.runner.DryRun {
        ui.Dim("[dry-run] 共通基盤サービスの稼働状態確認シミュレーション（起動プレビューのため未起動と仮定）")
        return nil, nil
    }

    bin, baseArgs := c.buildBaseArgs(opts)
    args := append(baseArgs, "ps", "--services", "--status", "running")

    out, err := c.runner.RunCommandOutput(opts.WorkDir, bin, args...)
    if err != nil {
        return nil, fmt.Errorf("サービスの稼働状態確認に失敗しました: %w", err)
    }

    if out == "" {
        return nil, nil
    }

    var running []string
    for _, line := range strings.Split(out, "\n") {
        s := strings.TrimSpace(line)
        if s != "" {
            running = append(running, s)
        }
    }
    return running, nil
}

// EnsureServicesRunning は opts.Services で指定されたサービスが稼働中か検証し、停止しているサービスのみを自動起動します。
func (c *ComposeClient) EnsureServicesRunning(opts ComposeOptions) error {
    if len(opts.Services) == 0 {
        return nil
    }

    running, err := c.RunningServices(opts)
    if err != nil {
        return err
    }

    missing := findMissingServices(opts.Services, running)
    if len(missing) == 0 {
        ui.Success("共通基盤サービス (%s) は既に稼働中です。", strings.Join(opts.Services, ", "))
        return nil
    }

    ui.Info("共通基盤サービス [%s] が停止しているため自動起動します...", strings.Join(missing, ", "))
    upOpts := opts
    upOpts.Services = missing
    return c.Up(upOpts, false)
}
