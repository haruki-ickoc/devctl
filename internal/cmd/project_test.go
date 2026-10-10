package cmd

import (
    "testing"

    "github.com/spf13/cobra"
    "github.com/haruki-ickoc/devctl/internal/config"
    "github.com/haruki-ickoc/devctl/internal/docker"
)

func TestBuildCmd_FlagsAndRegistration(t *testing.T) {
    // topBuildCmd の登録確認
    foundTop := false
    for _, c := range RootCmd.Commands() {
        if c.Name() == "build" {
            foundTop = true
            break
        }
    }
    if !foundTop {
        t.Errorf("RootCmd に 'build' コマンドが登録されていません")
    }

    // projectCmd 配下の build 登録確認
    foundProjectBuild := false
    for _, c := range projectCmd.Commands() {
        if c.Name() == "build" {
            foundProjectBuild = true
            break
        }
    }
    if !foundProjectBuild {
        t.Errorf("projectCmd に 'build' サブコマンドが登録されていません")
    }

    // フラグの存在確認
    for _, c := range []*cobra.Command{projectBuildCmd, topBuildCmd} {
        if c.Flag("no-cache") == nil {
            t.Errorf("コマンド '%s' に --no-cache フラグが存在しません", c.Name())
        }
        if c.Flag("pull") == nil {
            t.Errorf("コマンド '%s' に --pull フラグが存在しません", c.Name())
        }
        if c.Flag("file") == nil {
            t.Errorf("コマンド '%s' に -f/--file フラグが存在しません", c.Name())
        }
    }
}

func TestBuildCmd_Execution(t *testing.T) {
    // dry-run モードでテスト設定を初期化
    dryRun = true
    verbose = false
    runner = docker.NewRunner(true, false)
    composeClient = docker.NewComposeClient(runner, "docker compose")

    cfg = &config.Config{
        Defaults: config.DefaultsConfig{
            ComposeCmd: "docker compose",
        },
        Projects: map[string]config.ProjectItem{
            "test-app": {
                Name:         "test-app",
                WorkDir:      t.TempDir(),
                ComposeFiles: []string{"compose.yml"},
            },
        },
    }

    t.Run("プロジェクト指定でBuildがDryRun実行できること", func(t *testing.T) {
        projectNoCache = false
        projectPull = false
        projectComposeFiles = nil

        err := projectBuildCmd.RunE(projectBuildCmd, []string{"test-app"})
        if err != nil {
            t.Fatalf("projectBuildCmd の実行に失敗しました: %v", err)
        }
    })

    t.Run("サービス指定とフラグありでBuildがDryRun実行できること", func(t *testing.T) {
        projectNoCache = true
        projectPull = true
        projectComposeFiles = []string{"compose.prod.yml"}

        err := topBuildCmd.RunE(topBuildCmd, []string{"test-app", "app"})
        if err != nil {
            t.Fatalf("topBuildCmd の実行に失敗しました: %v", err)
        }
    })

    t.Run("存在しないプロジェクトでエラーになること", func(t *testing.T) {
        err := topBuildCmd.RunE(topBuildCmd, []string{"unknown-project"})
        if err == nil {
            t.Errorf("存在しないプロジェクトでエラーが発生しませんでした")
        }
    })
}

func TestStartStopCmd_FlagsAndRegistration(t *testing.T) {
    // RootCmd への登録確認
    for _, target := range []string{"start", "stop"} {
        found := false
        for _, c := range RootCmd.Commands() {
            if c.Name() == target {
                found = true
                break
            }
        }
        if !found {
            t.Errorf("RootCmd に '%s' コマンドが登録されていません", target)
        }
    }

    // projectCmd 配下への登録確認
    for _, target := range []string{"start", "stop"} {
        found := false
        for _, c := range projectCmd.Commands() {
            if c.Name() == target {
                found = true
                break
            }
        }
        if !found {
            t.Errorf("projectCmd に '%s' サブコマンドが登録されていません", target)
        }
    }

    // -f / --file フラグの存在確認
    for _, c := range []*cobra.Command{projectStartCmd, topStartCmd, projectStopCmd, topStopCmd} {
        if c.Flag("file") == nil {
            t.Errorf("コマンド '%s' に -f/--file フラグが存在しません", c.Name())
        }
    }
}

func TestStartStopCmd_Execution(t *testing.T) {
    dryRun = true
    verbose = false
    runner = docker.NewRunner(true, false)
    composeClient = docker.NewComposeClient(runner, "docker compose")

    cfg = &config.Config{
        Defaults: config.DefaultsConfig{
            ComposeCmd: "docker compose",
        },
        Projects: map[string]config.ProjectItem{
            "test-app": {
                Name:         "test-app",
                WorkDir:      t.TempDir(),
                ComposeFiles: []string{"compose.yml"},
            },
        },
    }

    t.Run("プロジェクト指定でStartおよびStopがDryRun実行できること", func(t *testing.T) {
        projectComposeFiles = nil
        if err := projectStartCmd.RunE(projectStartCmd, []string{"test-app"}); err != nil {
            t.Fatalf("projectStartCmd の実行に失敗しました: %v", err)
        }
        if err := projectStopCmd.RunE(projectStopCmd, []string{"test-app"}); err != nil {
            t.Fatalf("projectStopCmd の実行に失敗しました: %v", err)
        }
    })

    t.Run("サービス指定と-fフラグ指定でtopStartCmd/topStopCmdがDryRun実行できること", func(t *testing.T) {
        projectComposeFiles = []string{"compose.prod.yml"}
        if err := topStartCmd.RunE(topStartCmd, []string{"test-app", "worker"}); err != nil {
            t.Fatalf("topStartCmd の実行に失敗しました: %v", err)
        }
        if err := topStopCmd.RunE(topStopCmd, []string{"test-app", "worker"}); err != nil {
            t.Fatalf("topStopCmd の実行に失敗しました: %v", err)
        }
    })

    t.Run("存在しないプロジェクトでエラーになること", func(t *testing.T) {
        if err := topStartCmd.RunE(topStartCmd, []string{"unknown-project"}); err == nil {
            t.Errorf("topStartCmd: 存在しないプロジェクトでエラーが発生しませんでした")
        }
        if err := topStopCmd.RunE(topStopCmd, []string{"unknown-project"}); err == nil {
            t.Errorf("topStopCmd: 存在しないプロジェクトでエラーが発生しませんでした")
        }
    })
}
