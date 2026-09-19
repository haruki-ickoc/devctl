package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/skyou/devctl/internal/ui"
)

// Runner handles command execution.
type Runner struct {
	DryRun  bool
	Verbose bool
}

// NewRunner creates a new command runner.
func NewRunner(dryRun, verbose bool) *Runner {
	return &Runner{
		DryRun:  dryRun,
		Verbose: verbose,
	}
}

// RunCommand executes a command in a given working directory with streaming stdout and stderr.
func (r *Runner) RunCommand(dir string, name string, args ...string) error {
	fullCmd := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	if dir != "" {
		ui.Dim("[exec] (in %s): %s", dir, fullCmd)
	} else {
		ui.Dim("[exec]: %s", fullCmd)
	}

	if r.DryRun {
		ui.Info("[dry-run] Skipped execution of: %s", fullCmd)
		return nil
	}

	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// RunShell runs an arbitrary shell command string inside a specified working directory.
func (r *Runner) RunShell(dir, shellCmd string) error {
	if dir != "" {
		ui.Dim("[shell] (in %s): %s", dir, shellCmd)
	} else {
		ui.Dim("[shell]: %s", shellCmd)
	}

	if r.DryRun {
		ui.Info("[dry-run] Skipped execution of: %s", shellCmd)
		return nil
	}

	// Use sh -c to execute compound shell scripts / pipes
	cmd := exec.Command("sh", "-c", shellCmd)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// RunCommandOutput runs a command and captures its trimmed stdout output.
func (r *Runner) RunCommandOutput(dir string, name string, args ...string) (string, error) {
	if r.DryRun {
		ui.Dim("[dry-run inspect]: %s %s", name, strings.Join(args, " "))
		return "", nil
	}
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
