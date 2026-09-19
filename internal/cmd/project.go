package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/skyou/devctl/internal/config"
	"github.com/skyou/devctl/internal/docker"
	"github.com/skyou/devctl/internal/ui"
)

var (
	projectBuild         bool
	projectRemoveVolumes bool
	projectFollowLogs    bool
	projectTailLogs      string
)

var projectCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"proj", "p"},
	Short:   "Manage individual application projects",
	Long:    `Start, stop, inspect, and run tasks for registered projects from anywhere.`,
}

// resolveTargetProject determines which project to target.
// It checks explicit command line argument first, then falls back to current working directory.
func resolveTargetProject(args []string) (*config.ProjectItem, []string, error) {
	if len(args) > 0 {
		projectName := args[0]
		if proj, ok := cfg.Projects[projectName]; ok {
			return &proj, args[1:], nil
		}
		// If first arg isn't a project, maybe current dir is a project
		if currentDir, err := os.Getwd(); err == nil {
			if proj, found := cfg.FindProjectByWorkingDir(currentDir); found {
				return proj, args, nil
			}
		}
		return nil, nil, fmt.Errorf("project '%s' not found in configuration", projectName)
	}

	// No argument provided: check current directory
	currentDir, err := os.Getwd()
	if err == nil {
		if proj, found := cfg.FindProjectByWorkingDir(currentDir); found {
			ui.Info("Using detected project '%s' based on current directory (%s)", proj.Name, currentDir)
			return proj, nil, nil
		}
	}

	return nil, nil, fmt.Errorf("missing project name argument, and current directory is not a registered project")
}

// prepareDependencies ensures shared network and core services are up if project depends on them.
func prepareDependencies(proj *config.ProjectItem) error {
	if proj.DependsOn.Network {
		if err := networkManager.Ensure(cfg.Network.Name, cfg.Network.Driver, cfg.Network.Attachable); err != nil {
			return fmt.Errorf("failed to ensure network for %s: %w", proj.Name, err)
		}
	}

	if len(proj.DependsOn.Core) > 0 {
		ui.Info("Ensuring core dependencies for %s: %v", proj.Name, proj.DependsOn.Core)
		opts := docker.ComposeOptions{
			WorkDir:      cfg.Core.WorkDir,
			ComposeFiles: cfg.Core.ComposeFiles,
			EnvFile:      cfg.Core.EnvFile,
			Services:     proj.DependsOn.Core,
		}
		if err := composeClient.Up(opts, false); err != nil {
			return fmt.Errorf("failed to start required core services: %w", err)
		}
	}
	return nil
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all registered projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(cfg.Projects) == 0 {
			ui.Warn("No projects defined in config.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PROJECT\tWORKDIR\tTASKS\tDESCRIPTION")

		var names []string
		for name := range cfg.Projects {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			p := cfg.Projects[name]
			taskCount := len(p.Tasks)
			fmt.Fprintf(w, "%s\t%s\t%d task(s)\t%s\n", p.Name, p.WorkDir, taskCount, p.Description)
		}
		w.Flush()
		return nil
	},
}

var projectUpCmd = &cobra.Command{
	Use:   "up [project-name] [services...]",
	Short: "Start project containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		proj, remainingArgs, err := resolveTargetProject(args)
		if err != nil {
			return err
		}

		if err := prepareDependencies(proj); err != nil {
			return err
		}

		opts := docker.ComposeOptions{
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
			Services:     remainingArgs,
		}

		if err := composeClient.Up(opts, projectBuild); err != nil {
			return fmt.Errorf("failed to start project %s: %w", proj.Name, err)
		}

		ui.Success("Project '%s' started successfully.", proj.Name)
		return nil
	},
}

var projectDownCmd = &cobra.Command{
	Use:   "down [project-name] [services...]",
	Short: "Stop project containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		proj, remainingArgs, err := resolveTargetProject(args)
		if err != nil {
			return err
		}

		opts := docker.ComposeOptions{
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
			Services:     remainingArgs,
		}

		if err := composeClient.Down(opts, projectRemoveVolumes); err != nil {
			return fmt.Errorf("failed to stop project %s: %w", proj.Name, err)
		}

		ui.Success("Project '%s' stopped.", proj.Name)
		return nil
	},
}

var projectRestartCmd = &cobra.Command{
	Use:   "restart [project-name] [services...]",
	Short: "Restart project containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		proj, remainingArgs, err := resolveTargetProject(args)
		if err != nil {
			return err
		}

		opts := docker.ComposeOptions{
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
			Services:     remainingArgs,
		}

		return composeClient.Restart(opts)
	},
}

var projectPsCmd = &cobra.Command{
	Use:   "ps [project-name]",
	Short: "Show containers for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		proj, remainingArgs, err := resolveTargetProject(args)
		if err != nil {
			return err
		}

		opts := docker.ComposeOptions{
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
			Services:     remainingArgs,
		}

		return composeClient.Ps(opts)
	},
}

var projectLogsCmd = &cobra.Command{
	Use:   "logs [project-name] [services...]",
	Short: "View logs for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		proj, remainingArgs, err := resolveTargetProject(args)
		if err != nil {
			return err
		}

		opts := docker.ComposeOptions{
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
			Services:     remainingArgs,
		}

		return composeClient.Logs(opts, projectFollowLogs, projectTailLogs)
	},
}

