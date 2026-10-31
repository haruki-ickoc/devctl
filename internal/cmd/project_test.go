package cmd

import (
    "os"
    "path/filepath"
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

func TestPrepareDependencies_WithCoreAndNetwork(t *testing.T) {
    dryRun = true
    verbose = false
    runner = docker.NewRunner(true, false)
    composeClient = docker.NewComposeClient(runner, "docker compose")
    networkManager = docker.NewNetworkManager(runner)

    coreDir := t.TempDir()
    projDir := t.TempDir()

    cfg = &config.Config{
        Defaults: config.DefaultsConfig{
            ComposeCmd: "docker compose",
        },
        Network: config.NetworkConfig{
            Name:       "dev-network",
            Driver:     "bridge",
            Attachable: true,
        },
        Core: config.CoreConfig{
            WorkDir:      coreDir,
            ComposeFiles: []string{"compose.yml"},
        },
        Projects: map[string]config.ProjectItem{
            "dep-app": {
                Name:         "dep-app",
                WorkDir:      projDir,
                ComposeFiles: []string{"compose.yml"},
                DependsOn: config.DependsOnConfig{
                    Network: true,
                    Core:    []string{"traefik", "dbgate"},
                },
            },
        },
    }

    t.Run("up/start/restartで依存ネットワークとCoreサービスの生存検証・自動起動が動作すること", func(t *testing.T) {
        projectComposeFiles = nil
        projectBuild = false

        if err := projectUpCmd.RunE(projectUpCmd, []string{"dep-app"}); err != nil {
            t.Fatalf("projectUpCmd (with deps) failed: %v", err)
        }
        if err := projectStartCmd.RunE(projectStartCmd, []string{"dep-app"}); err != nil {
            t.Fatalf("projectStartCmd (with deps) failed: %v", err)
        }
        if err := projectRestartCmd.RunE(projectRestartCmd, []string{"dep-app"}); err != nil {
            t.Fatalf("projectRestartCmd (with deps) failed: %v", err)
        }
    })

    t.Run("core.workdirが未設定の場合はエラーを返すこと", func(t *testing.T) {
        cfg.Core.WorkDir = ""
        err := projectUpCmd.RunE(projectUpCmd, []string{"dep-app"})
        if err == nil {
            t.Errorf("core.workdir 未設定時にエラーが発生しませんでした")
        }
    })
}

func TestPullCmd_FlagsAndRegistration(t *testing.T) {
    // RootCmd への pull 短縮コマンド登録確認
    foundTop := false
    for _, c := range RootCmd.Commands() {
        if c.Name() == "pull" {
            foundTop = true
            break
        }
    }
    if !foundTop {
        t.Errorf("RootCmd に 'pull' コマンドが登録されていません")
    }

    // projectCmd 配下の pull 登録確認
    foundProjectPull := false
    for _, c := range projectCmd.Commands() {
        if c.Name() == "pull" {
            foundProjectPull = true
            break
        }
    }
    if !foundProjectPull {
        t.Errorf("projectCmd に 'pull' サブコマンドが登録されていません")
    }

    // coreCmd 配下の pull 登録確認
    foundCorePull := false
    for _, c := range coreCmd.Commands() {
        if c.Name() == "pull" {
            foundCorePull = true
            break
        }
    }
    if !foundCorePull {
        t.Errorf("coreCmd に 'pull' サブコマンドが登録されていません")
    }

    // projectPullCmd / topPullCmd の -f / --file フラグ確認
    for _, c := range []*cobra.Command{projectPullCmd, topPullCmd} {
        if c.Flag("file") == nil {
            t.Errorf("コマンド '%s' に -f/--file フラグが存在しません", c.Name())
        }
    }
}

func TestPullCmd_Execution(t *testing.T) {
    dryRun = true
    verbose = false
    runner = docker.NewRunner(true, false)
    composeClient = docker.NewComposeClient(runner, "docker compose")

    coreDir := t.TempDir()
    projDir := t.TempDir()

    cfg = &config.Config{
        Defaults: config.DefaultsConfig{
            ComposeCmd: "docker compose",
        },
        Core: config.CoreConfig{
            WorkDir:      coreDir,
            ComposeFiles: []string{"compose.yml"},
            EnvFile:      ".env",
        },
        Projects: map[string]config.ProjectItem{
            "test-app": {
                Name:         "test-app",
                WorkDir:      projDir,
                ComposeFiles: []string{"compose.yml"},
                EnvFile:      "_docker/.env",
            },
        },
    }

    t.Run("プロジェクト指定で全サービスまたは指定サービスのPullがDryRun実行できること", func(t *testing.T) {
        projectComposeFiles = nil
        if err := projectPullCmd.RunE(projectPullCmd, []string{"test-app"}); err != nil {
            t.Fatalf("projectPullCmd の実行に失敗しました: %v", err)
        }

        projectComposeFiles = []string{"compose.prod.yml"}
        if err := topPullCmd.RunE(topPullCmd, []string{"test-app", "app", "db"}); err != nil {
            t.Fatalf("topPullCmd の実行に失敗しました: %v", err)
        }
    })

    t.Run("Coreで全サービスおよび指定サービスのPullがDryRun実行できること", func(t *testing.T) {
        if err := corePullCmd.RunE(corePullCmd, nil); err != nil {
            t.Fatalf("corePullCmd (全サービス) の実行に失敗しました: %v", err)
        }
        if err := corePullCmd.RunE(corePullCmd, []string{"traefik", "dbgate"}); err != nil {
            t.Fatalf("corePullCmd (サービス指定) の実行に失敗しました: %v", err)
        }
    })

    t.Run("存在しないプロジェクトまたはcore.workdir未設定時にエラーになること", func(t *testing.T) {
        if err := topPullCmd.RunE(topPullCmd, []string{"unknown-project"}); err == nil {
            t.Errorf("存在しないプロジェクトでエラーが発生しませんでした")
        }

        cfg.Core.WorkDir = ""
        if err := corePullCmd.RunE(corePullCmd, nil); err == nil {
            t.Errorf("core.workdir 未設定時にエラーが発生しませんでした")
        }
    })
}

func TestRunCmd_MultiCommandExecution(t *testing.T) {
    dryRun = true
    verbose = false
    runner = docker.NewRunner(true, false)
    composeClient = docker.NewComposeClient(runner, "docker compose")

    projDir := t.TempDir()
    cfg = &config.Config{
        Defaults: config.DefaultsConfig{
            ComposeCmd: "docker compose",
        },
        Projects: map[string]config.ProjectItem{
            "test-app": {
                Name:         "test-app",
                WorkDir:      projDir,
                ComposeFiles: []string{"compose.yml"},
                Tasks: map[string]config.TaskItem{
                    "single": {
                        Description: "単一タスク",
                        Command:     "echo one",
                    },
                    "multi": {
                        Commands: []string{"echo one", "echo two", "echo three"},
                    },
                    "empty": {},
                },
            },
        },
    }

    t.Run("単一コマンドおよび複数コマンドのタスクが正常にDryRun実行できること", func(t *testing.T) {
        if err := projectRunCmd.RunE(projectRunCmd, []string{"test-app", "single"}); err != nil {
            t.Fatalf("単一コマンドタスクの実行に失敗しました: %v", err)
        }
        if err := topRunCmd.RunE(topRunCmd, []string{"test-app", "multi"}); err != nil {
            t.Fatalf("複数コマンドタスクの実行に失敗しました: %v", err)
        }
    })

    t.Run("空タスクや未定義タスクでエラーになること", func(t *testing.T) {
        if err := topRunCmd.RunE(topRunCmd, []string{"test-app", "empty"}); err == nil {
            t.Errorf("空タスクでエラーが発生しませんでした")
        }
        if err := topRunCmd.RunE(topRunCmd, []string{"test-app", "not-found"}); err == nil {
            t.Errorf("未定義タスクでエラーが発生しませんでした")
        }
    })
}

func TestUpCmd_WithBuildEnvOverride(t *testing.T) {
    dryRun = true
    verbose = false
    runner = docker.NewRunner(true, false)
    composeClient = docker.NewComposeClient(runner, "docker compose")

    projDir := t.TempDir()
    dockerDir := filepath.Join(projDir, "_docker")
    if err := os.MkdirAll(dockerDir, 0755); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(filepath.Join(projDir, ".env"), []byte("BUILD_ENV=local\n"), 0644); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(filepath.Join(dockerDir, "compose.yaml"), []byte(""), 0644); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(filepath.Join(dockerDir, "compose.local.yaml"), []byte(""), 0644); err != nil {
        t.Fatal(err)
    }

    cfg = &config.Config{
        Defaults: config.DefaultsConfig{
            ComposeCmd: "docker compose",
        },
        Projects: map[string]config.ProjectItem{
            "petfood": {
                Name:         "petfood",
                WorkDir:      projDir,
                ComposeFiles: []string{"_docker/compose.yaml"},
            },
        },
    }

    projectComposeFiles = nil
    projectBuild = false
    if err := topUpCmd.RunE(topUpCmd, []string{"petfood"}); err != nil {
        t.Fatalf("BUILD_ENV=local 環境での devctl up petfood に失敗しました: %v", err)
    }
}
