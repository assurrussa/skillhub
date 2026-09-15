package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newSourceRegressionBackend(t *testing.T, cwd string) (*Backend, string, string) {
	t.Helper()
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	backend, err := New(Context{
		CallerCWD: cwd,
		Env: map[string]string{
			"SKILLHUB_CONFIG_DIR": configDir,
			"SKILLHUB_CACHE_DIR":  cacheDir,
		},
	})
	if err != nil {
		t.Fatalf("new backend: %v", err)
	}
	return backend, configDir, cacheDir
}

func writeRegressionSource(t *testing.T, configDir, name, location string) {
	t.Helper()
	if err := os.MkdirAll(location, 0o755); err != nil {
		t.Fatalf("mkdir source fixture: %v", err)
	}
	content := SourcesHeader + "\n" + name + "\tpath\t" + location + "\t-\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(content), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
}

func TestRenameSourcePreflightsInstalledRegistry(t *testing.T) {
	projectDir := t.TempDir()
	backend, configDir, _ := newSourceRegressionBackend(t, projectDir)
	writeRegressionSource(t, configDir, "alpha", filepath.Join(projectDir, "source"))
	original, err := os.ReadFile(filepath.Join(configDir, "sources.tsv"))
	if err != nil {
		t.Fatalf("read original sources: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "installed.tsv"), []byte("broken\n"), 0o644); err != nil {
		t.Fatalf("write malformed installed registry: %v", err)
	}

	if _, err := backend.RenameSource(SourceRenameOptions{OldName: "alpha", NewName: "beta"}); err == nil {
		t.Fatal("expected malformed installed registry to reject rename")
	}
	after, err := os.ReadFile(filepath.Join(configDir, "sources.tsv"))
	if err != nil {
		t.Fatalf("read sources after failed rename: %v", err)
	}
	if string(after) != string(original) {
		t.Fatalf("failed rename mutated sources.tsv:\n%s", after)
	}
}

func TestRenameSourcePreservesUnrelatedLockfileRows(t *testing.T) {
	projectDir := t.TempDir()
	backend, configDir, _ := newSourceRegressionBackend(t, projectDir)
	sourceDir := filepath.Join(projectDir, "source")
	writeRegressionSource(t, configDir, "alpha", sourceDir)

	installedPath := filepath.Join(projectDir, ".agents", "skills", "skill-a")
	targetRoot := filepath.Dir(installedPath)
	installed := InstalledUsageHeader + "\n" + strings.Join([]string{
		"alpha",
		"skill-a",
		TargetCodex,
		ScopeProject,
		projectDir,
		targetRoot,
		installedPath,
		"-",
		sourceDir,
		"catalog/skills.tsv",
		"hash-a",
		"2026-09-15T00:00:00Z",
		"2026-09-15T00:00:00Z",
	}, "\t") + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "installed.tsv"), []byte(installed), 0o644); err != nil {
		t.Fatalf("write installed registry: %v", err)
	}

	lock := `lockfile_version = 1

[[skills]]
source = "alpha"
skill = "skill-a"
qualified_skill = "alpha/skill-a"

[[skills]]
source = "other"
skill = "skill-b"
qualified_skill = "other/skill-b"
`
	lockPath := filepath.Join(projectDir, "skills.lock.toml")
	if err := os.WriteFile(lockPath, []byte(lock), 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	summary, err := backend.RenameSource(SourceRenameOptions{OldName: "alpha", NewName: "beta"})
	if err != nil {
		t.Fatalf("rename source: %v", err)
	}
	if summary.InstalledUpdated != 1 || summary.LockfilesUpdated != 1 {
		t.Fatalf("unexpected rename summary: %#v", summary)
	}
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lockfile: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `source = "beta"`) || !strings.Contains(text, `qualified_skill = "beta/skill-a"`) {
		t.Fatalf("renamed row not updated:\n%s", text)
	}
	if !strings.Contains(text, `source = "other"`) || !strings.Contains(text, `qualified_skill = "other/skill-b"`) {
		t.Fatalf("unrelated lockfile row was lost:\n%s", text)
	}
}

