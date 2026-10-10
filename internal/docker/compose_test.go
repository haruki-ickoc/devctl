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
