package config

import (
    "os"
    "path/filepath"
    "reflect"
    "testing"
)

func TestDetectComposeFiles(t *testing.T) {
    tempDir := t.TempDir()

    // 1. ファイルが存在しない場合は compose.yml を返す
    res := detectComposeFiles(tempDir)
    if len(res) != 1 || res[0] != "compose.yml" {
        t.Fatalf("ファイル未存在時は compose.yml が期待されますが、%v でした", res)
    }

    // 2. docker-compose.yml が存在する場合
    dcFile := filepath.Join(tempDir, "docker-compose.yml")
    if err := os.WriteFile(dcFile, []byte(""), 0644); err != nil {
        t.Fatal(err)
    }
    res = detectComposeFiles(tempDir)
    if len(res) != 1 || res[0] != "docker-compose.yml" {
        t.Fatalf("docker-compose.yml の検出を期待しましたが、%v でした", res)
    }

    // 3. より優先度の高い compose.yml が追加された場合
    cFile := filepath.Join(tempDir, "compose.yml")
    if err := os.WriteFile(cFile, []byte(""), 0644); err != nil {
        t.Fatal(err)
    }
    res = detectComposeFiles(tempDir)
    if len(res) != 1 || res[0] != "compose.yml" {
        t.Fatalf("compose.yml の優先検出を期待しましたが、%v でした", res)
    }

    // 4. 最も優先度の高い compose.yaml が追加された場合
    cyFile := filepath.Join(tempDir, "compose.yaml")
    if err := os.WriteFile(cyFile, []byte(""), 0644); err != nil {
        t.Fatal(err)
    }
    res = detectComposeFiles(tempDir)
    if len(res) != 1 || res[0] != "compose.yaml" {
        t.Fatalf("compose.yaml の優先検出を期待しましたが、%v でした", res)
    }

    // 5. compose.override.yml が追加された場合、自動的にマージ検出されること
    ovFile := filepath.Join(tempDir, "compose.override.yml")
    if err := os.WriteFile(ovFile, []byte(""), 0644); err != nil {
        t.Fatal(err)
    }
    res = detectComposeFiles(tempDir)
    if len(res) != 2 || res[0] != "compose.yaml" || res[1] != "compose.override.yml" {
        t.Fatalf("compose.yaml と compose.override.yml の両方の検出を期待しましたが、%v でした", res)
    }
}

func TestFindOverrideFile(t *testing.T) {
    tempDir := t.TempDir()

    // 存在しない場合
    if ov := FindOverrideFile(tempDir, "compose.yml"); ov != "" {
        t.Errorf("存在しない override ファイルが検出されました: %s", ov)
    }

    // compose.override.yml を作成
    ovPath := filepath.Join(tempDir, "compose.override.yml")
    if err := os.WriteFile(ovPath, []byte(""), 0644); err != nil {
        t.Fatal(err)
    }

    if ov := FindOverrideFile(tempDir, "compose.yml"); ov != "compose.override.yml" {
        t.Errorf("compose.override.yml が検出されるべきですが、%s でした", ov)
    }
}

func TestApplyDefaultsAndResolvePaths(t *testing.T) {
    cfg := &Config{
        Version: "1",
        Network: NetworkConfig{},
        Core: CoreConfig{
            WorkDir: "~/workspace/toolbox",
        },
        Projects: map[string]ProjectItem{
            "sample": {
                WorkDir: "~/workspace/sample",
            },
        },
    }

    cfg.applyDefaultsAndResolvePaths()

    // デフォルト値の検証
    if cfg.Defaults.ComposeCmd != "docker compose" {
        t.Errorf("Defaults.ComposeCmd の期待値は 'docker compose' ですが、%s でした", cfg.Defaults.ComposeCmd)
    }
    if cfg.Network.Name != "internal-network" {
        t.Errorf("Network.Name の期待値は 'internal-network' ですが、%s でした", cfg.Network.Name)
    }
    if cfg.Network.Driver != "bridge" {
        t.Errorf("Network.Driver の期待値は 'bridge' ですが、%s でした", cfg.Network.Driver)
    }

    // チルダ展開の検証
    homeDir, _ := os.UserHomeDir()
    expectedCoreDir := filepath.Join(homeDir, "workspace/toolbox")
    if cfg.Core.WorkDir != expectedCoreDir {
        t.Errorf("Core.WorkDir の展開値期待は %s ですが、%s でした", expectedCoreDir, cfg.Core.WorkDir)
    }

    // 既存プロジェクト(sample)に compose.yml が存在するため自動検出されること
    proj := cfg.Projects["sample"]
    if len(proj.ComposeFiles) == 0 || proj.ComposeFiles[0] != "compose.yml" {
        t.Errorf("sample の ComposeFiles は compose.yml が自動検出されるはずですが、%v でした", proj.ComposeFiles)
    }
}

