//nolint:goconst // Repeated source and skill IDs make the preflight scenarios easier to audit.
package core

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type renamePreflightFixture struct {
	backend *Backend
	source  Source
	raw     string
	project string
	custom  string
	config  string
	cache   string
}

func newRenamePreflightFixture(t *testing.T, sourceType string) renamePreflightFixture {
	t.Helper()
	// Git commands in this fixture are local-only and must not load user hooks
	// or user-level git configuration.
	gitConfig := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(gitConfig, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	project := t.TempDir()
	backend, config, cache := newSourceRegressionBackend(t, project)
	upstream := t.TempDir()
	if err := os.WriteFile(filepath.Join(upstream, "SKILL.md"), []byte("# Unnamed root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if sourceType == SourceTypeGit {
		for _, args := range [][]string{
			{"init", "--quiet", "--initial-branch=main", "--template="},
			{"add", "SKILL.md"},
			{
				"-c", "user.name=Test",
				"-c", "user.email=test@example.invalid",
				"-c", "commit.gpgsign=false",
				"commit", "--quiet", "-m", "fixture",
			},
		} {
			cmd := exec.CommandContext(context.Background(), "git", args...)
			cmd.Dir = upstream
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git fixture: %v\n%s", err, out)
			}
		}
	}
	if _, err := backend.AddSource(SourceAddOptions{Name: "alpha", Type: sourceType, Location: upstream}); err != nil {
		t.Fatal(err)
	}
	custom := t.TempDir()
	for _, opts := range []InstallOptions{
		{Names: []string{"alpha/alpha"}, Target: TargetDirectory, Dir: custom},
		{Names: []string{"alpha/alpha"}, Target: TargetCodex, Scope: ScopeProject, Project: project},
	} {
		if _, _, err := backend.Install(opts); err != nil {
			t.Fatal(err)
		}
	}
	sources, err := backend.ListSources()
	if err != nil || len(sources) != 1 {
		t.Fatalf("source fixture: %+v, %v", sources, err)
	}
	raw, err := backend.SourcePath(sources[0])
	if err != nil {
		t.Fatal(err)
	}
	return renamePreflightFixture{
		backend: backend,
		source:  sources[0],
		raw:     raw,
		project: project,
		custom:  custom,
		config:  config,
		cache:   cache,
	}
}

func (f renamePreflightFixture) snapshot(t *testing.T) map[string]string {
	t.Helper()
	state := map[string]string{}
	for _, root := range []string{f.project, f.custom, f.config, f.cache} {
		if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				state[path] = "directory"
				return nil
			}
			data, err := os.ReadFile(path) //nolint:gosec // The walked tree is a private test fixture with no concurrent writers.
			if err != nil {
				return err
			}
			state[path] = string(data)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	return state
}

func (f renamePreflightFixture) assertRejected(t *testing.T, name, reason string) {
	t.Helper()
	before := f.snapshot(t)
	summary, err := f.backend.RenameSource(SourceRenameOptions{OldName: "alpha", NewName: name})
	if err == nil || !strings.Contains(err.Error(), reason) {
		t.Fatalf("expected %q rejection, got summary=%+v err=%v", reason, summary, err)
	}
	if !reflect.DeepEqual(before, f.snapshot(t)) {
		t.Fatal("rejected rename changed registry, source config, installs, sidecars, lockfile, or cache")
	}
}

func TestRootRenameRejectsCatalogCollisionsBeforeMutation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		next   string
		path   string
		reason string
	}{
		{"nested", "beta", "skills/beta", "duplicate generated skill name beta"},
		{"plugin", "beta", "plugins/plugin/skills/beta", "would change plugin/beta from beta to plugin_beta"},
		{"legacy", "a_foo", "plugins/a/skills/foo", "ambiguous legacy skill name a_foo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRenamePreflightFixture(t, SourceTypePath)
			dir := filepath.Join(f.raw, filepath.FromSlash(tc.path))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			content := fmt.Sprintf("---\nname: %s\n---\nOther skill\n", filepath.Base(dir))
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := f.backend.SyncSources("alpha"); err != nil {
				t.Fatal(err)
			}
			f.assertRejected(t, tc.next, tc.reason)
			if _, err := f.backend.SyncSources("alpha"); err != nil {
				t.Fatalf("original source stopped working after rejected rename: %v", err)
			}
		})
	}
}

func TestRootRenameRequiresAvailableSourceAndAllowsRetry(t *testing.T) {
	for _, sourceType := range []string{SourceTypePath, SourceTypeGit} {
		t.Run(sourceType, func(t *testing.T) {
			f := newRenamePreflightFixture(t, sourceType)
			offline := filepath.Join(t.TempDir(), "offline")
			if err := os.Rename(f.raw, offline); err != nil {
				t.Fatal(err)
			}
			f.assertRejected(t, "beta", "cannot inspect source alpha before rename")
			if err := os.Rename(offline, f.raw); err != nil {
				t.Fatal(err)
			}
			if _, err := f.backend.RenameSource(SourceRenameOptions{OldName: "alpha", NewName: "beta"}); err != nil {
				t.Fatalf("retry after restoring source: %v", err)
			}
			updated, err := f.backend.UpdateInstalled(InstalledUpdateOptions{Target: TargetDirectory, Dir: f.custom})
			if err != nil || updated.Skipped != 0 || updated.Updated+updated.Unchanged != 1 {
				t.Fatalf("update after successful retry: %+v, %v", updated, err)
			}
			installed := filepath.Join(f.project, ".agents", "skills", "beta")
			if err := os.RemoveAll(installed); err != nil {
				t.Fatal(err)
			}
			restored, err := f.backend.Restore(RestoreOptions{Project: f.project})
			if err != nil || restored.Installed != 1 || restored.Skipped != 0 {
				t.Fatalf("restore after successful retry: %+v, %v", restored, err)
			}
			assertOwnedSkill(t, installed, filepath.Join(f.custom, "beta"))
		})
	}
}

func TestRootRenameDoesNotClaimExistingDestinationForMissingInstall(t *testing.T) {
	f := newRenamePreflightFixture(t, SourceTypePath)
	if err := os.RemoveAll(filepath.Join(f.custom, "alpha")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(f.custom, "beta"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.custom, "beta", "SKILL.md"), []byte("UNMANAGED"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.assertRejected(t, "beta", "destination already exists")
}
