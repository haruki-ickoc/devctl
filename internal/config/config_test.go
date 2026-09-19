package config

import (
    "os"
    "path/filepath"
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

