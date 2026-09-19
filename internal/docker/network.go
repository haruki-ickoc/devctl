package docker

import (
	"fmt"
	"strings"

	"github.com/skyou/devctl/internal/ui"
)

// NetworkManager は Docker ネットワークの作成・存在確認・削除を管理します。
type NetworkManager struct {
	runner *Runner
}

// NewNetworkManager は新しい NetworkManager インスタンスを生成します。
func NewNetworkManager(runner *Runner) *NetworkManager {
	return &NetworkManager{runner: runner}
}

// Exists は指定された名前の Docker ネットワークが存在するか確認します。
func (n *NetworkManager) Exists(name string) (bool, error) {
	if n.runner.DryRun {
		ui.Dim("[dry-run] ネットワーク '%s' の存在確認シミュレーション（作成プレビューのため未作成と仮定）", name)
		return false, nil
	}
	out, err := n.runner.RunCommandOutput("", "docker", "network", "ls", "--filter", fmt.Sprintf("name=^%s$", name), "--format", "{{.Name}}")
	if err != nil {
		return false, fmt.Errorf("Docker ネットワークの確認に失敗しました (Docker デーモンは起動していますか？): %w", err)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == name {
			return true, nil
		}
	}
	return false, nil
}

// Ensure はネットワークが存在しない場合に自動作成します。
func (n *NetworkManager) Ensure(name, driver string, attachable bool) error {
	exists, err := n.Exists(name)
	if err != nil {
		return err
	}
	if exists {
		ui.Success("ネットワーク '%s' は既に存在します。", name)
		return nil
	}

	ui.Info("ネットワーク '%s' が見つかりません。作成します...", name)
	return n.Create(name, driver, attachable)
}

// Create は指定された名前・ドライバ・設定で Docker ネットワークを作成します。
func (n *NetworkManager) Create(name, driver string, attachable bool) error {
	args := []string{"network", "create"}
	if driver != "" {
		args = append(args, "--driver", driver)
	}
	if attachable {
		args = append(args, "--attachable")
	}
	args = append(args, name)

	if err := n.runner.RunCommand("", "docker", args...); err != nil {
		return fmt.Errorf("ネットワーク '%s' の作成に失敗しました: %w", name, err)
	}
	ui.Success("ネットワーク '%s' を作成しました。", name)
	return nil
}

// Remove は指定された Docker ネットワークを削除します。
func (n *NetworkManager) Remove(name string) error {
	exists, err := n.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		ui.Warn("ネットワーク '%s' は存在しません。", name)
		return nil
	}

	if err := n.runner.RunCommand("", "docker", "network", "rm", name); err != nil {
		return fmt.Errorf("ネットワーク '%s' の削除に失敗しました: %w", name, err)
	}
	ui.Success("ネットワーク '%s' を削除しました。", name)
	return nil
}
