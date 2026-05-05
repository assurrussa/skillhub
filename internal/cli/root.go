package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/assurrussa/skillhub/internal/tui"
	"github.com/spf13/cobra"
)

var repoFlag string

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "skillhub",
		Short:         "Discover and install agent skills from registered sources",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			return tui.Run(repoRoot)
		},
	}

	root.PersistentFlags().StringVar(&repoFlag, "repo", "", "skillhub repository path")

	sources := &cobra.Command{
		Use:   "sources",
		Short: "Manage skill sources",
	}
	sources.AddCommand(sourceListCommand())
	sources.AddCommand(scriptCommand("sync [source-name]", "Sync all sources or one source", "scripts/sources.sh", []string{"sync"}, cobra.MaximumNArgs(1)))
	sources.AddCommand(sourceAddCommand())
	sources.AddCommand(sourceRemoveCommand())
	sources.AddCommand(sourceDefaultsCommand())

	targets := &cobra.Command{
		Use:   "targets",
		Short: "List supported and planned install targets",
	}
	targets.AddCommand(targetListCommand())
	targets.AddCommand(targetDetectCommand())

	installed := &cobra.Command{
		Use:   "installed",
		Short: "Inspect installed skills",
	}
	installed.AddCommand(installedListCommand())
	installed.AddCommand(installedUpdateCommand())
	installed.AddCommand(installedUninstallCommand())

	skills := &cobra.Command{
		Use:   "skills",
		Short: "List, search, and install skills",
	}
	skills.AddCommand(scriptCommand("list", "List available skills", "scripts/skills.sh", []string{"list"}, cobra.NoArgs))
	skills.AddCommand(scriptCommand("search <query>", "Search available skills", "scripts/skills.sh", []string{"search"}, cobra.MinimumNArgs(1)))
	skills.AddCommand(installCommand("install [<source>/]<skill-name>...", "Install selected skills"))

	root.AddCommand(sources, targets, installed, skills)
	root.AddCommand(scriptCommand("list", "List available skills", "scripts/skills.sh", []string{"list"}, cobra.NoArgs))
	root.AddCommand(scriptCommand("search <query>", "Search available skills", "scripts/skills.sh", []string{"search"}, cobra.MinimumNArgs(1)))
	root.AddCommand(installCommand("install [<source>/]<skill-name>...", "Install selected skills"))
	root.AddCommand(versionCommand())
	root.AddCommand(updateCommand())
	root.AddCommand(&cobra.Command{
		Use:   "tui",
		Short: "Open the interactive skill dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			return tui.Run(repoRoot)
		},
	})

	return root
}

func sourceListCommand() *cobra.Command {
	var tsv bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured skill sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			scriptArgs := []string{"list"}
			if tsv {
				scriptArgs = append(scriptArgs, "--tsv")
			}
			return runScript(repoRoot, "scripts/sources.sh", scriptArgs...)
		},
	}
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	return cmd
}

func sourceAddCommand() *cobra.Command {
	var name string
	var sourceType string
	var ref string
	var catalog string
	cmd := &cobra.Command{
		Use:   "add <path-or-git-url>",
		Short: "Add a user skill source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			scriptArgs := []string{"add", args[0]}
			if name != "" {
				scriptArgs = append(scriptArgs, "--name", name)
			}
			if sourceType != "" {
				scriptArgs = append(scriptArgs, "--type", sourceType)
			}
			if ref != "" {
				scriptArgs = append(scriptArgs, "--ref", ref)
			}
			if catalog != "" {
				scriptArgs = append(scriptArgs, "--catalog", catalog)
			}
			return runScript(repoRoot, "scripts/sources.sh", scriptArgs...)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "source name; defaults to basename of path or git URL")
	cmd.Flags().StringVar(&sourceType, "type", "", "source type: path or git")
	cmd.Flags().StringVar(&ref, "ref", "", "git branch/tag/ref; defaults to main for git and - for path")
	cmd.Flags().StringVar(&catalog, "catalog", "", "catalog path inside the source; defaults to catalog/skills.tsv")
	return cmd
}

func sourceRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a user skill source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			return runScript(repoRoot, "scripts/sources.sh", "remove", args[0])
		},
	}
}

