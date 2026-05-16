package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assurrussa/skillhub/internal/core"
	"github.com/assurrussa/skillhub/internal/tui"
)

const (
	UseDefault   = "defaults"
	UseList      = "list"
	UseTargets   = "targets"
	UseInstalled = "installed"
)

var repoFlag string

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "skillhub",
		Short:         "Discover and install agent skills from registered sources",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
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
	sources.AddCommand(sourceStatusCommand())
	sources.AddCommand(sourceSyncCommand())
	sources.AddCommand(sourceAddCommand())
	sources.AddCommand(sourceRemoveCommand())
	sources.AddCommand(sourceDefaultsCommand())

	targets := &cobra.Command{
		Use:   UseTargets,
		Short: "List supported and planned install targets",
	}
	targets.AddCommand(targetListCommand())
	targets.AddCommand(targetDetectCommand())

	installed := &cobra.Command{
		Use:   UseInstalled,
		Short: "Inspect installed skills",
	}
	installed.AddCommand(installedListCommand())
	installed.AddCommand(installedUpdateCommand())
	installed.AddCommand(installedUninstallCommand())
	installed.AddCommand(installedUsageCommand())

	skills := &cobra.Command{
		Use:   "skills",
		Short: "List, search, and install skills",
	}
	skills.AddCommand(skillListCommand("list", "List available skills"))
	skills.AddCommand(skillSearchCommand("search <query>", "Search available skills"))
	skills.AddCommand(installCommand("install [<source>/]<skill-name>...", "Install selected skills"))
	skills.AddCommand(restoreCommand("restore [--project <path>]", "Restore project skills from skills.lock.toml"))

	root.AddCommand(sources, targets, installed, skills)
	root.AddCommand(skillListCommand("list", "List available skills"))
	root.AddCommand(skillSearchCommand("search <query>", "Search available skills"))
	root.AddCommand(installCommand("install [<source>/]<skill-name>...", "Install selected skills"))
	root.AddCommand(installCommand("add [<source>/]<skill-name>...", "Add selected skills"))
	root.AddCommand(restoreCommand("restore [--project <path>]", "Restore project skills from skills.lock.toml"))
	root.AddCommand(recommendCommand())
	root.AddCommand(versionCommand())
	root.AddCommand(updateCommand())
	root.AddCommand(&cobra.Command{
		Use:   "tui",
		Short: "Open the interactive skill dashboard",
		RunE: func(_ *cobra.Command, _ []string) error {
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
		Use:   UseList,
		Short: "List configured skill sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			sources, err := backend.ListSources()
			if err != nil {
				return err
			}
			renderSources(cmd.OutOrStdout(), sources, tsv)
			return nil
		},
	}
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	return cmd
}

func sourceStatusCommand() *cobra.Command {
	var tsv bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show source cache status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			statuses, err := backend.ListSourceStatuses()
			if err != nil {
				return err
			}
			renderSourceStatuses(cmd.OutOrStdout(), statuses, tsv)
			return nil
		},
	}
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	return cmd
}

func sourceSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync [source-name]",
		Short: "Sync all sources or one source",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			output, err := backend.SyncSources(name)
			_, _ = fmt.Fprint(cmd.OutOrStdout(), output)
			return err
		},
	}
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
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			output, err := backend.AddSource(core.SourceAddOptions{
				Location: args[0],
				Name:     name,
				Type:     sourceType,
				Ref:      ref,
				Catalog:  catalog,
			})
			_, _ = fmt.Fprint(cmd.OutOrStdout(), output)
			return err
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
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			output, err := backend.RemoveSource(args[0])
			_, _ = fmt.Fprint(cmd.OutOrStdout(), output)
			return err
		},
	}
}

func sourceDefaultsCommand() *cobra.Command {
	defaults := &cobra.Command{
		Use:     UseDefault,
		Aliases: []string{"default"},
		Short:   "List and add recommended source presets",
	}
	var tsv bool
	list := &cobra.Command{
		Use:   UseList,
		Short: "List recommended source presets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			sources, err := backend.ListDefaultSources()
			if err != nil {
				return err
			}
			renderSources(cmd.OutOrStdout(), sources, tsv)
			return nil
		},
	}
	list.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	defaults.AddCommand(list)
	defaults.AddCommand(&cobra.Command{
		Use:   "add <source-name>",
		Short: "Add a recommended source preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			output, err := backend.AddDefaultSource(args[0])
			_, _ = fmt.Fprint(cmd.OutOrStdout(), output)
			return err
		},
	})
	return defaults
}