func TestRenameSourceSkipsMissingInstallSidecar(t *testing.T) {
	projectDir := t.TempDir()
	backend, configDir, _ := newSourceRegressionBackend(t, projectDir)
	sourceDir := filepath.Join(projectDir, "source")
	writeRegressionSource(t, configDir, "alpha", sourceDir)

	installedPath := filepath.Join(projectDir, "missing-global", "skill-a")
	installed := InstalledUsageHeader + "\n" + strings.Join([]string{
		"alpha",
		"skill-a",
		TargetDirectory,
		ScopeCustom,
		"-",
		filepath.Dir(installedPath),
		installedPath,
		"-",
		sourceDir,
		"catalog/skills.tsv",
		"hash-a",
		"2026-09-15T00:00:00Z",
		"2026-09-15T00:00:00Z",
	}, "\t") + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "installed.tsv"), []byte(installed), 0o644); err != nil {
		t.Fatalf("write installed registry: %v", err)
	}

	summary, err := backend.RenameSource(SourceRenameOptions{OldName: "alpha", NewName: "beta"})
	if err != nil {
		t.Fatalf("rename source with stale install: %v", err)
	}
	if summary.InstalledUpdated != 1 {
		t.Fatalf("expected stale registry row to be renamed, got %#v", summary)
	}
	usage, err := backend.ReadUsage("")
	if err != nil {
		t.Fatalf("read usage: %v", err)
	}
	if len(usage) != 1 || usage[0].Source != "beta" || usage[0].QualifiedSkill != "beta/skill-a" {
		t.Fatalf("stale registry row not renamed: %#v", usage)
	}
	if _, err := os.Stat(filepath.Join(installedPath, ".skillhub.json")); !os.IsNotExist(err) {
		t.Fatalf("rename must not recreate metadata under missing install, stat err=%v", err)
	}
}

