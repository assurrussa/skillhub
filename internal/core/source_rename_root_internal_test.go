package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenameSourceMigratesAliasDerivedRootSkillID(t *testing.T) {
	projectDir := t.TempDir()
	backend, configDir, _ := newSourceRegressionBackend(t, projectDir)
	sourceDir := filepath.Join(projectDir, "root-source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	rootSkill := []byte("# Root skill without frontmatter name\n")
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), rootSkill, 0o644); err != nil {
		t.Fatalf("write root skill: %v", err)
	}
	writeRegressionSource(t, configDir, "alpha", sourceDir)

	projectRoot := filepath.Join(projectDir, ".agents", "skills")
	projectInstall := filepath.Join(projectRoot, "alpha")
	customRoot := filepath.Join(projectDir, "custom-skills")
	customInstall := filepath.Join(customRoot, "alpha")
	for _, installPath := range []string{projectInstall, customInstall} {
		if err := os.MkdirAll(installPath, 0o755); err != nil {
			t.Fatalf("mkdir install %s: %v", installPath, err)
		}
		if err := os.WriteFile(filepath.Join(installPath, "SKILL.md"), rootSkill, 0o644); err != nil {
			t.Fatalf("write installed skill %s: %v", installPath, err)
		}
	}

	customMeta := metadataFile{
		SchemaVersion:  "2",
		Source:         "alpha",
		Skill:          "alpha",
		QualifiedSkill: "alpha/alpha",
		Target:         TargetDirectory,
		Scope:          ScopeCustom,
		ProjectPath:    "-",
		TargetRoot:     customRoot,
		InstalledPath:  customInstall,
		SourceRef:      "-",
		SourceLocation: sourceDir,
		Catalog:        "catalog/skills.tsv",
		ContentHash:    "-",
		InstalledAt:    "2026-09-15T00:00:00Z",
		UpdatedAt:      "2026-09-15T00:00:00Z",
	}
	metaData, err := json.MarshalIndent(customMeta, "", "  ")
	if err != nil {
		t.Fatalf("marshal sidecar: %v", err)
	}
	metaData = append(metaData, '\n')
	if err := os.WriteFile(filepath.Join(customInstall, ".skillhub.json"), metaData, 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	rows := []string{
		strings.Join([]string{
			"alpha", "alpha", TargetCodex, ScopeProject, projectDir,
			projectRoot, projectInstall, "-", sourceDir, "catalog/skills.tsv", "-",
			"2026-09-15T00:00:00Z", "2026-09-15T00:00:00Z",
		}, "\t"),
		strings.Join([]string{
			"alpha", "alpha", TargetDirectory, ScopeCustom, "-",
			customRoot, customInstall, "-", sourceDir, "catalog/skills.tsv", "-",
			"2026-09-15T00:00:00Z", "2026-09-15T00:00:00Z",
		}, "\t"),
	}
	installed := InstalledUsageHeader + "\n" + strings.Join(rows, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "installed.tsv"), []byte(installed), 0o644); err != nil {
		t.Fatalf("write installed registry: %v", err)
	}

	lock := `lockfile_version = 1

[[skills]]
source = "alpha"
source_type = "path"
skill = "alpha"
qualified_skill = "alpha/alpha"
target = "codex"
scope = "project"
project_path = "."
target_root = ".agents/skills"
installed_path = ".agents/skills/alpha"
source_ref = "-"
source_location = "` + sourceDir + `"
catalog = "catalog/skills.tsv"
content_hash = "-"
installed_at = "2026-09-15T00:00:00Z"
updated_at = "2026-09-15T00:00:00Z"
`
	lockPath := filepath.Join(projectDir, "skills.lock.toml")
	if err := os.WriteFile(lockPath, []byte(lock), 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	summary, err := backend.RenameSource(SourceRenameOptions{OldName: "alpha", NewName: "beta"})
	if err != nil {
		t.Fatalf("rename alias-derived root source: %v", err)
	}
	if summary.InstalledUpdated != 2 || summary.LockfilesUpdated != 1 {
		t.Fatalf("unexpected rename summary: %#v", summary)
	}
	if summary.RenamedSkillOld != "alpha" || summary.RenamedSkillNew != "beta" {
		t.Fatalf("expected renamed root skill alpha->beta in summary, got %#v", summary)
	}

	for _, oldPath := range []string{projectInstall, customInstall} {
		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Fatalf("old install path must be gone after skill-id migration: %s (err=%v)", oldPath, err)
		}
	}
	projectBeta := filepath.Join(projectRoot, "beta")
	customBeta := filepath.Join(customRoot, "beta")
	for _, newPath := range []string{projectBeta, customBeta} {
		if _, err := os.Stat(filepath.Join(newPath, "SKILL.md")); err != nil {
			t.Fatalf("renamed install missing at %s: %v", newPath, err)
		}
	}

	usage, err := backend.ReadUsage("")
	if err != nil {
		t.Fatalf("read migrated usage: %v", err)
	}
	if len(usage) != 2 {
		t.Fatalf("expected two migrated usage rows, got %#v", usage)
	}
	for _, row := range usage {
		if row.Source != "beta" || row.Skill != "beta" || row.QualifiedSkill != "beta/beta" {
			t.Fatalf("root skill identity not migrated: %#v", row)
		}
		if filepath.Base(row.InstalledPath) != "beta" {
			t.Fatalf("installed path not migrated: %#v", row)
		}
	}

	metaBytes, err := os.ReadFile(filepath.Join(customBeta, ".skillhub.json"))
	if err != nil {
		t.Fatalf("read migrated sidecar: %v", err)
	}
	var migratedMeta metadataFile
	if err := json.Unmarshal(metaBytes, &migratedMeta); err != nil {
		t.Fatalf("unmarshal migrated sidecar: %v", err)
	}
	if migratedMeta.Source != "beta" || migratedMeta.Skill != "beta" || migratedMeta.QualifiedSkill != "beta/beta" || migratedMeta.InstalledPath != customBeta {
		t.Fatalf("sidecar identity/path not migrated: %#v", migratedMeta)
	}

	lockBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read migrated lockfile: %v", err)
	}
	lockText := string(lockBytes)
	for _, expected := range []string{
		`source = "beta"`,
		`skill = "beta"`,
		`qualified_skill = "beta/beta"`,
		`installed_path = ".agents/skills/beta"`,
	} {
		if !strings.Contains(lockText, expected) {
			t.Fatalf("migrated lockfile missing %s:\n%s", expected, lockText)
		}
	}

	sources, err := backend.ListSources()
	if err != nil || len(sources) != 1 || sources[0].Name != "beta" {
		t.Fatalf("renamed source not persisted: sources=%#v err=%v", sources, err)
	}
	materialized, _, err := backend.CatalogSource(sources[0])
	if err != nil {
		t.Fatalf("rematerialize renamed root source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(materialized, "skills", "beta", "SKILL.md")); err != nil {
		t.Fatalf("new root skill id is not materialized: %v", err)
	}

	restore, err := backend.Restore(RestoreOptions{Project: projectDir, Check: true})
	if err != nil {
		t.Fatalf("check restore after root skill migration: %v", err)
	}
	if restore.Skipped != 0 {
		t.Fatalf("migrated root skill must remain restorable: %#v", restore)
	}
}
