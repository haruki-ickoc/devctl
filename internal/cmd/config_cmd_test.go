package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigInit_Default(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// 引数なし、フラグリセット
	configInitFrom = ""
	configInitForce = false

	err := configInitCmd.RunE(configInitCmd, []string{})
	if err != nil {
		t.Fatalf("config init の実行に失敗しました: %v", err)
	}

	destPath := filepath.Join(tempHome, ".config", "devctl", "config.yaml")
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("生成された設定ファイルの読み込みに失敗しました: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "dev-network") {
		t.Errorf("デフォルトテンプレートに 'dev-network' が含まれていません: %s", content)
	}
}

func TestConfigInit_SkipExistingWithoutForce(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	destPath := filepath.Join(tempHome, ".config", "devctl", "config.yaml")
	_ = os.MkdirAll(filepath.Dir(destPath), 0755)
	_ = os.WriteFile(destPath, []byte("original-content"), 0644)

	configInitFrom = ""
	configInitForce = false

	err := configInitCmd.RunE(configInitCmd, []string{})
	if err != nil {
		t.Fatalf("既存ファイル存在時にエラーが発生しました: %v", err)
	}

	// 上書きされていないことを確認
	data, _ := os.ReadFile(destPath)
	if string(data) != "original-content" {
		t.Errorf("--force なしで既存ファイルが上書きされてしまいました: %s", string(data))
	}
}

func TestConfigInit_ForceOverwrite(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	destPath := filepath.Join(tempHome, ".config", "devctl", "config.yaml")
	_ = os.MkdirAll(filepath.Dir(destPath), 0755)
	_ = os.WriteFile(destPath, []byte("original-content"), 0644)

	configInitFrom = ""
	configInitForce = true

	err := configInitCmd.RunE(configInitCmd, []string{})
	if err != nil {
		t.Fatalf("--force 実行時にエラーが発生しました: %v", err)
	}

	// 上書きされていることを確認
	data, _ := os.ReadFile(destPath)
	if string(data) == "original-content" {
		t.Errorf("--force 指定時に既存ファイルが上書きされませんでした")
	}
}

func TestConfigInit_ImportExistingValidFile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// インポート元ファイルの作成
	sourceYAML := `version: "1"
defaults:
  compose_cmd: "docker compose"
network:
  name: "imported-net"
`
	sourcePath := filepath.Join(t.TempDir(), "team-config.yaml")
	_ = os.WriteFile(sourcePath, []byte(sourceYAML), 0644)

	configInitFrom = ""
	configInitForce = false

	// 引数でソースファイルを指定
	err := configInitCmd.RunE(configInitCmd, []string{sourcePath})
	if err != nil {
		t.Fatalf("インポート実行に失敗しました: %v", err)
	}

	destPath := filepath.Join(tempHome, ".config", "devctl", "config.yaml")
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("インポートされた設定ファイルが読めません: %v", err)
	}

	if !strings.Contains(string(data), "imported-net") {
		t.Errorf("インポートされた設定内容が一致しません: %s", string(data))
	}
}

func TestConfigInit_ImportInvalidFile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// 構文不正ファイルの作成
	brokenYAML := `version: "1"
defaults:
  - invalid list
    syntax error:
`
	sourcePath := filepath.Join(t.TempDir(), "broken-config.yaml")
	_ = os.WriteFile(sourcePath, []byte(brokenYAML), 0644)

	configInitFrom = ""
	configInitForce = false

	err := configInitCmd.RunE(configInitCmd, []string{sourcePath})
	if err == nil {
		t.Fatalf("構文不正ファイルのインポートでエラーが発生しませんでした")
	}

	// 配置先が作成されていないことを確認
	destPath := filepath.Join(tempHome, ".config", "devctl", "config.yaml")
	if _, err := os.Stat(destPath); err == nil {
		t.Errorf("構文不正ファイルが配置先に作成されてしまいました")
	}
}
