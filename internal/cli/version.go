package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "skillhub %s\n", version)
			fmt.Fprintf(out, "commit: %s\n", commit)
			fmt.Fprintf(out, "built: %s\n", built)
			fmt.Fprintf(out, "repo: %s\n", emptyValue(repoPath))
			fmt.Fprintf(out, "install_script: %s\n", emptyValue(installScript))
			return nil
		},
	}
}

func updateCommand() *cobra.Command {
	var binDir string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update skillhub from its installed checkout",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			script, err := resolveInstallScript()
			if err != nil {
				return err
			}

			scriptArgs := []string{script, "--update"}
			if binDir != "" {
				scriptArgs = append(scriptArgs, "--bin-dir", binDir)
			}

			updateCmd := exec.Command("sh", scriptArgs...)
			updateCmd.Dir = filepath.Dir(script)
			updateCmd.Env = os.Environ()
			updateCmd.Stdout = cmd.OutOrStdout()
			updateCmd.Stderr = cmd.ErrOrStderr()
			updateCmd.Stdin = os.Stdin
			return updateCmd.Run()
		},
	}
	cmd.Flags().StringVar(&binDir, "bin-dir", "", "install directory for the skillhub command")
	return cmd
}

func resolveInstallScript() (string, error) {
	if installScript != "" {
		if _, err := os.Stat(installScript); err == nil {
			return installScript, nil
		}
	}

	repoRoot, err := resolveRepoRoot()
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
