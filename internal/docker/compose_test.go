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

    t.Run(".envのBUILD_ENV=localに応じて_docker/compose.local.yamlが自動適用されること", func(t *testing.T) {
        tempDir := t.TempDir()
        dockerDir := filepath.Join(tempDir, "_docker")
        if err := os.MkdirAll(dockerDir, 0755); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte("APP_NAME=petfood\nBUILD_ENV=local\n"), 0644); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(dockerDir, "compose.yaml"), []byte(""), 0644); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(dockerDir, "compose.local.yaml"), []byte(""), 0644); err != nil {
            t.Fatal(err)
        }

        opts := ComposeOptions{
            WorkDir:      tempDir,
            ComposeFiles: []string{"_docker/compose.yaml"},
        }
        _, args := client.buildBaseArgs(opts)
        expected := []string{"compose", "-f", "_docker/compose.yaml", "-f", "_docker/compose.local.yaml"}
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("引数の不一致: 期待値 %v, 実際 %v", expected, args)
        }
    })

    t.Run("BUILD_ENVが異なる環境のoverrideや標準compose.override.ymlを誤って適用しないこと", func(t *testing.T) {
        tempDir := t.TempDir()
        if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte("BUILD_ENV=staging\n"), 0644); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(tempDir, "compose.yaml"), []byte(""), 0644); err != nil {
            t.Fatal(err)
        }
        // local 用および標準 override が存在していても、staging 用がなければ適用されないこと
        if err := os.WriteFile(filepath.Join(tempDir, "compose.local.yaml"), []byte(""), 0644); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(tempDir, "compose.override.yaml"), []byte(""), 0644); err != nil {
            t.Fatal(err)
        }

        opts := ComposeOptions{
            WorkDir:      tempDir,
            ComposeFiles: []string{"compose.yaml", "compose.override.yaml"},
        }
        _, args := client.buildBaseArgs(opts)
        expected := []string{"compose", "-f", "compose.yaml"}
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("異なる環境のoverrideが誤適用されました: 期待値 %v, 実際 %v", expected, args)
        }
    })

    t.Run("BUILD_ENVが設定されていても--file指定時は指定ファイルが優先されること", func(t *testing.T) {
        tempDir := t.TempDir()
        if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte("BUILD_ENV=local\n"), 0644); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(tempDir, "compose.yaml"), []byte(""), 0644); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(tempDir, "compose.local.yaml"), []byte(""), 0644); err != nil {
            t.Fatal(err)
        }

        opts := ComposeOptions{
            WorkDir:            tempDir,
            ComposeFiles:       []string{"compose.yaml"},
            CustomComposeFiles: []string{"compose.prod.yaml"},
        }
        _, args := client.buildBaseArgs(opts)
        expected := []string{"compose", "-f", "compose.yaml", "-f", "compose.prod.yaml"}
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("引数の不一致: 期待値 %v, 実際 %v", expected, args)
        }
    })

    t.Run("opts.EnvFileに定義されたBUILD_ENVも読み取れること", func(t *testing.T) {
        tempDir := t.TempDir()
        dockerDir := filepath.Join(tempDir, "_docker")
        if err := os.MkdirAll(dockerDir, 0755); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(dockerDir, ".env"), []byte("export BUILD_ENV=\"local\" # comment\n"), 0644); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(tempDir, "compose.yml"), []byte(""), 0644); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(tempDir, "compose.local.yml"), []byte(""), 0644); err != nil {
            t.Fatal(err)
        }

        opts := ComposeOptions{
            WorkDir:      tempDir,
            ComposeFiles: []string{"compose.yml"},
            EnvFile:      "_docker/.env",
        }
        _, args := client.buildBaseArgs(opts)
        expected := []string{"compose", "-f", "compose.yml", "-f", "compose.local.yml", "--env-file", "_docker/.env"}
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

func TestComposeClient_Pull(t *testing.T) {
    runner := NewRunner(true, false) // DryRun: true
    client := NewComposeClient(runner, "docker compose")

    t.Run("通常のPull実行がDryRunで正常終了すること", func(t *testing.T) {
        opts := ComposeOptions{
            WorkDir:      t.TempDir(),
            ComposeFiles: []string{"compose.yml"},
            EnvFile:      ".env",
        }
        if err := client.Pull(opts); err != nil {
            t.Errorf("Pullの実行に失敗しました: %v", err)
        }
    })

    t.Run("サービス指定およびカスタムComposeファイル指定時のPullがDryRunで正常終了すること", func(t *testing.T) {
        opts := ComposeOptions{
            WorkDir:            t.TempDir(),
            ComposeFiles:       []string{"compose.yml"},
            CustomComposeFiles: []string{"compose.prod.yml"},
            EnvFile:            ".env.prod",
            Services:           []string{"app", "db"},
        }
        if err := client.Pull(opts); err != nil {
            t.Errorf("Pullの実行に失敗しました: %v", err)
        }
    })

    t.Run("コマンド実行失敗時にエラーが伝播すること", func(t *testing.T) {
        failRunner := NewRunner(false, false)
        failClient := NewComposeClient(failRunner, "false")
        opts := ComposeOptions{
            WorkDir:      t.TempDir(),
            ComposeFiles: []string{"compose.yml"},
        }
        if err := failClient.Pull(opts); err == nil {
            t.Errorf("コマンド失敗時にエラーが返されませんでした")
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

func TestComposeClient_RunShellTasks(t *testing.T) {
    t.Run("複数コマンドが定義順に実行されること", func(t *testing.T) {
        runner := NewRunner(false, false)
        client := NewComposeClient(runner, "docker compose")
        tempDir := t.TempDir()
        outFile := filepath.Join(tempDir, "order.txt")

        cmds := []string{
            "echo step1 >> " + outFile,
            "STEP_VAR=step2 sh -c 'echo $STEP_VAR >> " + outFile + "'",
            "echo step3 >> " + outFile,
        }
        if err := client.RunShellTasks(tempDir, cmds); err != nil {
            t.Fatalf("RunShellTasks の実行に失敗しました: %v", err)
        }

        data, err := os.ReadFile(outFile)
        if err != nil {
            t.Fatalf("結果ファイルの読み込みに失敗しました: %v", err)
        }
        expected := "step1\nstep2\nstep3\n"
        if string(data) != expected {
            t.Errorf("実行順序の不一致: 期待値 %q, 実際 %q", expected, string(data))
        }
    })

    t.Run("途中のコマンドが失敗した場合は即時中断(Fail-Fast)し後続コマンドが実行されないこと", func(t *testing.T) {
        runner := NewRunner(false, false)
        client := NewComposeClient(runner, "docker compose")
        tempDir := t.TempDir()
        outFile := filepath.Join(tempDir, "failfast.txt")

        cmds := []string{
            "echo first >> " + outFile,
            "exit 1",
            "echo third >> " + outFile,
        }
        err := client.RunShellTasks(tempDir, cmds)
        if err == nil {
            t.Fatalf("途中コマンドの失敗時にエラーが返されませんでした")
        }

        data, readErr := os.ReadFile(outFile)
        if readErr != nil {
            t.Fatalf("結果ファイルの読み込みに失敗しました: %v", readErr)
        }
        expected := "first\n"
        if string(data) != expected {
            t.Errorf("Fail-Fast 検証失敗: 後続コマンドが実行された可能性があります。期待値 %q, 実際 %q", expected, string(data))
        }
    })
}
