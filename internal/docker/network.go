package docker

import (
	"fmt"
	"strings"

	"github.com/skyou/devctl/internal/ui"
)

// NetworkManager manages Docker networks.
type NetworkManager struct {
	runner *Runner
}

// NewNetworkManager creates a new NetworkManager.
func NewNetworkManager(runner *Runner) *NetworkManager {
	return &NetworkManager{runner: runner}
}

// Exists checks if a docker network exists.
func (n *NetworkManager) Exists(name string) (bool, error) {
	if n.runner.DryRun {
		ui.Dim("[dry-run] Simulating check for network '%s' (assumed missing for setup preview)", name)
		return false, nil
	}
	out, err := n.runner.RunCommandOutput("", "docker", "network", "ls", "--filter", fmt.Sprintf("name=^%s$", name), "--format", "{{.Name}}")
	if err != nil {
		return false, fmt.Errorf("failed to check docker network (is Docker daemon running?): %w", err)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == name {
			return true, nil
		}
	}
	return false, nil
}

// Ensure creates the network if it does not already exist.
func (n *NetworkManager) Ensure(name, driver string, attachable bool) error {
	exists, err := n.Exists(name)
	if err != nil {
		return err
	}
	if exists {
		ui.Success("Network '%s' already exists.", name)
		return nil
	}

	ui.Info("Network '%s' not found. Creating...", name)
	return n.Create(name, driver, attachable)
}

// Create creates a docker network.
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
		return fmt.Errorf("failed to create network '%s': %w", name, err)
	}
	ui.Success("Network '%s' created successfully.", name)
	return nil
}

// Remove deletes a docker network.
func (n *NetworkManager) Remove(name string) error {
	exists, err := n.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		ui.Warn("Network '%s' does not exist.", name)
		return nil
	}

	if err := n.runner.RunCommand("", "docker", "network", "rm", name); err != nil {
		return fmt.Errorf("failed to remove network '%s': %w", name, err)
	}
	ui.Success("Network '%s' removed.", name)
	return nil
}
