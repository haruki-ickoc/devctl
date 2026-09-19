package docker

import (
    "fmt"
    "os"
    "os/exec"
    "strings"

    "github.com/skyou/devctl/internal/ui"
)

// Runner は OS/Docker コマンドの実行を担当する構造体です。
// DryRun モードおよび Verbose モードを制御します。
type Runner struct {
    DryRun  bool
    Verbose bool
}

// NewRunner は新しい Runner インスタンスを生成します。
func NewRunner(dryRun, verbose bool) *Runner {
    return &Runner{
        DryRun:  dryRun,
        Verbose: verbose,
    }
}

// RunCommand は指定された作業ディレクトリ内で外部コマンドを実行し、標準出力・標準エラー出力をストリーミングします。
func (r *Runner) RunCommand(dir string, name string, args ...string) error {
    fullCmd := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
    if dir != "" {
        ui.Dim("[実行] (作業ディレクトリ: %s): %s", dir, fullCmd)
    } else {
        ui.Dim("[実行]: %s", fullCmd)
    }

    if r.DryRun {
        ui.Info("[dry-run] 実行をシミュレーション（スキップ）: %s", fullCmd)
        return nil
    }

    cmd := exec.Command(name, args...)
    if dir != "" {
        cmd.Dir = dir
    }
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin

    return cmd.Run()
}

// RunShell は指定された作業ディレクトリ内でシェルコマンド文字列（sh -c）を実行します。
func (r *Runner) RunShell(dir, shellCmd string) error {
    if dir != "" {
        ui.Dim("[シェル] (作業ディレクトリ: %s): %s", dir, shellCmd)
    } else {
        ui.Dim("[シェル]: %s", shellCmd)
    }

    if r.DryRun {
        ui.Info("[dry-run] シェル実行をシミュレーション（スキップ）: %s", shellCmd)
        return nil
    }

    // パイプや複合コマンドを実行するため sh -c を使用
    cmd := exec.Command("sh", "-c", shellCmd)
    if dir != "" {
        cmd.Dir = dir
    }
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin

    return cmd.Run()
}

// RunCommandOutput はコマンドを実行し、その標準出力を取得して返します。
func (r *Runner) RunCommandOutput(dir string, name string, args ...string) (string, error) {
    if r.DryRun {
        ui.Dim("[dry-run 取得]: %s %s", name, strings.Join(args, " "))
        return "", nil
    }
    cmd := exec.Command(name, args...)
    if dir != "" {
        cmd.Dir = dir
    }
    out, err := cmd.CombinedOutput()
    return strings.TrimSpace(string(out)), err
}
