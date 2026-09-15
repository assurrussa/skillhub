package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeCatalogCanUseCompatibilityCategory(t *testing.T) {
	projectDir := t.TempDir()
	backend, configDir, _ := newSourceRegressionBackend(t, projectDir)
	sourceDir := filepath.Join(projectDir, "native-source")
	skillDir := filepath.Join(sourceDir, "skills", "native-compat")
	if err := os.MkdirAll(filepath.Join(sourceDir, "catalog"), 0o755); err != nil {
		t.Fatalf("mkdir catalog: %v", err)
	}
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	catalog := "name\tcategory\ttriggers\tdescription\n" +
		"native-compat\t" + generatedSkillCompatibilityCategory + "\tnative\tNative compatibility category\n"
	if err := os.WriteFile(filepath.Join(sourceDir, "catalog", "skills.tsv"), []byte(catalog), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	skill := "---\nname: native-compat\ndescription: Native compatibility category\n---\n\n# Native compat\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skill), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	writeRegressionSource(t, configDir, "native", sourceDir)

	skills, warning, err := backend.ListSkills("")
	if err != nil {
		t.Fatalf("list native skills: %v", err)
	}
	if warning != "" {
		t.Fatalf("unexpected warning: %s", warning)
	}
	if len(skills) != 1 || skills[0].Name != "native-compat" || skills[0].Category != generatedSkillCompatibilityCategory {
		t.Fatalf("native compatibility category must remain visible: %#v", skills)
	}

	matched, _, err := backend.ListSkills(generatedSkillCompatibilityCategory)
	if err != nil {
		t.Fatalf("search native compatibility category: %v", err)
	}
	if len(matched) != 1 || matched[0].Name != "native-compat" {
		t.Fatalf("native compatibility category must remain searchable: %#v", matched)
	}

	targetDir := filepath.Join(projectDir, "install-all")
	if _, _, err := backend.Install(InstallOptions{All: true, Target: TargetDirectory, Dir: targetDir}); err != nil {
		t.Fatalf("install --all must include native compatibility category: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "native-compat", "SKILL.md")); err != nil {
		t.Fatalf("native compatibility category skill was not installed by --all: %v", err)
	}
}