func sourceDefaultsCommand() *cobra.Command {
	defaults := &cobra.Command{
		Use:   "defaults",
		Short: "List and add recommended source presets",
	}
	defaults.AddCommand(scriptCommand("list", "List recommended source presets", "scripts/sources.sh", []string{"defaults", "list"}, cobra.NoArgs))
	defaults.AddCommand(scriptCommand("add <source-name>", "Add a recommended source preset", "scripts/sources.sh", []string{"defaults", "add"}, cobra.ExactArgs(1)))
	return defaults
}

func targetListCommand() *cobra.Command {
	var tsv bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List install targets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			scriptArgs := []string{"list"}
			if tsv {
				scriptArgs = append(scriptArgs, "--tsv")
			}
			return runScript(repoRoot, "scripts/targets.sh", scriptArgs...)
		},
	}
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	return cmd
}

func targetDetectCommand() *cobra.Command {
	var tsv bool
	var project string
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Show resolved paths for supported install targets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			scriptArgs := []string{"detect"}
			if tsv {
				scriptArgs = append(scriptArgs, "--tsv")
			}
			if cmd.Flags().Changed("project") {
				scriptArgs = append(scriptArgs, "--project", project)
			}
			return runScript(repoRoot, "scripts/targets.sh", scriptArgs...)
		},
	}
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	cmd.Flags().StringVar(&project, "project", "", "project directory for project-scope detection")
	return cmd
}

func installedListCommand() *cobra.Command {
	var target string
	var scope string
	var project string
	var dir string
	var tsv bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed skills for a target",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			scriptArgs := []string{"list"}
			if cmd.Flags().Changed("target") {
				scriptArgs = append(scriptArgs, "--target", target)
			}
			if cmd.Flags().Changed("scope") {
				scriptArgs = append(scriptArgs, "--scope", scope)
			}
			if cmd.Flags().Changed("project") {
				scriptArgs = append(scriptArgs, "--project", project)
			}
			if cmd.Flags().Changed("dir") {
				scriptArgs = append(scriptArgs, "--dir", dir)
			}
			if tsv {
				scriptArgs = append(scriptArgs, "--tsv")
			}
			return runScript(repoRoot, "scripts/installed.sh", scriptArgs...)
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "install target id; defaults to codex")
	cmd.Flags().StringVar(&scope, "scope", "", "install scope for scoped targets: global or project")
	cmd.Flags().StringVar(&project, "project", "", "project directory for project-scope installs")
	cmd.Flags().StringVar(&dir, "dir", "", "explicit directory for --target directory")
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	return cmd
}

func installedUpdateCommand() *cobra.Command {
	var target string
	var scope string
	var project string
	var dir string
	var verbose bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update managed installed skills for a target",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			scriptArgs := []string{"update"}
			if cmd.Flags().Changed("target") {
				scriptArgs = append(scriptArgs, "--target", target)
			}
			if cmd.Flags().Changed("scope") {
				scriptArgs = append(scriptArgs, "--scope", scope)
			}
			if cmd.Flags().Changed("project") {
				scriptArgs = append(scriptArgs, "--project", project)
			}
			if cmd.Flags().Changed("dir") {
				scriptArgs = append(scriptArgs, "--dir", dir)
			}
			if verbose {
				scriptArgs = append(scriptArgs, "--verbose")
			}
			return runScript(repoRoot, "scripts/installed.sh", scriptArgs...)
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "install target id; defaults to codex")
	cmd.Flags().StringVar(&scope, "scope", "", "install scope for scoped targets: global or project")
	cmd.Flags().StringVar(&project, "project", "", "project directory for project-scope installs")
	cmd.Flags().StringVar(&dir, "dir", "", "explicit directory for --target directory")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print per-skill update details")
	return cmd
}

