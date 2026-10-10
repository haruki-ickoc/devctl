package docker

import (
    "os"
    "path/filepath"
    "reflect"
    "testing"
)

func TestBuildBaseArgs(t *testing.T) {
    client := NewComposeClient(nil, "docker compose")

    t.Run("overrideが存在しない場合はベースファイルのみ渡されること", func(t *testing.T) {
        tempDir := t.TempDir()
        opts := ComposeOptions{
            WorkDir:      tempDir,
            ComposeFiles: []string{"compose.yml"},
        }
        bin, args := client.buildBaseArgs(opts)
        if bin != "docker" {
            t.Errorf("期待されるバイナリは 'docker' ですが、%s でした", bin)
        }
        expected := []string{"compose", "-f", "compose.yml"}
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("引数の不一致: 期待値 %v, 実際 %v", expected, args)
        }
    })

    t.Run("overrideが存在する場合は自動で付加されること", func(t *testing.T) {
        tempDir := t.TempDir()
        ovPath := filepath.Join(tempDir, "compose.override.yml")
        if err := os.WriteFile(ovPath, []byte(""), 0644); err != nil {
            t.Fatal(err)
        }

        opts := ComposeOptions{
            WorkDir:      tempDir,
            ComposeFiles: []string{"compose.yml"},
        }
        _, args := client.buildBaseArgs(opts)
        expected := []string{"compose", "-f", "compose.yml", "-f", "compose.override.yml"}
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("引数の不一致: 期待値 %v, 実際 %v", expected, args)
        }
    })

    t.Run("すでにComposeFilesにoverrideが含まれている場合は二重追加されないこと", func(t *testing.T) {
        tempDir := t.TempDir()
        ovPath := filepath.Join(tempDir, "compose.override.yml")
        if err := os.WriteFile(ovPath, []byte(""), 0644); err != nil {
            t.Fatal(err)
        }

        opts := ComposeOptions{
            WorkDir:      tempDir,
            ComposeFiles: []string{"compose.yml", "compose.override.yml"},
        }
        _, args := client.buildBaseArgs(opts)
        expected := []string{"compose", "-f", "compose.yml", "-f", "compose.override.yml"}
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("引数の不一致: 期待値 %v, 実際 %v", expected, args)
        }
    })

    t.Run("CustomComposeFiles指定時は開発用overrideが除外されカスタムファイルが適用されること", func(t *testing.T) {
        tempDir := t.TempDir()
        ovPath := filepath.Join(tempDir, "compose.override.yml")
        if err := os.WriteFile(ovPath, []byte(""), 0644); err != nil {
            t.Fatal(err)
        }

        opts := ComposeOptions{
            WorkDir:            tempDir,
            ComposeFiles:       []string{"compose.yml", "compose.override.yml"},
            CustomComposeFiles: []string{"compose.prod.yml"},
        }
        _, args := client.buildBaseArgs(opts)
        expected := []string{"compose", "-f", "compose.yml", "-f", "compose.prod.yml"}
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("引数の不一致: 期待値 %v, 実際 %v", expected, args)
        }
    })

    t.Run("EnvFileが指定された場合は--env-fileが付加されること", func(t *testing.T) {
        opts := ComposeOptions{
            ComposeFiles: []string{"compose.yml"},
            EnvFile:      ".env.test",
        }
        _, args := client.buildBaseArgs(opts)
        expected := []string{"compose", "-f", "compose.yml", "--env-file", ".env.test"}
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("引数の不一致: 期待値 %v, 実際 %v", expected, args)
        }
    })
}

func TestComposeClient_Build(t *testing.T) {
    runner := NewRunner(true, false) // DryRun: true
    client := NewComposeClient(runner, "docker compose")

    t.Run("通常のBuild実行がDryRunで正常終了すること", func(t *testing.T) {
        opts := ComposeOptions{
            WorkDir:      t.TempDir(),
            ComposeFiles: []string{"compose.yml"},
        }
        err := client.Build(opts, false, false)
        if err != nil {
            t.Errorf("Buildの実行に失敗しました: %v", err)
        }
    })

    t.Run("サービス指定と各フラグ（no-cache, pull）が指定された場合も正常終了すること", func(t *testing.T) {
        opts := ComposeOptions{
            WorkDir:            t.TempDir(),
            ComposeFiles:       []string{"compose.yml"},
            CustomComposeFiles: []string{"compose.prod.yml"},
            Services:           []string{"app", "web"},
        }
        err := client.Build(opts, true, true)
        if err != nil {
            t.Errorf("Buildの実行に失敗しました: %v", err)
        }
    })
}