func targetListCommand() *cobra.Command {
	var tsv bool
	cmd := &cobra.Command{
		Use:   UseList,
		Short: "List install targets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			targets, err := backend.ListTargets()
			if err != nil {
				return err
			}
			renderTargets(cmd.OutOrStdout(), targets, tsv)
			return nil
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			detections, err := backend.DetectTargets(project)
			if err != nil {
				return err
			}
			renderTargetDetections(cmd.OutOrStdout(), detections, tsv)
			return nil
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			rows, err := backend.ListInstalled(core.InstalledListOptions{
				Target:  target,
				Scope:   scope,
				Project: project,
				Dir:     dir,
			})
			if err != nil {
				return err
			}
			renderInstalled(cmd.OutOrStdout(), rows, tsv)
			return nil
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			summary, err := backend.UpdateInstalled(core.InstalledUpdateOptions{
				Target:  target,
				Scope:   scope,
				Project: project,
				Dir:     dir,
				Verbose: verbose,
			})
			_, _ = fmt.Fprint(cmd.OutOrStdout(), summary.Output)
			return err
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
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			output, err := backend.Uninstall(core.UninstallOptions{
				Skill:   args[0],
				Target:  target,
				Scope:   scope,
				Project: project,
				Dir:     dir,
				Force:   force,
			})
			_, _ = fmt.Fprint(cmd.OutOrStdout(), output)
			return err
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "install target id; defaults to codex")
	cmd.Flags().StringVar(&scope, "scope", "", "install scope for scoped targets: global or project")
	cmd.Flags().StringVar(&project, "project", "", "project directory for project-scope installs")
	cmd.Flags().StringVar(&dir, "dir", "", "explicit directory for --target directory")
	cmd.Flags().BoolVar(&force, "force", false, "remove an unmanaged skill directory")
	return cmd
}

func installedUsageCommand() *cobra.Command {
	var tsv bool
	cmd := &cobra.Command{
		Use:   "usage [[<source>/]<skill>]",
		Short: "Show where managed skills are used",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			filter := ""
			if len(args) > 0 {
				filter = args[0]
			}
			rows, err := backend.ReadUsage(filter)
			if err != nil {
				return err
			}
			renderUsage(cmd.OutOrStdout(), rows, tsv)
			return nil
		},
	}
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	cmd.AddCommand(installedUsageUpdateCommand())
	return cmd
}

func installedUsageUpdateCommand() *cobra.Command {
	var projects bool
	var verbose bool
	var target string
	var project string
	cmd := &cobra.Command{
		Use:   "update [[<source>/]<skill>...]",
		Short: "Update managed project-scope skills recorded in usage registry",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			summary, err := backend.UpdateUsage(core.UsageUpdateOptions{
				Projects: projects,
				Target:   target,
				Project:  project,
				Filters:  args,
				Verbose:  verbose,
			})
			_, _ = fmt.Fprint(cmd.OutOrStdout(), summary.Output)
			return err
		},
	}
	cmd.Flags().BoolVar(&projects, "projects", false, "update project-scope installs recorded in usage registry")
	cmd.Flags().StringVar(&target, "target", "", "limit project usage update to a recorded target")
	cmd.Flags().StringVar(&project, "project", "", "limit project usage update to a recorded project path")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print per-skill update details")
	return cmd
}

func recommendCommand() *cobra.Command {
	var project string
	var tsv bool
	cmd := &cobra.Command{
		Use:   "recommend",
		Short: "Recommend a minimal skill set for a project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			rows, warning, err := backend.Recommend(core.RecommendOptions{Project: project})
			if warning != "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), warning)
			}
			if err != nil {
				return err
			}
			projectLabel := project
			if projectLabel == "" {
				projectLabel, _ = backend.ProjectDir("")
			}
			renderRecommend(cmd.OutOrStdout(), rows, projectLabel, tsv)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project directory to analyze; defaults to caller working directory")
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	return cmd
}