func installedUninstallCommand() *cobra.Command {
	var target string
	var scope string
	var project string
	var dir string
	var force bool
	cmd := &cobra.Command{
		Use:   "uninstall <skill>",
		Short: "Uninstall an installed skill from a target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			scriptArgs := []string{"uninstall", args[0]}
			if cmd.Flags().Changed("target") {
				scriptArgs = append(scriptArgs, "--target", target)
			}
			if cmd.Flags().Changed("scope") {
				scriptArgs = append(scriptArgs, "--scope", scope)
			}
			if cmd.Flags().Changed("project") {
				scriptArgs = append(scriptArgs, "--project", project)
			}
			if cmd.Flags().Changed("dir") {
				scriptArgs = append(scriptArgs, "--dir", dir)
			}
			if force {
				scriptArgs = append(scriptArgs, "--force")
			}
			return runScript(repoRoot, "scripts/installed.sh", scriptArgs...)
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "install target id; defaults to codex")
	cmd.Flags().StringVar(&scope, "scope", "", "install scope for scoped targets: global or project")
	cmd.Flags().StringVar(&project, "project", "", "project directory for project-scope installs")
	cmd.Flags().StringVar(&dir, "dir", "", "explicit directory for --target directory")
	cmd.Flags().BoolVar(&force, "force", false, "remove an unmanaged skill directory")
	return cmd
}

func scriptCommand(use, short, script string, prefix []string, args cobra.PositionalArgs) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  args,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			return runScript(repoRoot, script, append(prefix, args...)...)
		},
	}
}

func installCommand(use, short string) *cobra.Command {
	var all bool
	var target string
	var scope string
	var project string
	var dir string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args: func(cmd *cobra.Command, args []string) error {
			if all {
				if len(args) != 0 {
					return fmt.Errorf("--all cannot be combined with skill names")
				}
				return nil
			}
			return cobra.MinimumNArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			scriptArgs := []string{"install"}
			if all {
				scriptArgs = append(scriptArgs, "--all")
			}
			if cmd.Flags().Changed("target") {
				scriptArgs = append(scriptArgs, "--target", target)
			}
			if cmd.Flags().Changed("scope") {
				scriptArgs = append(scriptArgs, "--scope", scope)
			}
			if cmd.Flags().Changed("project") {
				scriptArgs = append(scriptArgs, "--project", project)
			}
			if cmd.Flags().Changed("dir") {
				scriptArgs = append(scriptArgs, "--dir", dir)
			}
			if !all {
				scriptArgs = append(scriptArgs, args...)
			}
			return runScript(repoRoot, "scripts/skills.sh", scriptArgs...)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "install all cataloged skills")
	cmd.Flags().StringVar(&target, "target", "", "install target id; defaults to codex")
	cmd.Flags().StringVar(&scope, "scope", "", "install scope for scoped targets: global or project")
	cmd.Flags().StringVar(&project, "project", "", "project directory for project-scope installs")
	cmd.Flags().StringVar(&dir, "dir", "", "explicit directory for --target directory")
	return cmd
}

func runScript(repoRoot, script string, args ...string) error {
	scriptPath := filepath.Join(repoRoot, script)
	cmd := exec.Command("sh", append([]string{scriptPath}, args...)...)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	if os.Getenv("SKILLHUB_CALLER_CWD") == "" {
		callerCwd, err := os.Getwd()
		if err != nil {
			callerCwd = "."
		}
		cmd.Env = append(cmd.Env, "SKILLHUB_CALLER_CWD="+callerCwd)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func resolveRepoRoot() (string, error) {
	if repoFlag != "" {
		return cleanRepoRoot(repoFlag)
	}
	if env := os.Getenv("SKILLHUB_REPO"); env != "" {
		return cleanRepoRoot(env)
	}
	if wd, err := os.Getwd(); err == nil {
		if root, ok := findRepoRoot(wd); ok {
			return root, nil
		}
	}
	return "", fmt.Errorf("skillhub repository not found; run from the checkout or set SKILLHUB_REPO")
}

func cleanRepoRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !hasRepoFiles(abs) {
		return "", fmt.Errorf("invalid skillhub repository %s: missing required skillhub files", abs)
	}
	return abs, nil
}

func findRepoRoot(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		if hasRepoFiles(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir || strings.TrimSpace(parent) == "" {
			return "", false
		}
		dir = parent
	}
}

func hasRepoFiles(dir string) bool {
	required := []string{
		filepath.Join("defaults", "sources.tsv"),
		filepath.Join("targets", "targets.tsv"),
		filepath.Join("scripts", "skills.sh"),
		filepath.Join("scripts", "sources.sh"),
		filepath.Join("scripts", "installed.sh"),
		filepath.Join("scripts", "targets.sh"),
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			return false
		}
	}
	return true
}
