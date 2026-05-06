package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func testRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func runScriptForTest(t *testing.T, env []string, script string, args ...string) (string, error) {
	t.Helper()
	root := testRepoRoot(t)
	cmdArgs := append([]string{filepath.Join(root, script)}, args...)
	cmd := exec.Command("sh", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func writeTestSourceCatalog(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "catalog"), 0o755); err != nil {
		t.Fatalf("mkdir source catalog: %v", err)
	}
	catalog := strings.Join([]string{
		"name\tcategory\ttriggers\tdescription",
		"reusable-module-rules\tarchitecture\treusable module,library,public surface,external consumer,release readiness,replace\tReusable rules",
		"docs-project-rules\tdocumentation\tdocs,documentation,readme,architecture,runbook,audit,report\tDocs rules",
		"go-project-rules\tgo\tgo,golang,go.mod,go.work,backend,library\tGo rules",
		"rules-selector\ttooling\tselect rules,install skills,recommend skills,project analysis,catalog,skillhub\tSelector",
		"project-workflow-rules\tworkflow\tscope,review,dirty worktree,contract change,verification,repo orientation\tWorkflow rules",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "catalog", "skills.tsv"), []byte(catalog), 0o644); err != nil {
		t.Fatalf("write source catalog: %v", err)
	}
}

func writeTestSources(t *testing.T, configDir, sourceDir string) {
	t.Helper()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	sources := "name\ttype\tlocation\tref\tcatalog\n" +
		"local\tpath\t" + sourceDir + "\t-\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(sources), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
}

func TestRecommendRanksSkillsFromProjectSignals(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeTestSourceCatalog(t, sourceDir)
	writeTestSources(t, configDir, sourceDir)

	for _, dir := range []string{projectDir, filepath.Join(projectDir, "docs"), filepath.Join(projectDir, "pkg")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	files := map[string]string{
		"AGENTS.md":           "Use local project workflow rules and verification before completion.\n",
		"README.md":           "# Go service\n\nDocuments OpenAPI contracts and architecture.\n",
		"go.mod":              "module example.com/project\n\ngo 1.26.0\n",
		"go.work":             "go 1.26.0\n\nuse .\n",
		"docs/openapi.yaml":   "openapi: 3.0.0\ninfo:\n  title: API\n  version: v1\n",
		"pkg/public.go":       "package pkg\n",
		"cmd/service/main.go": "package main\nfunc main() {}\n",
	}
	for name, content := range files {
		path := filepath.Join(projectDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	output, err := runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/recommend.sh", "--project", projectDir, "--tsv")
	if err != nil {
		t.Fatalf("recommend failed: %v\n%s", err, output)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 5 {
		t.Fatalf("expected ranked recommendations, got:\n%s", output)
	}
	if lines[0] != "source\tskill\treason\tinstall_arg" {
		t.Fatalf("recommend TSV header changed: %q", lines[0])
	}
	gotSkills := make([]string, 0, len(lines)-1)
	gotReasons := map[string]string{}
	for _, line := range lines[1:] {
		parts := strings.Split(line, "\t")
		if len(parts) != 4 {
			t.Fatalf("bad recommendation row %q", line)
		}
		gotSkills = append(gotSkills, parts[1])
		gotReasons[parts[1]] = parts[2]
	}
	wantOrder := []string{"go-project-rules", "docs-project-rules", "reusable-module-rules", "project-workflow-rules"}
	for i, want := range wantOrder {
		if i >= len(gotSkills) || gotSkills[i] != want {
			t.Fatalf("expected ranked skill %d to be %s, got order %#v\n%s", i, want, gotSkills, output)
		}
	}
	for skill, reasonPart := range map[string]string{
		"go-project-rules":       "go.mod",
		"docs-project-rules":     "OpenAPI",
		"reusable-module-rules":  "pkg/",
		"project-workflow-rules": "AGENTS.md",
	} {
		if !strings.Contains(gotReasons[skill], reasonPart) {
			t.Fatalf("expected %s reason to mention %q, got %q", skill, reasonPart, gotReasons[skill])
		}
	}
}

func TestRecommendEmptyProjectReturnsNoRecommendations(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeTestSourceCatalog(t, sourceDir)
	writeTestSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	output, err := runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/recommend.sh", "--project", projectDir)
	if err != nil {
		t.Fatalf("recommend failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "No shared skills recommended") {
		t.Fatalf("expected no recommendations for empty project, got:\n%s", output)
	}
}

func TestUsageUpdateProjectsFiltersByTargetAndProject(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	projectA := filepath.Join(tmp, "project-a")
	projectB := filepath.Join(tmp, "project-b")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	registry := strings.Join([]string{
		"source\tskill\ttarget\tscope\tproject_path\ttarget_root\tinstalled_path\tsource_ref\tsource_location\tcatalog\tcontent_hash\tinstalled_at\tupdated_at",
		"local\trules-selector\tcodex\tproject\t" + projectA + "\t" + filepath.Join(projectA, ".agents", "skills") + "\t" + filepath.Join(projectA, ".agents", "skills", "rules-selector") + "\t-\t" + tmp + "\tcatalog/skills.tsv\told\t2026-05-05T00:00:00Z\t2026-05-05T00:00:00Z",
		"local\trules-selector\tclaude\tproject\t" + projectB + "\t" + filepath.Join(projectB, ".claude", "skills") + "\t" + filepath.Join(projectB, ".claude", "skills", "rules-selector") + "\t-\t" + tmp + "\tcatalog/skills.tsv\told\t2026-05-05T00:00:00Z\t2026-05-05T00:00:00Z",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(configDir, "installed.tsv"), []byte(registry), 0o644); err != nil {
		t.Fatalf("write installed registry: %v", err)
	}

	output, err := runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/installed.sh", "usage", "update", "--projects", "--target", "codex", "--project", projectA, "rules-selector", "-v")
	if err != nil {
		t.Fatalf("usage update failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "Skipped codex/project rules-selector") {
		t.Fatalf("expected codex/project row to be processed, got:\n%s", output)
	}
	if strings.Contains(output, "claude/project") {
		t.Fatalf("expected claude/project row to be filtered out, got:\n%s", output)
	}
}