func skillListCommand(use, short string) *cobra.Command {
	var tsv bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			skills, warning, err := backend.ListSkills("")
			if warning != "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), warning)
			}
			if err != nil {
				return err
			}
			renderSkills(cmd.OutOrStdout(), skills, tsv)
			return nil
		},
	}
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	return cmd
}

func skillSearchCommand(use, short string) *cobra.Command {
	var tsv bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			skills, warning, err := backend.ListSkills(strings.Join(args, " "))
			if warning != "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), warning)
			}
			if err != nil {
				return err
			}
			renderSkills(cmd.OutOrStdout(), skills, tsv)
			return nil
		},
	}
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated output")
	return cmd
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
					return errors.New("--all cannot be combined with skill names")
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
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			output, warning, err := backend.Install(core.InstallOptions{
				All:     all,
				Names:   args,
				Target:  target,
				Scope:   scope,
				Project: project,
				Dir:     dir,
			})
			if warning != "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), warning)
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), output)
			return err
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "install all cataloged skills")
	cmd.Flags().StringVar(&target, "target", "", "install target id; defaults to codex")
	cmd.Flags().StringVar(&scope, "scope", "", "install scope for scoped targets: global or project")
	cmd.Flags().StringVar(&project, "project", "", "project directory for project-scope installs")
	cmd.Flags().StringVar(&dir, "dir", "", "explicit directory for --target directory")
	return cmd
}

func restoreCommand(use, short string) *cobra.Command {
	var project string
	var check bool
	var tsv bool
	var verbose bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			if tsv {
				summary, err := backend.Restore(core.RestoreOptions{Project: project, Check: true, Verbose: verbose})
				renderRestoreRows(cmd.OutOrStdout(), summary.Rows)
				return err
			}
			summary, err := backend.Restore(core.RestoreOptions{Project: project, Check: check, Verbose: verbose})
			if len(summary.Rows) == 0 {
				projectPath, _ := backend.ProjectDir(project)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(),
					"No project lockfile found: %s\n",
					filepath.Join(projectPath, "skills.lock.toml"))
				return err
			}
			if verbose {
				for _, row := range summary.Rows {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s/%s %s: %s\n",
						row.Status, row.Source, row.Skill, row.Target, row.Reason)
				}
			}
			if check {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(),
					"Project lockfile check: missing=%d changed=%d unchanged=%d skipped=%d failed=%d\n",
					summary.Installed, summary.Updated, summary.Unchanged, summary.Skipped, summary.Failed)
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(),
					"Restored project skills: installed=%d updated=%d unchanged=%d skipped=%d failed=%d\n",
					summary.Installed, summary.Updated, summary.Unchanged, summary.Skipped, summary.Failed)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project directory containing skills.lock.toml")
	cmd.Flags().BoolVar(&check, "check", false, "check lockfile status without mutating project skills")
	cmd.Flags().BoolVar(&tsv, "tsv", false, "print tab-separated restore status")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print per-skill restore details")
	return cmd
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
	return "", errors.New("skillhub repository not found; run from the checkout or set SKILLHUB_REPO")
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
		"go.mod",
		filepath.Join("cmd", "skillhub", "main.go"),
		filepath.Join("defaults", "sources.tsv"),
		filepath.Join("targets", "targets.tsv"),
	}
	for _, rel := range required {
		// #nosec G703 -- rel comes from the fixed repository file list above.
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			return false
		}
	}
	return true
}

func renderSources(out io.Writer, sources []core.Source, tsv bool) {
	if tsv {
		_, _ = fmt.Fprintln(out, core.SourcesHeader)
		for _, row := range sources {
			_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", row.Name, row.Type, row.Location, row.Ref, row.Catalog)
		}
		return
	}
	_, _ = fmt.Fprintf(out, "%-20s %-8s %-48s %-12s %s\n",
		"name", "type", "location", "ref", "catalog")
	_, _ = fmt.Fprintf(out, "%-20s %-8s %-48s %-12s %s\n",
		"--------------------", "--------", "------------------------------------------------", "------------", "-------")
	for _, row := range sources {
		_, _ = fmt.Fprintf(out, "%-20s %-8s %-48s %-12s %s\n", row.Name, row.Type, row.Location, row.Ref, row.Catalog)
	}
	if len(sources) == 0 {
		_, _ = fmt.Fprintln(out, "No sources configured. Run: skillhub sources defaults list")
	}
}

