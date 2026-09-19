package docker

import (
	"strings"

	"github.com/skyou/devctl/internal/ui"
)

// ComposeClient handles Docker Compose commands for core infra and projects.
type ComposeClient struct {
	runner     *Runner
	composeCmd string
}

// NewComposeClient creates a new Docker Compose client.
func NewComposeClient(runner *Runner, composeCmd string) *ComposeClient {
	if composeCmd == "" {
		composeCmd = "docker compose"
	}
	return &ComposeClient{
		runner:     runner,
		composeCmd: composeCmd,
	}
}

// ComposeOptions configures compose invocations.
type ComposeOptions struct {
	WorkDir      string
	ComposeFiles []string
	EnvFile      string
	Services     []string
	ExtraArgs    []string
}

// buildBaseArgs builds the base command and arguments for docker compose.
func (c *ComposeClient) buildBaseArgs(opts ComposeOptions) (string, []string) {
	cmdParts := strings.Fields(c.composeCmd)
	bin := cmdParts[0]
	args := cmdParts[1:]

	for _, f := range opts.ComposeFiles {
		args = append(args, "-f", f)
	}

	if opts.EnvFile != "" {
		args = append(args, "--env-file", opts.EnvFile)
	}

	return bin, args
}

// Up starts services in detached mode.
func (c *ComposeClient) Up(opts ComposeOptions, build bool) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "up", "-d")
	if build {
		args = append(args, "--build")
	}
	args = append(args, opts.Services...)
	args = append(args, opts.ExtraArgs...)

	ui.Step("Starting containers in %s...", opts.WorkDir)
	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Down stops and removes containers.
func (c *ComposeClient) Down(opts ComposeOptions, removeVolumes bool) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "down")
	if removeVolumes {
		args = append(args, "-v")
	}
	args = append(args, opts.ExtraArgs...)

	ui.Step("Stopping containers in %s...", opts.WorkDir)
	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Restart restarts services.
func (c *ComposeClient) Restart(opts ComposeOptions) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "restart")
	args = append(args, opts.Services...)
	args = append(args, opts.ExtraArgs...)

	ui.Step("Restarting containers in %s...", opts.WorkDir)
	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Logs displays or follows service logs.
func (c *ComposeClient) Logs(opts ComposeOptions, follow bool, tail string) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "logs")
	if follow {
		args = append(args, "-f")
	}
	if tail != "" {
		args = append(args, "--tail", tail)
	}
	args = append(args, opts.Services...)
	args = append(args, opts.ExtraArgs...)

	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Ps lists running containers in the specified compose environment.
func (c *ComposeClient) Ps(opts ComposeOptions) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "ps")
	args = append(args, opts.ExtraArgs...)

	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Exec executes a command inside a running container.
func (c *ComposeClient) Exec(opts ComposeOptions, service string, cmd []string) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "exec", service)
	args = append(args, cmd...)

	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// Run executes a one-off command on a service.
func (c *ComposeClient) Run(opts ComposeOptions, service string, cmd []string) error {
	bin, baseArgs := c.buildBaseArgs(opts)
	args := append(baseArgs, "run", "--rm", service)
	args = append(args, cmd...)

	return c.runner.RunCommand(opts.WorkDir, bin, args...)
}

// RunShellTask executes a custom task defined in config (replaces Makefile target).
func (c *ComposeClient) RunShellTask(workDir, shellCommand string) error {
	ui.Step("Running custom task: %s", shellCommand)
	return c.runner.RunShell(workDir, shellCommand)
}