func TestFindProjectByWorkingDir(t *testing.T) {
    tempDir := t.TempDir()
    projDir := filepath.Join(tempDir, "my-app")
    subDir := filepath.Join(projDir, "src", "components")
    _ = os.MkdirAll(subDir, 0755)

    cfg := &Config{
        Projects: map[string]ProjectItem{
            "my-app": {
                WorkDir: projDir,
            },
        },
    }
    cfg.applyDefaultsAndResolvePaths()

    // プロジェクトルートでの検索
    if proj, ok := cfg.FindProjectByWorkingDir(projDir); !ok || proj.Name != "my-app" {
        t.Errorf("プロジェクトルートでの検索に失敗しました: ok=%v, proj=%v", ok, proj)
    }

    // サブディレクトリでの検索
    if proj, ok := cfg.FindProjectByWorkingDir(subDir); !ok || proj.Name != "my-app" {
        t.Errorf("サブディレクトリでの検索に失敗しました: ok=%v, proj=%v", ok, proj)
    }

    // 無関係のディレクトリでの検索
    if proj, ok := cfg.FindProjectByWorkingDir(tempDir); ok {
        t.Errorf("無関係のディレクトリでプロジェクトが検出されてしまいました: %v", proj)
    }
}

func TestValidateFile(t *testing.T) {
    tempDir := t.TempDir()

    // 1. 正常な devctl 設定ファイル
    validYAML := `version: "1"
defaults:
  compose_cmd: "docker compose"
network:
  name: "custom-net"
  subnet: "172.20.0.0/16"
  gateway: "172.20.0.1"
`
    validPath := filepath.Join(tempDir, "valid.yaml")
    if err := os.WriteFile(validPath, []byte(validYAML), 0644); err != nil {
        t.Fatal(err)
    }

    cfg, err := ValidateFile(validPath)
    if err != nil {
        t.Fatalf("正常な設定ファイルでエラーが発生しました: %v", err)
    }
    if cfg.Network.Name != "custom-net" {
        t.Errorf("Network.Name の期待値は 'custom-net' ですが、%s でした", cfg.Network.Name)
    }
    if cfg.Network.Subnet != "172.20.0.0/16" {
        t.Errorf("Network.Subnet の期待値は '172.20.0.0/16' ですが、%s でした", cfg.Network.Subnet)
    }
    if cfg.Network.Gateway != "172.20.0.1" {
        t.Errorf("Network.Gateway の期待値は '172.20.0.1' ですが、%s でした", cfg.Network.Gateway)
    }

    // 2. YAML 構文不正ファイル
    invalidYAML := `version: "1"
defaults:
  - invalid list instead of map
    broken syntax:
`
    invalidPath := filepath.Join(tempDir, "invalid.yaml")
    if err := os.WriteFile(invalidPath, []byte(invalidYAML), 0644); err != nil {
        t.Fatal(err)
    }

    if _, err := ValidateFile(invalidPath); err == nil {
        t.Errorf("構文不正ファイルでエラーが発生しませんでした")
    }

    // 3. version フィールドがないファイル
    noVersionYAML := `network:
  name: "foo"
`
    noVersionPath := filepath.Join(tempDir, "no_version.yaml")
    if err := os.WriteFile(noVersionPath, []byte(noVersionYAML), 0644); err != nil {
        t.Fatal(err)
    }

    if _, err := ValidateFile(noVersionPath); err == nil {
        t.Errorf("version フィールドが存在しないファイルでエラーが発生しませんでした")
    }

    // 4. 存在しないファイル
    if _, err := ValidateFile(filepath.Join(tempDir, "not_exist.yaml")); err == nil {
        t.Errorf("存在しないファイルでエラーが発生しませんでした")
    }
}

