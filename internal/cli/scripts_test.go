package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
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

func runScriptSplitForTest(t *testing.T, env []string, script string, args ...string) (string, string, error) {
	t.Helper()
	root := testRepoRoot(t)
	cmdArgs := append([]string{filepath.Join(root, script)}, args...)
	cmd := exec.Command("sh", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
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

func writeInstallableTestSource(t *testing.T, dir string, skillName string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "catalog"), 0o755); err != nil {
		t.Fatalf("mkdir installable source catalog: %v", err)
	}
	catalog := "name\tcategory\ttriggers\tdescription\n" +
		skillName + "\ttooling\t" + skillName + "\tInstallable test skill\n"
	if err := os.WriteFile(filepath.Join(dir, "catalog", "skills.tsv"), []byte(catalog), 0o644); err != nil {
		t.Fatalf("write installable source catalog: %v", err)
	}
	skillDir := filepath.Join(dir, "skills", skillName)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir installable source skill: %v", err)
	}
	content := "---\nname: " + skillName + "\ndescription: Installable test skill\n---\n\n# " + skillName + "\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write installable source skill: %v", err)
	}
}

func writeTestSourceCatalogWithExtraSkill(t *testing.T, dir string, skill string) {
	t.Helper()
	writeTestSourceCatalog(t, dir)
	if skill == "" {
		return
	}
	path := filepath.Join(dir, "catalog", "skills.tsv")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open source catalog: %v", err)
	}
	defer file.Close()
	if _, err := file.WriteString(skill + "\ttooling\t" + skill + "\tExtra skill\n"); err != nil {
		t.Fatalf("append source catalog: %v", err)
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

func writeTestGitSources(t *testing.T, configDir, name, location string) {
	t.Helper()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	sources := "name\ttype\tlocation\tref\tcatalog\n" +
		name + "\tgit\t" + location + "\tmain\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(sources), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
}

func writeSourceSyncedAt(t *testing.T, cacheDir, name string, at time.Time) {
	t.Helper()
	stateDir := filepath.Join(cacheDir, "source-state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir source state: %v", err)
	}
	value := []byte(strconv.FormatInt(at.Unix(), 10) + "\n")
	if err := os.WriteFile(filepath.Join(stateDir, name+".synced_at"), value, 0o644); err != nil {
		t.Fatalf("write source sync state: %v", err)
	}
}

func readSourceSyncedAt(t *testing.T, cacheDir, name string) int64 {
	t.Helper()
	value, err := os.ReadFile(filepath.Join(cacheDir, "source-state", name+".synced_at"))
	if err != nil {
		t.Fatalf("read source sync state: %v", err)
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(string(value)), 10, 64)
	if err != nil {
		t.Fatalf("parse source sync state %q: %v", value, err)
	}
	return parsed
}

func writeCachedGitCatalog(t *testing.T, cacheDir, name, extraSkill string) string {
	t.Helper()
	sourceDir := filepath.Join(cacheDir, "sources", name)
	writeTestSourceCatalogWithExtraSkill(t, sourceDir, extraSkill)
	return sourceDir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func gitIgnored(t *testing.T, dir string, path string) bool {
	t.Helper()
	cmd := exec.Command("git", "check-ignore", "-q", path)
	cmd.Dir = dir
	err := cmd.Run()
	return err == nil
}

func commitGitSource(t *testing.T, dir, message string) {
	t.Helper()
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Skillhub Test", "-c", "user.email=skillhub@example.test", "commit", "-m", message)
}

func initGitSource(t *testing.T, dir string, extraSkill string) {
	t.Helper()
	writeTestSourceCatalogWithExtraSkill(t, dir, extraSkill)
	runGit(t, dir, "init")
	runGit(t, dir, "checkout", "-b", "main")
	commitGitSource(t, dir, "initial catalog")
}

func writeNestedSkill(t *testing.T, sourceDir, rel, description, body string) {
	t.Helper()
	skillDir := filepath.Join(sourceDir, "skills", filepath.FromSlash(rel))
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir nested skill: %v", err)
	}
	name := filepath.Base(filepath.FromSlash(rel))
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n# " + name + "\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write nested skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "notes.md"), []byte("notes for "+rel+"\n"), 0o644); err != nil {
		t.Fatalf("write nested skill notes: %v", err)
	}
}

