package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeOwnedPluginSkill(t *testing.T, sourceDir, plugin, name, body string) string {
	t.Helper()
	dir := filepath.Join(sourceDir, "plugins", plugin, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + name + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func assertOwnedSkill(t *testing.T, dir, sourceDir string) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join(sourceDir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("skill identity changed at %s: got %q, want %q", dir, got, want)
	}
}

func TestPluginLegacyIDsKeepTheirOwnersThroughUpdateAndRestore(t *testing.T) {
	project := t.TempDir()
	backend, _, _ := newSourceRegressionBackend(t, project)
	sourceDir := t.TempDir()
	a := writeOwnedPluginSkill(t, sourceDir, "a", "foo", "OWNER A v1")
	b := writeOwnedPluginSkill(t, sourceDir, "b", "a_foo", "OWNER B v1")
	owners := map[string]string{"a_foo": a, "b_a_foo": b}

	// Emulate a pre-upgrade catalog and installations, not fresh canonical ones.
	for id, dir := range owners {
		if err := copyDir(dir, filepath.Join(sourceDir, "skills", id)); err != nil {
			t.Fatal(err)
		}
	}
	catalogFile := filepath.Join(sourceDir, "catalog", "skills.tsv")
	if err := CatalogTable.WriteFile(catalogFile, []CatalogRow{
		{Name: "a_foo", Category: "a", Triggers: "foo", Description: "Owner A"},
		{Name: "b_a_foo", Category: "b", Triggers: "a_foo", Description: "Owner B"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.AddSource(SourceAddOptions{Name: "plugins", Type: SourceTypePath, Location: sourceDir}); err != nil {
		t.Fatal(err)
	}
	custom := t.TempDir()
	for _, opts := range []InstallOptions{
		{All: true, Target: TargetDirectory, Dir: custom},
		{All: true, Target: TargetCodex, Scope: ScopeProject, Project: project},
	} {
		if _, _, err := backend.Install(opts); err != nil {
			t.Fatal(err)
		}
	}
	lock, err := os.ReadFile(filepath.Join(project, "skills.lock.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Dir(catalogFile), filepath.Join(sourceDir, "skills")} {
		if err := os.RemoveAll(path); err != nil {
			t.Fatal(err)
		}
	}
	writeOwnedPluginSkill(t, sourceDir, "a", "foo", "OWNER A v2")
	writeOwnedPluginSkill(t, sourceDir, "b", "a_foo", "OWNER B v2")
	if _, err := backend.SyncSources("plugins"); err != nil {
		t.Fatal(err)
	}
	updated, err := backend.UpdateInstalled(InstalledUpdateOptions{Target: TargetDirectory, Dir: custom})
	if err != nil || updated.Updated != 2 || updated.Skipped != 0 || updated.Failed != 0 {
		t.Fatalf("legacy update: %+v, %v", updated, err)
	}
	for id, dir := range owners {
		assertOwnedSkill(t, filepath.Join(custom, id), dir)
		meta, ok := readMetadata(filepath.Join(custom, id, ".skillhub.json"))
		if !ok || meta.Skill != id || meta.QualifiedSkill != "plugins/"+id {
			t.Fatalf("legacy metadata changed: %+v", meta)
		}
	}

	// Restore the old portable lockfile on another machine with no sources or
	// registry. The historical ID must still resolve to its original content.
	freshProject := t.TempDir()
	fresh, _, _ := newSourceRegressionBackend(t, freshProject)
	if err := os.WriteFile(filepath.Join(freshProject, "skills.lock.toml"), lock, 0o644); err != nil {
		t.Fatal(err)
	}
	restored, err := fresh.Restore(RestoreOptions{Project: freshProject})
	if err != nil || restored.Installed != 2 || restored.Skipped != 0 || restored.Failed != 0 {
		t.Fatalf("legacy restore: %+v, %v", restored, err)
	}
	for id, dir := range owners {
		assertOwnedSkill(t, filepath.Join(freshProject, ".agents", "skills", id), dir)
	}

	skills, _, err := backend.ListSkills("")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, skill := range skills {
		got[skill.Name] = true
	}
	if !reflect.DeepEqual(got, map[string]bool{"foo": true, "b_a_foo": true}) {
		t.Fatalf("discovery must use non-conflicting primary IDs: %+v", skills)
	}
	allDir := t.TempDir()
	if _, _, err := backend.Install(InstallOptions{All: true, Target: TargetDirectory, Dir: allDir}); err != nil {
		t.Fatal(err)
	}
	assertOwnedSkill(t, filepath.Join(allDir, "foo"), a)
	assertOwnedSkill(t, filepath.Join(allDir, "b_a_foo"), b)
	if _, err := os.Stat(filepath.Join(allDir, "a_foo")); !os.IsNotExist(err) {
		t.Fatalf("--all installed an extra legacy alias: %v", err)
	}
}

func TestPluginPrimaryIDDoesNotChangeWhenAnotherPluginIsAdded(t *testing.T) {
	backend, _, _ := newSourceRegressionBackend(t, t.TempDir())
	sourceDir := t.TempDir()
	b := writeOwnedPluginSkill(t, sourceDir, "b", "a_foo", "OWNER B")
	if _, err := backend.AddSource(SourceAddOptions{Name: "plugins", Type: SourceTypePath, Location: sourceDir}); err != nil {
		t.Fatal(err)
	}
	skills, _, err := backend.ListSkills("")
	if err != nil || len(skills) != 1 || skills[0].Name != "b_a_foo" {
		t.Fatalf("underscore namespace must be stable before collisions exist: %+v, %v", skills, err)
	}
	target := t.TempDir()
	if _, _, err := backend.Install(InstallOptions{All: true, Target: TargetDirectory, Dir: target}); err != nil {
		t.Fatal(err)
	}
	writeOwnedPluginSkill(t, sourceDir, "a", "foo", "OWNER A")
	updated, err := backend.UpdateInstalled(InstalledUpdateOptions{Target: TargetDirectory, Dir: target})
	if err != nil || updated.Updated+updated.Unchanged != 1 || updated.Skipped != 0 {
		t.Fatalf("update after source growth: %+v, %v", updated, err)
	}
	assertOwnedSkill(t, filepath.Join(target, "b_a_foo"), b)
}

func TestGeneratedSkillOwnershipDoesNotDependOnDiscoveryOrder(t *testing.T) {
	source := t.TempDir()
	a := filepath.Join(writeOwnedPluginSkill(t, source, "a", "foo", "A"), "SKILL.md")
	b := filepath.Join(writeOwnedPluginSkill(t, source, "b", "a_foo", "B"), "SKILL.md")
	for _, files := range [][]string{{a, b}, {b, a}} {
		plans, err := planGeneratedSkills("plugins", source, files)
		if err != nil {
			t.Fatal(err)
		}
		names := map[string]string{}
		for _, plan := range plans {
			names[plan.name] = plan.file
			if plan.legacyName != "" {
				names[plan.legacyName] = plan.file
			}
		}
		if !reflect.DeepEqual(names, map[string]string{"foo": a, "a_foo": a, "b_a_foo": b}) {
			t.Fatalf("incorrect ID ownership: %+v", names)
		}
	}
}

func TestOutdatedGeneratedCatalogIsRebuiltBeforeReading(t *testing.T) {
	backend, _, _ := newSourceRegressionBackend(t, t.TempDir())
	source := Source{Name: "plugins", Type: SourceTypeGit, Location: "https://example.invalid/plugins", Ref: "main", Catalog: "catalog/skills.tsv"}
	raw, err := backend.SourcePath(source)
	if err != nil {
		t.Fatal(err)
	}
	a := writeOwnedPluginSkill(t, raw, "a", "foo", "OWNER A")
	b := writeOwnedPluginSkill(t, raw, "b", "a_foo", "OWNER B")
	generated, err := backend.generatedSourcePath(source.Name)
	if err != nil {
		t.Fatal(err)
	}
	// Seed a cache from the earlier, unsafe canonical-first naming policy.
	if err := copyDir(b, filepath.Join(generated, "skills", "a_foo")); err != nil {
		t.Fatal(err)
	}
	if err := CatalogTable.WriteFile(filepath.Join(generated, source.Catalog), []CatalogRow{
		{Name: "a_foo", Category: "b", Triggers: "b,a_foo", Description: "Wrong owner for historical ID"},
	}); err != nil {
		t.Fatal(err)
	}
	materialized, _, err := backend.CatalogSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if !generatedCatalogIsCurrent(materialized) {
		t.Fatal("outdated cache was not rebuilt")
	}
	assertOwnedSkill(t, filepath.Join(materialized, "skills", "a_foo"), a)
	assertOwnedSkill(t, filepath.Join(materialized, "skills", "b_a_foo"), b)
}