func TestTaskItem_UnmarshalYAML(t *testing.T) {
    tempDir := t.TempDir()
    tasksYAML := `version: "1"
projects:
  sample:
    workdir: "~/workspace/sample"
    tasks:
      single-map:
        description: "単一コマンド定義"
        command: "docker compose exec app php artisan migrate"
      multi-seq:
        - "docker compose exec app composer install"
        - "docker compose exec app php artisan migrate"
        - "docker compose exec app php artisan db:seed"
      multi-map-commands:
        description: "複数コマンド定義 (commands)"
        commands:
          - "cmd-1 --flag"
          - "ENV_VAR=1 cmd-2"
      multi-map-command-list:
        description: "複数コマンド定義 (command リスト)"
        command:
          - "step-a"
          - "step-b"
      single-scalar: "echo hello"
`
    cfgPath := filepath.Join(tempDir, "tasks.yaml")
    if err := os.WriteFile(cfgPath, []byte(tasksYAML), 0644); err != nil {
        t.Fatal(err)
    }

    cfg, err := ValidateFile(cfgPath)
    if err != nil {
        t.Fatalf("tasks.yaml のパースに失敗しました: %v", err)
    }

    tasks := cfg.Projects["sample"].Tasks

    t.Run("既存の単一コマンド形式 (description + command) がパースできること", func(t *testing.T) {
        item := tasks["single-map"]
        if item.Description != "単一コマンド定義" {
            t.Errorf("Description 不一致: %s", item.Description)
        }
        expected := []string{"docker compose exec app php artisan migrate"}
        if !reflect.DeepEqual(item.GetCommands(), expected) {
            t.Errorf("GetCommands 不一致: 期待値 %v, 実際 %v", expected, item.GetCommands())
        }
    })

    t.Run("Issue #11 の直リスト形式 (tasks.<name> 直下の配列) が定義順にパースできること", func(t *testing.T) {
        item := tasks["multi-seq"]
        expected := []string{
            "docker compose exec app composer install",
            "docker compose exec app php artisan migrate",
            "docker compose exec app php artisan db:seed",
        }
        if !reflect.DeepEqual(item.GetCommands(), expected) {
            t.Errorf("GetCommands 不一致: 期待値 %v, 実際 %v", expected, item.GetCommands())
        }
        if item.Summary() == "" {
            t.Errorf("Description 未設定時の Summary() が空です")
        }
    })

    t.Run("description 付きの commands 配列形式が定義順にパースできること", func(t *testing.T) {
        item := tasks["multi-map-commands"]
        if item.Description != "複数コマンド定義 (commands)" {
            t.Errorf("Description 不一致: %s", item.Description)
        }
        expected := []string{"cmd-1 --flag", "ENV_VAR=1 cmd-2"}
        if !reflect.DeepEqual(item.GetCommands(), expected) {
            t.Errorf("GetCommands 不一致: 期待値 %v, 実際 %v", expected, item.GetCommands())
        }
    })

    t.Run("command キーに配列を指定した形式もパースできること", func(t *testing.T) {
        item := tasks["multi-map-command-list"]
        expected := []string{"step-a", "step-b"}
        if !reflect.DeepEqual(item.GetCommands(), expected) {
            t.Errorf("GetCommands 不一致: 期待値 %v, 実際 %v", expected, item.GetCommands())
        }
    })

    t.Run("スカラー文字列直書き形式もパースできること", func(t *testing.T) {
        item := tasks["single-scalar"]
        expected := []string{"echo hello"}
        if !reflect.DeepEqual(item.GetCommands(), expected) {
            t.Errorf("GetCommands 不一致: 期待値 %v, 実際 %v", expected, item.GetCommands())
        }
    })
}