func TestComposeClient_StartAndStop(t *testing.T) {
    runner := NewRunner(true, false) // DryRun: true
    client := NewComposeClient(runner, "docker compose")

    t.Run("StartおよびStopがDryRunで正常終了すること", func(t *testing.T) {
        opts := ComposeOptions{
            WorkDir:      t.TempDir(),
            ComposeFiles: []string{"compose.yml"},
        }
        if err := client.Start(opts); err != nil {
            t.Errorf("Startの実行に失敗しました: %v", err)
        }
        if err := client.Stop(opts); err != nil {
            t.Errorf("Stopの実行に失敗しました: %v", err)
        }
    })

    t.Run("サービス指定およびカスタムComposeファイル指定時のStart/Stopが正常終了すること", func(t *testing.T) {
        opts := ComposeOptions{
            WorkDir:            t.TempDir(),
            ComposeFiles:       []string{"compose.yml"},
            CustomComposeFiles: []string{"compose.prod.yml"},
            Services:           []string{"app", "worker"},
        }
        if err := client.Start(opts); err != nil {
            t.Errorf("Startの実行に失敗しました: %v", err)
        }
        if err := client.Stop(opts); err != nil {
            t.Errorf("Stopの実行に失敗しました: %v", err)
        }
    })
}

func TestFindMissingServices(t *testing.T) {
    t.Run("すべてのサービスが稼働中の場合は空リストが返ること", func(t *testing.T) {
        required := []string{"traefik", "dbgate"}
        running := []string{"traefik", "dbgate", "redis"}
        missing := findMissingServices(required, running)
        if len(missing) != 0 {
            t.Errorf("期待値は空ですが、%v が返されました", missing)
        }
    })

    t.Run("一部のサービスが停止中の場合は未起動サービスのみが返ること", func(t *testing.T) {
        required := []string{"traefik", "dbgate"}
        running := []string{"traefik"}
        missing := findMissingServices(required, running)
        expected := []string{"dbgate"}
        if !reflect.DeepEqual(missing, expected) {
            t.Errorf("不一致: 期待値 %v, 実際 %v", expected, missing)
        }
    })

    t.Run("すべて停止中の場合は全要求サービスが返ること", func(t *testing.T) {
        required := []string{"traefik", "dbgate"}
        running := []string{}
        missing := findMissingServices(required, running)
        expected := []string{"traefik", "dbgate"}
        if !reflect.DeepEqual(missing, expected) {
            t.Errorf("不一致: 期待値 %v, 実際 %v", expected, missing)
        }
    })
}

func TestComposeClient_EnsureServicesRunning(t *testing.T) {
    runner := NewRunner(true, false) // DryRun: true
    client := NewComposeClient(runner, "docker compose")

    t.Run("DryRun時に未起動サービスを検知して自動起動シミュレーションが成功すること", func(t *testing.T) {
        opts := ComposeOptions{
            WorkDir:      t.TempDir(),
            ComposeFiles: []string{"compose.yml"},
            Services:     []string{"traefik", "dbgate"},
        }
        if err := client.EnsureServicesRunning(opts); err != nil {
            t.Errorf("EnsureServicesRunning の実行に失敗しました: %v", err)
        }
    })

    t.Run("対象サービスが空の場合は何もせず正常終了すること", func(t *testing.T) {
        opts := ComposeOptions{
            WorkDir:      t.TempDir(),
            ComposeFiles: []string{"compose.yml"},
            Services:     nil,
        }
        if err := client.EnsureServicesRunning(opts); err != nil {
            t.Errorf("EnsureServicesRunning の実行に失敗しました: %v", err)
        }
    })
}