func initNestedGitSource(t *testing.T, dir string) {
	t.Helper()
	writeNestedSkill(t, dir, "engineering/tdd", "Test-driven development with tracer bullets", "TDD v1")
	writeNestedSkill(t, dir, "productivity/grill-me", "Interview the user relentlessly", "Grill v1")
	runGit(t, dir, "init")
	runGit(t, dir, "checkout", "-b", "main")
	commitGitSource(t, dir, "initial nested skills")
}

func writeRootSkill(t *testing.T, sourceDir, name, description, body string) {
	t.Helper()
	skillDir := filepath.Join(sourceDir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir root skill: %v", err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n# " + name + "\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write root skill: %v", err)
	}
}

func initRootGitSource(t *testing.T, dir string) {
	t.Helper()
	writeRootSkill(t, dir, "tdd", "Test-driven development with red-green-refactor loop", "TDD root")
	runGit(t, dir, "init")
	runGit(t, dir, "checkout", "-b", "main")
	commitGitSource(t, dir, "initial root skills")
}

func TestSkillsListUsesFreshCachedGitCatalogWithoutSync(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	writeTestGitSources(t, configDir, "cached", filepath.Join(tmp, "missing-remote"))
	writeCachedGitCatalog(t, cacheDir, "cached", "cached-only-rules")
	writeSourceSyncedAt(t, cacheDir, "cached", time.Now())

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "cached\tcached-only-rules\t") {
		t.Fatalf("expected cached catalog row, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("fresh cache should not emit refresh warnings, got stderr:\n%s", stderr)
	}
}

func TestSkillsListRefreshesExpiredGitCatalogAndUpdatesTimestamp(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	initGitSource(t, sourceDir, "")
	writeTestGitSources(t, configDir, "cached", sourceDir)

	if err := os.MkdirAll(filepath.Join(cacheDir, "sources"), 0o755); err != nil {
		t.Fatalf("mkdir cache sources: %v", err)
	}
	runGit(t, tmp, "clone", sourceDir, filepath.Join(cacheDir, "sources", "cached"))
	writeSourceSyncedAt(t, cacheDir, "cached", time.Now().Add(-11*time.Minute))
	oldStamp := readSourceSyncedAt(t, cacheDir, "cached")

	writeTestSourceCatalogWithExtraSkill(t, sourceDir, "new-rules")
	commitGitSource(t, sourceDir, "add new rules")

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "cached\tnew-rules\t") {
		t.Fatalf("expected refreshed catalog row, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if got := readSourceSyncedAt(t, cacheDir, "cached"); got <= oldStamp {
		t.Fatalf("expected sync timestamp to be updated from %d, got %d", oldStamp, got)
	}
}

func TestSkillsListUsesStaleCacheWithWarningWhenExpiredRefreshFails(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	writeTestGitSources(t, configDir, "cached", filepath.Join(tmp, "missing-remote"))
	writeCachedGitCatalog(t, cacheDir, "cached", "stale-rules")
	writeSourceSyncedAt(t, cacheDir, "cached", time.Now().Add(-11*time.Minute))

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list should use stale cache: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "cached\tstale-rules\t") {
		t.Fatalf("expected stale catalog row, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "Warning: using stale cache for source cached") {
		t.Fatalf("expected stale cache warning, got stderr:\n%s", stderr)
	}
}