var projectExecCmd = &cobra.Command{
	Use:   "exec [project-name] <service> <command...>",
	Short: "Execute a command inside a project container",
	RunE: func(cmd *cobra.Command, args []string) error {
		proj, remainingArgs, err := resolveTargetProject(args)
		if err != nil {
			return err
		}

		if len(remainingArgs) < 2 {
			return fmt.Errorf("usage: devctl project exec [project-name] <service> <command...>")
		}

		service := remainingArgs[0]
		execCmd := remainingArgs[1:]

		opts := docker.ComposeOptions{
			WorkDir:      proj.WorkDir,
			ComposeFiles: proj.ComposeFiles,
			EnvFile:      proj.EnvFile,
		}

		return composeClient.Exec(opts, service, execCmd)
	},
}

var projectRunCmd = &cobra.Command{
	Use:   "run [project-name] <task-name>",
	Short: "Run a predefined project task (Makefile replacement)",
	Long: `Executes a task command defined under the project's 'tasks' section in config.yaml.
Example:
  devctl run web-app migrate
  devctl project run web-app seed`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("missing task name or project name")
		}

		var proj *config.ProjectItem
		var taskName string

		// Try matching: devctl run <proj> <task>
		if p, ok := cfg.Projects[args[0]]; ok {
			if len(args) < 2 {
				// Show available tasks for this project
				ui.Info("Available tasks for project '%s':", p.Name)
				for tName, tItem := range p.Tasks {
					fmt.Printf("  %-15s %s\n", tName, tItem.Description)
				}
				return fmt.Errorf("missing task name to run")
			}
			proj = &p
			taskName = args[1]
		} else {
			// Try matching: devctl run <task> (from current project directory)
			currentDir, err := os.Getwd()
			if err == nil {
				if detected, found := cfg.FindProjectByWorkingDir(currentDir); found {
					proj = detected
					taskName = args[0]
				}
			}
		}

		if proj == nil {
			return fmt.Errorf("could not determine project for task. Specify project name: devctl run <project> <task>")
		}

		task, ok := proj.Tasks[taskName]
		if !ok {
			ui.Warn("Task '%s' not found for project '%s'. Available tasks:", taskName, proj.Name)
			for tName, tItem := range proj.Tasks {
				fmt.Printf("  %-15s %s\n", tName, tItem.Description)
			}
			return fmt.Errorf("task not defined")
		}

		ui.Info("Running task '%s' in project '%s'...", taskName, proj.Name)
		return composeClient.RunShellTask(proj.WorkDir, task.Command)
	},
}

// Top-level aliases for convenient everyday developer workflow
var (
	topUpCmd = &cobra.Command{
		Use:   "up [project-name] [services...]",
		Short: "Alias for 'project up'",
		RunE:  projectUpCmd.RunE,
	}
	topDownCmd = &cobra.Command{
		Use:   "down [project-name] [services...]",
		Short: "Alias for 'project down'",
		RunE:  projectDownCmd.RunE,
	}
	topRestartCmd = &cobra.Command{
		Use:   "restart [project-name] [services...]",
		Short: "Alias for 'project restart'",
		RunE:  projectRestartCmd.RunE,
	}
	topLogsCmd = &cobra.Command{
		Use:   "logs [project-name] [services...]",
		Short: "Alias for 'project logs'",
		RunE:  projectLogsCmd.RunE,
	}
	topRunCmd = &cobra.Command{
		Use:   "run [project-name] <task-name>",
		Short: "Alias for 'project run'",
		RunE:  projectRunCmd.RunE,
	}
	topListCmd = &cobra.Command{
		Use:   "list",
		Short: "Alias for 'project list'",
		RunE:  projectListCmd.RunE,
	}
)

func init() {
	RootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectUpCmd)
	projectCmd.AddCommand(projectDownCmd)
	projectCmd.AddCommand(projectRestartCmd)
	projectCmd.AddCommand(projectPsCmd)
	projectCmd.AddCommand(projectLogsCmd)
	projectCmd.AddCommand(projectExecCmd)
	projectCmd.AddCommand(projectRunCmd)

	// Add top-level ergonomic aliases
	RootCmd.AddCommand(topUpCmd)
	RootCmd.AddCommand(topDownCmd)
	RootCmd.AddCommand(topRestartCmd)
	RootCmd.AddCommand(topLogsCmd)
	RootCmd.AddCommand(topRunCmd)
	RootCmd.AddCommand(topListCmd)

	// Flags
	flags := []*cobra.Command{projectUpCmd, topUpCmd}
	for _, f := range flags {
		f.Flags().BoolVarP(&projectBuild, "build", "b", false, "build images before starting")
	}

	downFlags := []*cobra.Command{projectDownCmd, topDownCmd}
	for _, f := range downFlags {
		f.Flags().BoolVarP(&projectRemoveVolumes, "volumes", "v", false, "remove named volumes")
	}

	logFlags := []*cobra.Command{projectLogsCmd, topLogsCmd}
	for _, f := range logFlags {
		f.Flags().BoolVarP(&projectFollowLogs, "follow", "f", false, "follow log output")
		f.Flags().StringVarP(&projectTailLogs, "tail", "t", "100", "number of lines to show from end")
	}
}