func renderSourceStatuses(out io.Writer, statuses []core.SourceStatus, tsv bool) {
	if tsv {
		_, _ = fmt.Fprintln(out, core.SourceStatusesHeader)
		for _, row := range statuses {
			_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				sanitizeTSVField(row.Name), sanitizeTSVField(row.Type), sanitizeTSVField(row.Status),
				sanitizeTSVField(formatSourceStatusTime(row.LastSyncedAt)), sanitizeTSVField(row.CachePath),
				sanitizeTSVField(row.Ref), sanitizeTSVField(row.Catalog), sanitizeTSVField(row.Location),
				sanitizeTSVField(row.Message))
		}
		return
	}
	_, _ = fmt.Fprintf(out, "%-20s %-8s %-8s %-20s %-36s %s\n",
		"name", "type", "status", "last_synced_at", "cache_path", "message")
	_, _ = fmt.Fprintf(out, "%-20s %-8s %-8s %-20s %-36s %s\n",
		"--------------------", "--------", "--------", "--------------------",
		"------------------------------------", "-------")
	for _, row := range statuses {
		_, _ = fmt.Fprintf(out, "%-20s %-8s %-8s %-20s %-36s %s\n",
			row.Name,
			row.Type,
			row.Status,
			formatSourceStatusTime(row.LastSyncedAt),
			truncateTableCell(row.CachePath, 36),
			row.Message,
		)
		_, _ = fmt.Fprintf(out, "%-20s %-8s %-8s %-20s %-36s ref=%s catalog=%s location=%s\n",
			"", "", "", "", "", row.Ref, row.Catalog, row.Location)
	}
	if len(statuses) == 0 {
		_, _ = fmt.Fprintln(out, "No sources configured. Run: skillhub sources defaults list")
	}
}

func formatSourceStatusTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}

func sanitizeTSVField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}

func truncateTableCell(value string, width int) string {
	if width <= 0 || len(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	if width <= 3 {
		return value[:width]
	}
	return value[:width-3] + "..."
}

func renderTargets(out io.Writer, targets []core.Target, tsv bool) {
	if tsv {
		_, _ = fmt.Fprintln(out, core.TargetsHeader)
		for _, row := range targets {
			_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", row.ID, row.Label, row.Status, row.Adapter, row.Description)
		}
		return
	}
	_, _ = fmt.Fprintf(out,
		"%-14s %-12s %-10s %-10s %s\n",
		"id", "label", "status", "adapter", "description")
	_, _ = fmt.Fprintf(out,
		"%-14s %-12s %-10s %-10s %s\n",
		"--------------", "------------", "----------", "----------", "-----------")
	for _, row := range targets {
		_, _ = fmt.Fprintf(out, "%-14s %-12s %-10s %-10s %s\n", row.ID, row.Label, row.Status, row.Adapter, row.Description)
	}
}

func renderTargetDetections(out io.Writer, rows []core.TargetDetection, tsv bool) {
	if tsv {
		_, _ = fmt.Fprintln(out, core.TargetDetectionsHeader)
		for _, row := range rows {
			_, _ = fmt.Fprintf(out,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				row.Target, row.Scope, row.Status, row.Path, row.Exists, row.Skills, row.Managed)
		}
		return
	}
	_, _ = fmt.Fprintf(out,
		"%-12s %-8s %-10s %-6s %-6s %-7s %s\n",
		"target", "scope", "status", "exists", "skills", "managed", "path")
	_, _ = fmt.Fprintf(out,
		"%-12s %-8s %-10s %-6s %-6s %-7s %s\n",
		"------------", "--------", "----------", "------", "------", "-------", "----")
	for _, row := range rows {
		_, _ = fmt.Fprintf(out,
			"%-12s %-8s %-10s %-6s %-6s %-7s %s\n",
			row.Target, row.Scope, row.Status, row.Exists, row.Skills, row.Managed, row.Path)
	}
}

func renderSkills(out io.Writer, skills []core.Skill, tsv bool) {
	if tsv {
		_, _ = fmt.Fprintln(out, core.SkillsHeader)
		for _, row := range skills {
			_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", row.Source, row.Name, row.Category, row.Triggers, row.Description)
		}
		return
	}
	_, _ = fmt.Fprintf(out, "%-20s %-28s %-16s %s\n", "source", "name", "category", "description")
	_, _ = fmt.Fprintf(out,
		"%-20s %-28s %-16s %s\n",
		"--------------------", "----------------------------", "----------------", "-----------")
	for _, row := range skills {
		_, _ = fmt.Fprintf(out, "%-20s %-28s %-16s %s\n", row.Source, row.Name, row.Category, row.Description)
	}
}

func renderInstalled(out io.Writer, rows []core.InstalledSkill, tsv bool) {
	if tsv {
		_, _ = fmt.Fprintln(out, core.InstalledHeader)
		for _, row := range rows {
			_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				row.Target, row.Scope, row.Skill, row.Managed, row.Source,
				row.QualifiedSkill, row.InstalledPath, row.ContentHash, row.InstalledAt, row.Path)
		}
		return
	}
	_, _ = fmt.Fprintf(out, "%-12s %-8s %-28s %-8s %-20s %s\n", "target", "scope", "skill", "managed", "source", "path")
	_, _ = fmt.Fprintf(out,
		"%-12s %-8s %-28s %-8s %-20s %s\n",
		"------------", "--------", "----------------------------", "--------", "--------------------", "----")
	for _, row := range rows {
		_, _ = fmt.Fprintf(out, "%-12s %-8s %-28s %-8s %-20s %s\n", row.Target, row.Scope, row.Skill, row.Managed, row.Source, row.Path)
	}
}

