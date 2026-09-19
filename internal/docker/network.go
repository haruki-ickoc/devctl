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

// InspectIPAM は指定された Docker ネットワークの Subnet および Gateway 情報を取得します。
func (n *NetworkManager) InspectIPAM(name string) (subnet string, gateway string, err error) {
    if n.runner.DryRun {
        ui.Dim("[dry-run] ネットワーク '%s' の IPAM 設定確認シミュレーション", name)
        return "", "", nil
    }
    out, err := n.runner.RunCommandOutput("", "docker", "network", "inspect", name, "--format", `{{range .IPAM.Config}}{{.Subnet}},{{.Gateway}}{{"\n"}}{{end}}`)
    if err != nil {
        return "", "", fmt.Errorf("ネットワーク '%s' の情報取得に失敗しました: %w", name, err)
    }
    out = strings.TrimSpace(out)
    if out == "" {
        return "", "", nil
    }
    lines := strings.Split(out, "\n")
    if len(lines) > 0 {
        parts := strings.Split(strings.TrimSpace(lines[0]), ",")
        if len(parts) >= 1 {
            subnet = parts[0]
        }
        if len(parts) >= 2 {
            gateway = parts[1]
        }
    }
    return subnet, gateway, nil
}

// Ensure はネットワークが存在しない場合に自動作成します。
// 既に存在し、subnet または gateway が指定されている場合は設定値との整合性を確認して警告を出力します。
func (n *NetworkManager) Ensure(name, driver, subnet, gateway string, attachable bool) error {
    exists, err := n.Exists(name)
    if err != nil {
        return err
    }
    if exists {
        ui.Success("ネットワーク '%s' は既に存在します。", name)
        if !n.runner.DryRun && (subnet != "" || gateway != "") {
            actualSubnet, actualGateway, err := n.InspectIPAM(name)
            if err == nil {
                n.warnIfMismatch(name, subnet, gateway, actualSubnet, actualGateway)
            }
        }
        return nil
    }

    ui.Info("ネットワーク '%s' が見つかりません。作成します...", name)
    return n.Create(name, driver, subnet, gateway, attachable)
}

func (n *NetworkManager) warnIfMismatch(name, expectedSubnet, expectedGateway, actualSubnet, actualGateway string) {
    var mismatches []string
    if expectedSubnet != "" && actualSubnet != "" && expectedSubnet != actualSubnet {
        mismatches = append(mismatches, fmt.Sprintf("Subnet (設定: %s, 実環境: %s)", expectedSubnet, actualSubnet))
    }
    if expectedGateway != "" && actualGateway != "" && expectedGateway != actualGateway {
        mismatches = append(mismatches, fmt.Sprintf("Gateway (設定: %s, 実環境: %s)", expectedGateway, actualGateway))
    }

    if len(mismatches) > 0 {
        ui.Warn("▲ 警告: 既存ネットワーク '%s' の構成が config.yaml の設定と一致していません:", name)
        for _, m := range mismatches {
            ui.Warn("    - %s", m)
        }
        ui.Warn("  固定IPの競合や接続エラーを防ぐため、再作成する場合は 'devctl network rm' を実行してから再度起動してください。")
    }
}

// Create は指定された名前・ドライバ・設定で Docker ネットワークを作成します。
func (n *NetworkManager) Create(name, driver, subnet, gateway string, attachable bool) error {
    args := []string{"network", "create"}
    if driver != "" {
        args = append(args, "--driver", driver)
    }
    if attachable {
        args = append(args, "--attachable")
    }
    if subnet != "" {
        args = append(args, "--subnet", subnet)
    }
    if gateway != "" {
        args = append(args, "--gateway", gateway)
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
