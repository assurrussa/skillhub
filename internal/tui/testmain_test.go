package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMain gives the whole TUI test package an isolated Skillhub runtime by
// default. Individual tests can still override these variables with t.Setenv,
// but no test can accidentally fall back to the developer's real config/cache.
func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "skillhub-tui-tests-*")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create TUI test runtime: %v\n", err)
		os.Exit(1)
	}

	configDir := filepath.Join(root, "config")
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create TUI test config: %v\n", err)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create TUI test cache: %v\n", err)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}
	if err := os.Setenv("SKILLHUB_CONFIG_DIR", configDir); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "set TUI test config: %v\n", err)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}
	if err := os.Setenv("SKILLHUB_CACHE_DIR", cacheDir); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "set TUI test cache: %v\n", err)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(root)
	os.Exit(code)
}
