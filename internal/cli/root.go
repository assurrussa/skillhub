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
		Short:         "Discover and install Codex skills from registered sources",
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
	sources.AddCommand(scriptCommand("list", "List configured skill sources", "scripts/sources.sh", []string{"list"}, cobra.NoArgs))
	sources.AddCommand(scriptCommand("sync [source-name]", "Sync all sources or one source", "scripts/sources.sh", []string{"sync"}, cobra.MaximumNArgs(1)))

	skills := &cobra.Command{
		Use:   "skills",
		Short: "List, search, and install skills",
	}
	skills.AddCommand(scriptCommand("list", "List available skills", "scripts/skills.sh", []string{"list"}, cobra.NoArgs))
	skills.AddCommand(scriptCommand("search <query>", "Search available skills", "scripts/skills.sh", []string{"search"}, cobra.MinimumNArgs(1)))
	skills.AddCommand(installCommand("install <skill-name>...", "Install selected skills"))

	root.AddCommand(sources, skills)
	root.AddCommand(scriptCommand("list", "List available skills", "scripts/skills.sh", []string{"list"}, cobra.NoArgs))
	root.AddCommand(scriptCommand("search <query>", "Search available skills", "scripts/skills.sh", []string{"search"}, cobra.MinimumNArgs(1)))
	root.AddCommand(installCommand("install <skill-name>...", "Install selected skills"))
	root.AddCommand(&cobra.Command{
		Use:   "tui",
		Short: "Open the interactive skill selector",
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
			} else {
				scriptArgs = append(scriptArgs, args...)
			}
			return runScript(repoRoot, "scripts/skills.sh", scriptArgs...)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "install all cataloged skills")
	return cmd
}

func runScript(repoRoot, script string, args ...string) error {
	scriptPath := filepath.Join(repoRoot, script)
	cmd := exec.Command("sh", append([]string{scriptPath}, args...)...)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
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
	if _, err := os.Stat(filepath.Join(abs, "sources", "sources.tsv")); err != nil {
		return "", fmt.Errorf("invalid skillhub repository %s: %w", abs, err)
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
		filepath.Join("sources", "sources.tsv"),
		filepath.Join("scripts", "skills.sh"),
		filepath.Join("scripts", "sources.sh"),
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			return false
		}
	}
	return true
}