func renderUsage(out io.Writer, rows []core.InstalledSkill, tsv bool) {
	if tsv {
		_, _ = fmt.Fprintln(out, core.InstalledUsageHeader)
		for _, row := range rows {
			_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				row.Source, row.Skill, row.Target, row.Scope, row.ProjectPath, row.TargetRoot,
				row.InstalledPath, row.SourceRef, row.SourceLocation, row.Catalog,
				row.ContentHash, row.InstalledAt, row.UpdatedAt)
		}
		return
	}
	_, _ = fmt.Fprintf(out, "%-20s %-28s %-10s %-8s %-32s %s\n",
		"source", "skill", "target", "scope", "project", "installed_path")
	_, _ = fmt.Fprintf(out, "%-20s %-28s %-10s %-8s %-32s %s\n",
		"--------------------", "----------------------------", "----------", "--------",
		"--------------------------------", "--------------")
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(out, "No managed skill usage recorded.")
		return
	}
	for _, row := range rows {
		_, _ = fmt.Fprintf(out, "%-20s %-28s %-10s %-8s %-32s %s\n",
			row.Source, row.Skill, row.Target, row.Scope, row.ProjectPath, row.InstalledPath)
	}
}

func renderRestoreRows(out io.Writer, rows []core.RestoreRow) {
	_, _ = fmt.Fprintln(out, core.RestoreCheckHeader)
	for _, row := range rows {
		_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Source, row.Skill, row.Target, row.Status, row.InstalledPath, row.ContentHash, row.Reason)
	}
}

func renderRecommend(out io.Writer, rows []core.RecommendRow, project string, tsv bool) {
	if tsv {
		_, _ = fmt.Fprintln(out, core.RecommendHeader)
		for _, row := range rows {
			_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", row.Source, row.Skill, row.Reason, row.InstallArg)
		}
		return
	}
	if len(rows) == 0 {
		if project == "" {
			project = "."
		}
		_, _ = fmt.Fprintf(out, "No shared skills recommended for %s.\n", project)
		_, _ = fmt.Fprintln(out, "Reason: no supported project signals matched the active source catalog.")
		return
	}
	_, _ = fmt.Fprintf(out, "Recommended shared skills for %s:\n", project)
	for _, row := range rows {
		_, _ = fmt.Fprintf(out, "- %s/%s (score %d): %s\n", row.Source, row.Skill, row.Score, row.Reason)
	}
	_, _ = fmt.Fprintln(out, "\nInstall:")
	args := make([]string, 0, len(rows))
	for _, row := range rows {
		args = append(args, row.InstallArg)
	}
	_, _ = fmt.Fprintf(out,
		"skillhub install --target codex --scope project --project %s %s\n",
		shellQuote(project), strings.Join(args, " "))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