func TestSkillsListUsesStaleGeneratedCacheWithWarningWhenRefreshFails(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "nested-source")
	initNestedGitSource(t, sourceDir)
	writeTestGitSources(t, configDir, "nested", sourceDir)

	if err := os.MkdirAll(filepath.Join(cacheDir, "sources"), 0o755); err != nil {
		t.Fatalf("mkdir cache sources: %v", err)
	}
	runGit(t, tmp, "clone", sourceDir, filepath.Join(cacheDir, "sources", "nested"))
	writeSourceSyncedAt(t, cacheDir, "nested", time.Now().Add(-11*time.Minute))
	if err := os.RemoveAll(sourceDir); err != nil {
		t.Fatalf("remove source remote: %v", err)
	}

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list should use stale generated cache: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "nested\tengineering_tdd\t") {
		t.Fatalf("expected stale generated catalog row, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "Warning: using stale cache for source nested") {
		t.Fatalf("expected stale cache warning, got stderr:\n%s", stderr)
	}
}

func TestSkillsListMissingGitCacheFailsWithSyncHint(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	writeTestGitSources(t, configDir, "cached", filepath.Join(tmp, "missing-remote"))

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err == nil {
		t.Fatalf("expected missing cache to fail, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "Run: skillhub sources sync cached") {
		t.Fatalf("expected sources sync hint, got stderr:\n%s", stderr)
	}
}

func TestSourcesSyncForcesRefreshAndWritesTimestamp(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	initGitSource(t, sourceDir, "")
	writeTestGitSources(t, configDir, "cached", sourceDir)

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "sync", "cached")
	if err != nil {
		t.Fatalf("sources sync failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	firstStamp := readSourceSyncedAt(t, cacheDir, "cached")

	writeTestSourceCatalogWithExtraSkill(t, sourceDir, "synced-rules")
	commitGitSource(t, sourceDir, "add synced rules")

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "sync", "cached")
	if err != nil {
		t.Fatalf("second sources sync failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if got := readSourceSyncedAt(t, cacheDir, "cached"); got < firstStamp {
		t.Fatalf("expected sync timestamp to advance from %d, got %d", firstStamp, got)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list failed after sync: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "cached\tsynced-rules\t") {
		t.Fatalf("expected sync to refresh catalog, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestSourcesAddGitPerformsInitialSync(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	initGitSource(t, sourceDir, "added-rules")

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "add", sourceDir, "--name", "added", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Added source added") {
		t.Fatalf("expected add output, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	_ = readSourceSyncedAt(t, cacheDir, "added")

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list failed after add: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "added\tadded-rules\t") {
		t.Fatalf("expected initially synced catalog, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestSourcesAddGitReusesNameWithDifferentLocationRefreshesCache(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceA := filepath.Join(tmp, "source-a")
	sourceB := filepath.Join(tmp, "source-b")
	initGitSource(t, sourceA, "old-rules")
	initGitSource(t, sourceB, "new-rules")

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "add", sourceA, "--name", "shared", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("first sources add failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "remove", "shared")
	if err != nil {
		t.Fatalf("sources remove failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "add", sourceB, "--name", "shared", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("second sources add failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list failed after re-add: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "shared\tnew-rules\t") {
		t.Fatalf("expected re-added source catalog from new location, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if strings.Contains(stdout, "shared\told-rules\t") {
		t.Fatalf("expected old source cache not to leak after re-add, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestSourcesRemoveClearsGitCacheAndSyncState(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	initGitSource(t, sourceDir, "remove-rules")

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "add", sourceDir, "--name", "removable", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "sources", "removable", "catalog", "skills.tsv")); err != nil {
		t.Fatalf("expected source cache after add: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "source-state", "removable.synced_at")); err != nil {
		t.Fatalf("expected source sync state after add: %v", err)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "remove", "removable")
	if err != nil {
		t.Fatalf("sources remove failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "sources", "removable")); !os.IsNotExist(err) {
		t.Fatalf("expected source cache to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "source-state", "removable.synced_at")); !os.IsNotExist(err) {
		t.Fatalf("expected source sync state to be removed, stat err=%v", err)
	}
}

func TestSkillsListIgnoresFreshTimestampWhenCachedOriginDiffers(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceA := filepath.Join(tmp, "source-a")
	sourceB := filepath.Join(tmp, "source-b")
	initGitSource(t, sourceA, "old-rules")
	initGitSource(t, sourceB, "new-rules")
	writeTestGitSources(t, configDir, "shared", sourceB)

	if err := os.MkdirAll(filepath.Join(cacheDir, "sources"), 0o755); err != nil {
		t.Fatalf("mkdir cache sources: %v", err)
	}
	runGit(t, tmp, "clone", sourceA, filepath.Join(cacheDir, "sources", "shared"))
	writeSourceSyncedAt(t, cacheDir, "shared", time.Now())

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "shared\tnew-rules\t") {
		t.Fatalf("expected catalog from configured source location, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if strings.Contains(stdout, "shared\told-rules\t") {
		t.Fatalf("expected old origin cache not to be used, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestRecommendUsesFreshCachedGitCatalogWithoutSync(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	projectDir := filepath.Join(tmp, "project")
	writeTestGitSources(t, configDir, "cached", filepath.Join(tmp, "missing-remote"))
	writeCachedGitCatalog(t, cacheDir, "cached", "")
	writeSourceSyncedAt(t, cacheDir, "cached", time.Now())
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/project\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/recommend.sh", "--project", projectDir, "--tsv")
	if err != nil {
		t.Fatalf("recommend failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "cached\tgo-project-rules\t") {
		t.Fatalf("expected cached recommendation, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestNestedGitSourceCanBeAddedListedSearchedAndInstalled(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "nested-source")
	targetDir := filepath.Join(tmp, "target")
	initNestedGitSource(t, sourceDir)

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "add", sourceDir, "--name", "nested", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "nested\tengineering_tdd\tengineering\tengineering,tdd\tTest-driven development with tracer bullets") {
		t.Fatalf("expected generated engineering_tdd row, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stdout, "nested\tproductivity_grill-me\tproductivity\tproductivity,grill-me\tInterview the user relentlessly") {
		t.Fatalf("expected generated productivity_grill-me row, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "search", "--tsv", "tracer")
	if err != nil {
		t.Fatalf("skills search nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "nested\tengineering_tdd\t") {
		t.Fatalf("expected search to find generated nested skill, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "install", "nested/engineering_tdd", "--target", "directory", "--dir", targetDir)
	if err != nil {
		t.Fatalf("install generated nested skill failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	installedDir := filepath.Join(targetDir, "engineering_tdd")
	for _, path := range []string{
		filepath.Join(installedDir, "SKILL.md"),
		filepath.Join(installedDir, "notes.md"),
		filepath.Join(installedDir, ".skillhub.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected installed generated skill file %s: %v", path, err)
		}
	}
}

func TestRootSkillDirectoriesCanBeAddedAndListed(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "root-source")
	initRootGitSource(t, sourceDir)

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "add", sourceDir, "--name", "rooted", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add root skills failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "search", "--tsv", "red-green")
	if err != nil {
		t.Fatalf("skills search root skills failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "rooted\ttdd\ttdd\ttdd\tTest-driven development with red-green-refactor loop") {
		t.Fatalf("expected generated root tdd row, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestNestedInstalledUpdateUsesRegeneratedSource(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "nested-source")
	targetDir := filepath.Join(tmp, "target")
	initNestedGitSource(t, sourceDir)

	for _, args := range [][]string{
		{"scripts/sources.sh", "add", sourceDir, "--name", "nested", "--type", "git", "--ref", "main"},
		{"scripts/skills.sh", "install", "nested/engineering_tdd", "--target", "directory", "--dir", targetDir},
	} {
		stdout, stderr, err := runScriptSplitForTest(t, []string{
			"SKILLHUB_CONFIG_DIR=" + configDir,
			"SKILLHUB_CACHE_DIR=" + cacheDir,
		}, args[0], args[1:]...)
		if err != nil {
			t.Fatalf("%s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout, stderr)
		}
	}

	writeNestedSkill(t, sourceDir, "engineering/tdd", "Test-driven development with tracer bullets", "TDD v2")
	commitGitSource(t, sourceDir, "update tdd")

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/installed.sh", "update", "--target", "directory", "--dir", targetDir, "-v")
	if err != nil {
		t.Fatalf("installed update generated nested skill failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	content, err := os.ReadFile(filepath.Join(targetDir, "engineering_tdd", "SKILL.md"))
	if err != nil {
		t.Fatalf("read updated generated skill: %v", err)
	}
	if !strings.Contains(string(content), "TDD v2") {
		t.Fatalf("expected installed skill to update from regenerated nested source, got:\n%s", content)
	}
}

func TestNestedSourceSyncRebuildsGeneratedCatalog(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "nested-source")
	initNestedGitSource(t, sourceDir)

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "add", sourceDir, "--name", "nested", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	writeNestedSkill(t, sourceDir, "engineering/diagnose", "Disciplined diagnosis loop", "Diagnose v1")
	commitGitSource(t, sourceDir, "add diagnose")

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "sync", "nested")
	if err != nil {
		t.Fatalf("sources sync nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/skills.sh", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list nested after sync failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "nested\tengineering_diagnose\t") {
		t.Fatalf("expected regenerated catalog to include new nested skill, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestNestedSourceRemoveClearsGeneratedCache(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "nested-source")
	initNestedGitSource(t, sourceDir)

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "add", sourceDir, "--name", "nested", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "generated-sources", "nested", "catalog", "skills.tsv")); err != nil {
		t.Fatalf("expected generated catalog after add: %v", err)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "remove", "nested")
	if err != nil {
		t.Fatalf("sources remove nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "generated-sources", "nested")); !os.IsNotExist(err) {
		t.Fatalf("expected generated source cache to be removed, stat err=%v", err)
	}
}

func TestNestedSourceDuplicateFlattenedNamesFailClearly(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "nested-source")
	writeNestedSkill(t, sourceDir, "a_b/c", "First duplicate", "first")
	writeNestedSkill(t, sourceDir, "a/b_c", "Second duplicate", "second")

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "add", sourceDir, "--name", "dups", "--type", "path")
	if err == nil {
		t.Fatalf("expected duplicate generated skill names to fail, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "Duplicate generated skill name a_b_c") {
		t.Fatalf("expected duplicate generated skill error, got stderr:\n%s", stderr)
	}
}

func TestRecommendReadsGeneratedNestedCatalog(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "nested-source")
	projectDir := filepath.Join(tmp, "project")
	writeNestedSkill(t, sourceDir, "go/go-mod", "Go module guidance", "Go module body")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/project\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	stdout, stderr, err := runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/sources.sh", "add", sourceDir, "--name", "nested", "--type", "path")
	if err != nil {
		t.Fatalf("sources add nested path failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runScriptSplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "scripts/recommend.sh", "--project", projectDir, "--tsv")
	if err != nil {
		t.Fatalf("recommend generated nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "nested\tgo_go-mod\t") {
		t.Fatalf("expected recommend to include generated nested skill, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
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

func TestProjectInstallUsesCentralMetadataAndGitignore(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeInstallableTestSource(t, sourceDir, "rules-selector", "v1")
	writeTestSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	runGit(t, projectDir, "init")
	if err := os.WriteFile(filepath.Join(projectDir, ".gitignore"), []byte("existing.log\n"), 0o644); err != nil {
		t.Fatalf("write project gitignore: %v", err)
	}

	output, err := runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/skills.sh", "install", "rules-selector", "--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}

	installedDir := filepath.Join(projectDir, ".agents", "skills", "rules-selector")
	if _, err := os.Stat(filepath.Join(installedDir, "SKILL.md")); err != nil {
		t.Fatalf("expected project skill content to install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installedDir, ".skillhub.json")); !os.IsNotExist(err) {
		t.Fatalf("project install should not write sidecar metadata, stat err=%v", err)
	}
	registry, err := os.ReadFile(filepath.Join(configDir, "installed.tsv"))
	if err != nil {
		t.Fatalf("read installed registry: %v", err)
	}
	if !strings.Contains(string(registry), "local\trules-selector\tcodex\tproject\t"+projectDir) {
		t.Fatalf("expected central installed registry project row, got:\n%s", registry)
	}

	gitignore, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	if err != nil {
		t.Fatalf("read project gitignore: %v", err)
	}
	for _, want := range []string{
		"existing.log",
		"# >>> skillhub local metadata >>>",
		"skills/**/.skillhub.json",
		".*/skills/**/.skillhub.json",
		"# <<< skillhub local metadata <<<",
	} {
		if !strings.Contains(string(gitignore), want) {
			t.Fatalf("expected .gitignore to contain %q, got:\n%s", want, gitignore)
		}
	}
	if !gitIgnored(t, projectDir, ".agents/skills/rules-selector/.skillhub.json") {
		t.Fatalf("expected project gitignore to ignore skillhub sidecar metadata")
	}
	if gitIgnored(t, projectDir, ".agents/skills/rules-selector/SKILL.md") {
		t.Fatalf("project skill content must remain trackable")
	}
}

func TestGlobalAndDirectoryInstallsKeepSidecarMetadata(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	globalDir := filepath.Join(tmp, "global")
	directoryDir := filepath.Join(tmp, "directory")
	writeInstallableTestSource(t, sourceDir, "rules-selector", "v1")
	writeTestSources(t, configDir, sourceDir)

	output, err := runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir, "AGENT_SKILLS_DIR=" + globalDir}, "scripts/skills.sh", "install", "rules-selector")
	if err != nil {
		t.Fatalf("global install failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(globalDir, "rules-selector", ".skillhub.json")); err != nil {
		t.Fatalf("expected global install sidecar metadata: %v", err)
	}

	output, err = runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/skills.sh", "install", "rules-selector", "--target", "directory", "--dir", directoryDir)
	if err != nil {
		t.Fatalf("directory install failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(directoryDir, "rules-selector", ".skillhub.json")); err != nil {
		t.Fatalf("expected directory install sidecar metadata: %v", err)
	}
}

func TestProjectUsageUpdateWorksWithoutSidecarAndRemovesStaleSidecar(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeInstallableTestSource(t, sourceDir, "rules-selector", "v1")
	writeTestSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	runGit(t, projectDir, "init")

	output, err := runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/skills.sh", "install", "rules-selector", "--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}
	installedDir := filepath.Join(projectDir, ".agents", "skills", "rules-selector")
	staleMetadata := filepath.Join(installedDir, ".skillhub.json")
	if err := os.WriteFile(staleMetadata, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write stale sidecar metadata: %v", err)
	}

	writeInstallableTestSource(t, sourceDir, "rules-selector", "v2")
	output, err = runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/installed.sh", "usage", "update", "--projects", "--target", "codex", "--project", projectDir, "rules-selector", "-v")
	if err != nil {
		t.Fatalf("project usage update failed: %v\n%s", err, output)
	}
	content, err := os.ReadFile(filepath.Join(installedDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read updated project skill: %v", err)
	}
	if !strings.Contains(string(content), "v2") {
		t.Fatalf("expected project skill to update from central registry without sidecar, got:\n%s", content)
	}
	if _, err := os.Stat(staleMetadata); !os.IsNotExist(err) {
		t.Fatalf("project update should remove stale sidecar metadata, stat err=%v", err)
	}
}

func TestInstalledListMarksRegistryBackedProjectSkillManagedWithoutSidecar(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeInstallableTestSource(t, sourceDir, "rules-selector", "v1")
	writeTestSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	runGit(t, projectDir, "init")

	output, err := runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/skills.sh", "install", "rules-selector", "--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}

	output, err = runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/installed.sh", "list", "--target", "codex", "--scope", "project", "--project", projectDir, "--tsv")
	if err != nil {
		t.Fatalf("installed list failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "codex\tproject\trules-selector\tyes\tlocal\tlocal/rules-selector") {
		t.Fatalf("expected registry-backed project skill to be managed in installed list, got:\n%s", output)
	}
}

func TestTargetsDetectCountsRegistryBackedProjectSkillManagedWithoutSidecar(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeInstallableTestSource(t, sourceDir, "rules-selector", "v1")
	writeTestSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	runGit(t, projectDir, "init")

	output, err := runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/skills.sh", "install", "rules-selector", "--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}

	output, err = runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/targets.sh", "detect", "--tsv", "--project", projectDir)
	if err != nil {
		t.Fatalf("targets detect failed: %v\n%s", err, output)
	}
	want := "codex\tproject\tsupported\t" + filepath.Join(projectDir, ".agents", "skills") + "\tyes\t1\t1"
	if !strings.Contains(output, want) {
		t.Fatalf("expected targets detect to count registry-backed project managed skill %q, got:\n%s", want, output)
	}
}

func TestProjectSidecarWithoutRegistryIsUnmanagedForListDetectAndUninstall(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	projectDir := filepath.Join(tmp, "project")
	installedDir := filepath.Join(projectDir, ".agents", "skills", "rules-selector")
	if err := os.MkdirAll(installedDir, 0o755); err != nil {
		t.Fatalf("mkdir installed skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installedDir, "SKILL.md"), []byte("# rules-selector\n"), 0o644); err != nil {
		t.Fatalf("write project skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installedDir, ".skillhub.json"), []byte(`{"source":"local","skill":"rules-selector"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write stale sidecar: %v", err)
	}

	output, err := runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/installed.sh", "list", "--target", "codex", "--scope", "project", "--project", projectDir, "--tsv")
	if err != nil {
		t.Fatalf("installed list failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "codex\tproject\trules-selector\tno\t-\t-") {
		t.Fatalf("expected stale project sidecar without registry to be unmanaged, got:\n%s", output)
	}

	output, err = runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/targets.sh", "detect", "--tsv", "--project", projectDir)
	if err != nil {
		t.Fatalf("targets detect failed: %v\n%s", err, output)
	}
	want := "codex\tproject\tsupported\t" + filepath.Join(projectDir, ".agents", "skills") + "\tyes\t1\t0"
	if !strings.Contains(output, want) {
		t.Fatalf("expected targets detect to treat stale project sidecar as unmanaged %q, got:\n%s", want, output)
	}

	output, err = runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/installed.sh", "uninstall", "rules-selector", "--target", "codex", "--scope", "project", "--project", projectDir)
	if err == nil {
		t.Fatalf("expected uninstall to refuse stale project sidecar without registry, got:\n%s", output)
	}
	if !strings.Contains(output, "Refusing to uninstall unmanaged skill") {
		t.Fatalf("expected unmanaged uninstall refusal, got:\n%s", output)
	}
}

func TestProjectUpdateIgnoresSidecarWithoutRegistry(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	installedDir := filepath.Join(projectDir, ".agents", "skills", "rules-selector")
	writeInstallableTestSource(t, sourceDir, "rules-selector", "source v2")
	writeTestSources(t, configDir, sourceDir)
	if err := os.MkdirAll(installedDir, 0o755); err != nil {
		t.Fatalf("mkdir installed skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installedDir, "SKILL.md"), []byte("# rules-selector\n\nlocal v1\n"), 0o644); err != nil {
		t.Fatalf("write project skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installedDir, ".skillhub.json"), []byte(`{"source":"local","skill":"rules-selector"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write stale sidecar: %v", err)
	}

	output, err := runScriptForTest(t, []string{"SKILLHUB_CONFIG_DIR=" + configDir}, "scripts/installed.sh", "update", "--target", "codex", "--scope", "project", "--project", projectDir, "-v")
	if err != nil {
		t.Fatalf("project update failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "No managed installed skills found") {
		t.Fatalf("expected project update to ignore stale sidecar without registry, got:\n%s", output)
	}
	content, err := os.ReadFile(filepath.Join(installedDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read project skill: %v", err)
	}
	if strings.Contains(string(content), "source v2") {
		t.Fatalf("project update should not update stale sidecar without registry, got:\n%s", content)
	}
	if _, err := os.Stat(filepath.Join(configDir, "installed.tsv")); !os.IsNotExist(err) {
		t.Fatalf("project update should not create registry from stale sidecar, stat err=%v", err)
	}
}
