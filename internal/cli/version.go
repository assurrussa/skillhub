package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/assurrussa/skillhub/internal/core"
)

var (
	version       = "dev"
	commit        = "unknown"
	built         = "unknown"
	repoPath      = ""
	installScript = ""
)

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print skillhub version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "skillhub %s\n", version)
			_, _ = fmt.Fprintf(out, "commit: %s\n", commit)
			_, _ = fmt.Fprintf(out, "built: %s\n", built)
			_, _ = fmt.Fprintf(out, "repo: %s\n", emptyValue(repoPath))
			_, _ = fmt.Fprintf(out, "install_script: %s\n", emptyValue(installScript))
			return nil
		},
	}
}

func updateCommand() *cobra.Command {
	var binDir string
	var cascade bool
	var verbose bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update skillhub from its installed checkout",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			script, err := resolveInstallScript()
			if err != nil {
				return err
			}

			scriptArgs := []string{script, "--update"}
			if binDir != "" {
				scriptArgs = append(scriptArgs, "--bin-dir", binDir)
			}

			updateCmd := exec.CommandContext(context.Background(), "sh", scriptArgs...)
			updateCmd.Dir = filepath.Dir(script)
			updateCmd.Env = os.Environ()
			updateCmd.Stdout = cmd.OutOrStdout()
			updateCmd.Stderr = cmd.ErrOrStderr()
			updateCmd.Stdin = os.Stdin
			if err := updateCmd.Run(); err != nil {
				return err
			}

			if !cascade {
				return nil
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintln(out, "Updating managed installed skills...")
			repoRoot := filepath.Dir(script)
			backend, err := core.NewDefault(repoRoot)
			if err != nil {
				return err
			}
			summary, err := backend.UpdateInstalled(core.InstalledUpdateOptions{AllSupported: true, Verbose: verbose})
			_, _ = fmt.Fprint(out, summary.Output)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(out, "Updating recorded project usage...")
			usageSummary, err := backend.UpdateUsage(core.UsageUpdateOptions{Projects: true, Verbose: verbose})
			_, _ = fmt.Fprint(out, usageSummary.Output)
			return err
		},
	}
	cmd.Flags().StringVar(&binDir, "bin-dir", "", "install directory for the skillhub command")
	cmd.Flags().BoolVar(&cascade, "cascade", false, "also update managed installed skills after updating skillhub")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print per-skill cascade update details")
	return cmd
}

func resolveInstallScript() (string, error) {
	if installScript != "" {
		if _, err := os.Stat(installScript); err == nil {
			return installScript, nil
		}
	}

	repoRoot, err := resolveRequiredRepoRoot()
	if err != nil {
		return "", err
	}
	script := filepath.Join(repoRoot, "install.sh")
	if _, err := os.Stat(script); err != nil {
		return "", fmt.Errorf("install script not found: %w", err)
	}
	return script, nil
}

func emptyValue(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