func TestGitHubSourceNamesIncludeOwner(t *testing.T) {
	tests := map[string]string{
		"https://github.com/onmax/nuxt-skills":     "onmax-nuxt-skills",
		"https://github.com/humanlayer/skills.git": "humanlayer-skills",
		"git@github.com:antfu/skills.git":          "antfu-skills",
		"ssh://git@github.com/vueuse/skills.git":   "vueuse-skills",
		"https://git.example.test/team/skills.git": "skills",
	}
	for input, want := range tests {
		if got := deriveSourceName(input); got != want {
			t.Fatalf("deriveSourceName(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestPluginBundleMaterializationPreservesSkillNameAndLegacyAlias(t *testing.T) {
	projectDir := t.TempDir()
	backend, configDir, _ := newSourceRegressionBackend(t, projectDir)
	sourceDir := filepath.Join(projectDir, "humanlayer-skills")
	skillDir := filepath.Join(sourceDir, "plugins", "improve-claude-md", "skills", "improve-claude-md")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin skill: %v", err)
	}
	writeSkill := func(body string) {
		t.Helper()
		skill := "---\nname: improve-claude-md\ndescription: Improve CLAUDE.md\n---\n\n# Improve\n\n" + body + "\n"
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skill), 0o644); err != nil {
			t.Fatalf("write plugin skill: %v", err)
		}
	}
	writeSkill("v1")
	writeRegressionSource(t, configDir, "humanlayer", sourceDir)

	materialized, _, err := backend.CatalogSource(Source{
		Name:     "humanlayer",
		Type:     SourceTypePath,
		Location: sourceDir,
		Ref:      "-",
		Catalog:  "catalog/skills.tsv",
	})
	if err != nil {
		t.Fatalf("materialize plugin source: %v", err)
	}
	catalog, err := os.ReadFile(filepath.Join(materialized, "catalog", "skills.tsv"))
	if err != nil {
		t.Fatalf("read generated catalog: %v", err)
	}
	text := string(catalog)
	if !strings.Contains(text, "improve-claude-md\timprove-claude-md\t") {
		t.Fatalf("expected canonical plugin skill name, got:\n%s", text)
	}
	legacyName := "improve-claude-md_improve-claude-md"
	if !strings.Contains(text, legacyName+"\t"+generatedSkillCompatibilityCategory+"\t") {
		t.Fatalf("expected hidden legacy compatibility alias, got:\n%s", text)
	}
	if _, err := os.Stat(filepath.Join(materialized, "skills", "improve-claude-md", "SKILL.md")); err != nil {
		t.Fatalf("expected canonical generated skill directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(materialized, "skills", legacyName, "SKILL.md")); err != nil {
		t.Fatalf("expected legacy compatibility directory: %v", err)
	}

	skills, warning, err := backend.ListSkills("")
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if warning != "" {
		t.Fatalf("unexpected list warning: %s", warning)
	}
	if len(skills) != 1 || skills[0].Name != "improve-claude-md" {
		t.Fatalf("compatibility alias must stay hidden from discovery: %#v", skills)
	}

	targetDir := filepath.Join(projectDir, "legacy-install")
	if _, _, err := backend.Install(InstallOptions{
		Names:  []string{"humanlayer/" + legacyName},
		Target: TargetDirectory,
		Dir:    targetDir,
	}); err != nil {
		t.Fatalf("install legacy alias: %v", err)
	}
	legacyInstalled := filepath.Join(targetDir, legacyName, "SKILL.md")
	if _, err := os.Stat(legacyInstalled); err != nil {
		t.Fatalf("expected legacy install path: %v", err)
	}

	writeSkill("v2")
	update, err := backend.UpdateInstalled(InstalledUpdateOptions{Target: TargetDirectory, Dir: targetDir})
	if err != nil {
		t.Fatalf("update legacy alias install: %v", err)
	}
	if update.Updated != 1 {
		t.Fatalf("expected legacy install to update, got %#v", update)
	}
	updatedContent, err := os.ReadFile(legacyInstalled)
	if err != nil {
		t.Fatalf("read updated legacy install: %v", err)
	}
	if !strings.Contains(string(updatedContent), "v2") {
		t.Fatalf("legacy install was not updated from canonical source:\n%s", updatedContent)
	}

	restoreProject := filepath.Join(projectDir, "restore-project")
	if err := os.MkdirAll(restoreProject, 0o755); err != nil {
		t.Fatalf("mkdir restore project: %v", err)
	}
	lock := `lockfile_version = 1

[[skills]]
source = "humanlayer"
source_type = "path"
skill = "` + legacyName + `"
qualified_skill = "humanlayer/` + legacyName + `"
target = "codex"
scope = "project"
project_path = "."
target_root = ".agents/skills"
installed_path = ".agents/skills/` + legacyName + `"
source_ref = "-"
source_location = "` + sourceDir + `"
catalog = "catalog/skills.tsv"
content_hash = "-"
installed_at = "-"
updated_at = "-"
`
	if err := os.WriteFile(filepath.Join(restoreProject, "skills.lock.toml"), []byte(lock), 0o644); err != nil {
		t.Fatalf("write legacy lockfile: %v", err)
	}
	restored, err := backend.Restore(RestoreOptions{Project: restoreProject})
	if err != nil {
		t.Fatalf("restore legacy alias: %v", err)
	}
	if restored.Installed != 1 {
		t.Fatalf("expected legacy alias restore install, got %#v", restored)
	}
	if _, err := os.Stat(filepath.Join(restoreProject, ".agents", "skills", legacyName, "SKILL.md")); err != nil {
		t.Fatalf("legacy lockfile was not restorable: %v", err)
	}
}
