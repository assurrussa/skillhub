package core_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/assurrussa/skillhub/internal/core"
)

const (
	ruleSelector = "rules-selector"
	testEnvHome  = "HOME"
)

func writeCoreRepo(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "defaults"), 0o755); err != nil {
		t.Fatalf("mkdir defaults: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "targets"), 0o755); err != nil {
		t.Fatalf("mkdir targets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "defaults", "sources.tsv"), []byte(core.SourcesHeader+"\n"), 0o644); err != nil {
		t.Fatalf("write defaults: %v", err)
	}
	targets := core.TargetsHeader + "\n" +
		"codex\tCodex\tsupported\tskill-dir\tCodex skills\n" +
		"directory\tDirectory\tsupported\tskill-dir\tDirectory skills\n" +
		"claude\tClaude\tsupported\tskill-dir\tClaude skills\n"
	if err := os.WriteFile(filepath.Join(root, "targets", "targets.tsv"), []byte(targets), 0o644); err != nil {
		t.Fatalf("write targets: %v", err)
	}
}

func writeCoreSkill(t *testing.T, sourceDir, rel, description, body string) {
	t.Helper()
	skillDir := filepath.Join(sourceDir, "skills", filepath.FromSlash(rel))
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	name := filepath.Base(filepath.FromSlash(rel))
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n# " + name + "\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "notes.md"), []byte("notes\n"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}
}

func writeCoreCatalog(t *testing.T, sourceDir string, skills ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(sourceDir, "catalog"), 0o755); err != nil {
		t.Fatalf("mkdir catalog: %v", err)
	}
	var b strings.Builder
	_, _ = b.WriteString("name\tcategory\ttriggers\tdescription\n")
	for _, skill := range skills {
		_, _ = b.WriteString(skill + "\tgo\tgo,golang,go.mod\t" + skill + " desc\n")
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "catalog", "skills.tsv"), []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	for _, skill := range skills {
		writeCoreSkill(t, sourceDir, skill, skill+" desc", "body")
	}
}

func writeCoreSources(t *testing.T, configDir, sourceDir string) {
	t.Helper()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	sources := core.SourcesHeader + "\nlocal\tpath\t" + sourceDir + "\t-\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(sources), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
}

func writeCoreSourceSyncedAt(t *testing.T, cacheDir, name string, at time.Time) {
	t.Helper()
	stateDir := filepath.Join(cacheDir, "source-state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir source state: %v", err)
	}
	data := []byte(strconv.FormatInt(at.Unix(), 10) + "\n")
	if err := os.WriteFile(filepath.Join(stateDir, name+".synced_at"), data, 0o644); err != nil {
		t.Fatalf("write source sync state: %v", err)
	}
}

func coreBackend(
	t *testing.T,
	repo string,
	cwd string,
	configDir string,
	cacheDir string,
	extraEnv map[string]string,
) *core.Backend {
	t.Helper()
	env := map[string]string{
		testEnvHome:           filepath.Join(filepath.Dir(configDir), "home"),
		"SKILLHUB_CONFIG_DIR": configDir,
		"SKILLHUB_CACHE_DIR":  cacheDir,
	}
	for key, value := range extraEnv {
		env[key] = value
	}
	backend, err := core.New(core.Context{
		RepoRoot: repo, CallerCWD: cwd, Env: env,
		Now: func() time.Time { return time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new backend: %v", err)
	}
	return backend
}

func runCoreGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func commitCoreGit(t *testing.T, dir string) {
	t.Helper()
	runCoreGit(t, dir, "add", ".")
	runCoreGit(t, dir, "-c", "user.name=Skillhub Test", "-c", "user.email=skillhub@example.test", "commit", "-m", "snapshot")
}

func TestCoreAddGitHubTreeURLMaterializesNestedSkills(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "acton-contracts")
	writeCoreRepo(t, repo)
	writeCoreSkill(t, sourceDir, "func2tolk", "FunC to Tolk migration", "v1")
	writeCoreSkill(t, sourceDir, "tolk", "Tolk contracts", "v1")
	runCoreGit(t, sourceDir, "init")
	runCoreGit(t, sourceDir, "checkout", "-b", "main")
	commitCoreGit(t, sourceDir)
	runCoreGit(t, sourceDir, "checkout", "-b", "skills")
	gitConfig := filepath.Join(tmp, "gitconfig")
	gitConfigData := "[url \"file://" + filepath.ToSlash(sourceDir) + "\"]\n" +
		"\tinsteadOf = https://github.com/ton-blockchain/acton-contracts\n"
	if err := os.WriteFile(gitConfig, []byte(gitConfigData), 0o644); err != nil {
		t.Fatalf("write git config: %v", err)
	}
	backend := coreBackend(t, repo, tmp, configDir, cacheDir, map[string]string{"GIT_CONFIG_GLOBAL": gitConfig})

	if _, err := backend.AddSource(core.SourceAddOptions{
		Location: "https://github.com/ton-blockchain/acton-contracts/tree/skills/skills/",
		Name:     "acton",
	}); err != nil {
		t.Fatalf("add source: %v", err)
	}
	sources, err := backend.ListSources()
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 ||
		sources[0].Location != "https://github.com/ton-blockchain/acton-contracts" ||
		sources[0].Ref != "skills" {
		t.Fatalf("unexpected normalized source: %#v", sources)
	}
	catalog, err := os.ReadFile(filepath.Join(cacheDir, "generated-sources", "acton", "catalog", "skills.tsv"))
	if err != nil {
		t.Fatalf("read generated catalog: %v", err)
	}
	if !strings.Contains(string(catalog), "func2tolk\tfunc2tolk\t") || !strings.Contains(string(catalog), "tolk\ttolk\t") {
		t.Fatalf("expected generated acton skills, got:\n%s", catalog)
	}
	statuses, err := backend.ListSourceStatuses()
	if err != nil {
		t.Fatalf("list source statuses: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Status != core.SourceStatusFresh {
		t.Fatalf("expected rewritten git origin to be treated as fresh, got %#v", statuses)
	}
}

func TestCoreAddGitSourceSupportsTagRef(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceDir, "rules-selector")
	runCoreGit(t, sourceDir, "init")
	runCoreGit(t, sourceDir, "checkout", "-b", "main")
	commitCoreGit(t, sourceDir)
	runCoreGit(t, sourceDir, "tag", "v1")
	backend := coreBackend(t, repo, tmp, configDir, cacheDir, nil)

	if _, err := backend.AddSource(core.SourceAddOptions{
		Location: sourceDir,
		Name:     "tagged",
		Type:     core.SourceTypeGit,
		Ref:      "v1",
	}); err != nil {
		t.Fatalf("add tagged git source: %v", err)
	}
	sources, err := backend.ListSources()
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 || sources[0].Ref != "v1" {
		t.Fatalf("expected tag ref source, got %#v", sources)
	}
}

func TestCoreListSourceStatusesReturnsFreshStaleMissingLocal(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	localSource := filepath.Join(tmp, "local-source")
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, localSource, "local-rules")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	sources := core.SourcesHeader + "\n" +
		"local\tpath\t" + localSource + "\t-\tcatalog/skills.tsv\n" +
		"fresh\tgit\t" + filepath.Join(tmp, "fresh-remote") + "\tmain\tcatalog/skills.tsv\n" +
		"stale\tgit\t" + filepath.Join(tmp, "stale-remote") + "\tmain\tcatalog/skills.tsv\n" +
		"missing\tgit\t" + filepath.Join(tmp, "missing-remote") + "\tmain\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(sources), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
	writeCoreCatalog(t, filepath.Join(cacheDir, "sources", "fresh"), "fresh-rules")
	writeCoreCatalog(t, filepath.Join(cacheDir, "sources", "stale"), "stale-rules")
	now := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	writeCoreSourceSyncedAt(t, cacheDir, "fresh", now)
	writeCoreSourceSyncedAt(t, cacheDir, "stale", now.Add(-11*time.Minute))
	backend := coreBackend(t, repo, tmp, configDir, cacheDir, nil)

	statuses, err := backend.ListSourceStatuses()
	if err != nil {
		t.Fatalf("list source statuses: %v", err)
	}
	got := map[string]string{}
	for _, status := range statuses {
		got[status.Name] = status.Status
	}
	want := map[string]string{
		"local":   core.SourceStatusLocal,
		"fresh":   core.SourceStatusFresh,
		"stale":   core.SourceStatusStale,
		"missing": core.SourceStatusMissing,
	}
	for name, status := range want {
		if got[name] != status {
			t.Fatalf("expected %s status %s, got statuses %#v", name, status, statuses)
		}
	}
}

func TestCoreListSourceStatusesRejectsCacheLocalURLRewrite(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceA := filepath.Join(tmp, "source-a")
	sourceB := filepath.Join(tmp, "source-b")
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceA, "old-rules")
	writeCoreCatalog(t, sourceB, "new-rules")
	runCoreGit(t, sourceA, "init")
	runCoreGit(t, sourceA, "checkout", "-b", "main")
	commitCoreGit(t, sourceA)
	runCoreGit(t, sourceB, "init")
	runCoreGit(t, sourceB, "checkout", "-b", "main")
	commitCoreGit(t, sourceB)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	sources := core.SourcesHeader + "\n" +
		"shared\tgit\t" + sourceB + "\tmain\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(sources), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
	cachedSource := filepath.Join(cacheDir, "sources", "shared")
	if err := os.MkdirAll(filepath.Dir(cachedSource), 0o755); err != nil {
		t.Fatalf("mkdir cache sources: %v", err)
	}
	runCoreGit(t, tmp, "clone", sourceA, cachedSource)
	runCoreGit(t, cachedSource, "config", "url."+sourceA+".insteadOf", sourceB)
	writeCoreSourceSyncedAt(t, cacheDir, "shared", time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC))
	backend := coreBackend(t, repo, tmp, configDir, cacheDir, nil)

	statuses, err := backend.ListSourceStatuses()
	if err != nil {
		t.Fatalf("list source statuses: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Status != core.SourceStatusError {
		t.Fatalf("expected cache-local rewrite to be rejected, got %#v", statuses)
	}
}

func TestCoreListSkillsUsesStaleCacheWithoutSync(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	writeCoreRepo(t, repo)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	sources := core.SourcesHeader + "\n" +
		"cached\tgit\t" + filepath.Join(tmp, "missing-remote") + "\tmain\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(sources), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
	writeCoreCatalog(t, filepath.Join(cacheDir, "sources", "cached"), "stale-rules")
	writeCoreSourceSyncedAt(t, cacheDir, "cached", time.Date(2026, 5, 14, 9, 0, 0, 0, time.UTC))
	backend := coreBackend(t, repo, tmp, configDir, cacheDir, nil)

	skills, warning, err := backend.ListSkills("")
	if err != nil {
		t.Fatalf("list skills should use stale cache: %v", err)
	}
	if len(skills) != 1 || skills[0].Name != "stale-rules" {
		t.Fatalf("expected stale cached skill, got %#v", skills)
	}
	if !strings.Contains(warning, "Warning: using stale cache for source cached") {
		t.Fatalf("expected stale cache warning, got %q", warning)
	}
}

func TestCoreListSkillsPartialSourceFailureReturnsSkillsAndWarning(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	localSource := filepath.Join(tmp, "local-source")
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, localSource, "local-rules")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	sources := core.SourcesHeader + "\n" +
		"local\tpath\t" + localSource + "\t-\tcatalog/skills.tsv\n" +
		"missing\tgit\t" + filepath.Join(tmp, "missing-remote") + "\tmain\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(sources), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
	backend := coreBackend(t, repo, tmp, configDir, cacheDir, nil)

	skills, warning, err := backend.ListSkills("")
	if err != nil {
		t.Fatalf("partial source failure should not fail: %v", err)
	}
	if len(skills) != 1 || skills[0].Name != "local-rules" {
		t.Fatalf("expected available local skill, got %#v", skills)
	}
	if !strings.Contains(warning, "source missing unavailable") ||
		!strings.Contains(warning, "Run: skillhub sources sync missing") {
		t.Fatalf("expected missing source warning, got %q", warning)
	}
}

func TestCoreListSkillsAllMissingCachesReturnsActionableError(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	writeCoreRepo(t, repo)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	sources := core.SourcesHeader + "\n" +
		"missing\tgit\t" + filepath.Join(tmp, "missing-remote") + "\tmain\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(sources), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
	backend := coreBackend(t, repo, tmp, configDir, cacheDir, nil)

	_, warning, err := backend.ListSkills("")
	if err == nil {
		t.Fatalf("expected missing cache error")
	}
	if !strings.Contains(err.Error(), "Run: skillhub sources sync missing") {
		t.Fatalf("expected actionable sync error, got %v", err)
	}
	if !strings.Contains(warning, "source missing unavailable") {
		t.Fatalf("expected warning for missing source, got %q", warning)
	}
}

func TestCoreProjectInstallWritesLockfileAndRestoreUsesLockSource(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	restoreConfigDir := filepath.Join(tmp, "restore-config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, core.ScopeProject)
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceDir, ruleSelector)
	writeCoreSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	backend := coreBackend(t, repo, projectDir, configDir, cacheDir, nil)

	if _, _, err := backend.Install(core.InstallOptions{
		Names:   []string{ruleSelector},
		Target:  core.TargetCodex,
		Scope:   core.ScopeProject,
		Project: projectDir,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	lockfile, err := os.ReadFile(filepath.Join(projectDir, "skills.lock.toml"))
	if err != nil {
		t.Fatalf("read lockfile: %v", err)
	}
	if strings.Contains(string(lockfile), projectDir) ||
		!strings.Contains(string(lockfile), `installed_path = ".agents/skills/rules-selector"`) {
		t.Fatalf("lockfile should be portable, got:\n%s", lockfile)
	}
	installedPath := filepath.Join(projectDir, ".agents", "skills", ruleSelector)
	if err := os.RemoveAll(installedPath); err != nil {
		t.Fatalf("remove install: %v", err)
	}
	if err := os.MkdirAll(restoreConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir restore config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(restoreConfigDir, "sources.tsv"), []byte("broken\n"), 0o644); err != nil {
		t.Fatalf("write broken sources: %v", err)
	}
	restoreBackend := coreBackend(t, repo, projectDir, restoreConfigDir, filepath.Join(tmp, "restore-cache"), nil)
	summary, err := restoreBackend.Restore(core.RestoreOptions{Project: projectDir})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if summary.Installed != 1 || summary.Updated != 0 || summary.Skipped != 0 || summary.Failed != 0 {
		t.Fatalf("unexpected restore summary: %#v", summary)
	}
	if _, err := os.Stat(filepath.Join(installedPath, "SKILL.md")); err != nil {
		t.Fatalf("expected restored skill: %v", err)
	}
}

func TestCoreProjectInstallPartialSuccessWritesLockfile(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, core.ScopeProject)
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceDir, "go-project-rules")
	writeCoreSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	backend := coreBackend(t, repo, projectDir, configDir, cacheDir, nil)

	output, warning, err := backend.Install(core.InstallOptions{
		Names:   []string{"go-project-rules", "missing-skill"},
		Target:  core.TargetCodex,
		Scope:   core.ScopeProject,
		Project: projectDir,
	})
	if err == nil || !strings.Contains(err.Error(), "unknown skill: missing-skill") {
		t.Fatalf("expected missing skill error, got output=%q warning=%q err=%v", output, warning, err)
	}
	installedPath := filepath.Join(projectDir, ".agents", "skills", "go-project-rules")
	if _, err := os.Stat(filepath.Join(installedPath, "SKILL.md")); err != nil {
		t.Fatalf("expected successful skill install to remain: %v", err)
	}
	registry, err := os.ReadFile(filepath.Join(configDir, "installed.tsv"))
	if err != nil {
		t.Fatalf("read installed registry: %v", err)
	}
	if !strings.Contains(string(registry), "local\tgo-project-rules\tcodex\tproject\t"+projectDir) {
		t.Fatalf("expected successful install in registry, got:\n%s", registry)
	}
	lockfile, err := os.ReadFile(filepath.Join(projectDir, "skills.lock.toml"))
	if err != nil {
		t.Fatalf("expected lockfile for successful partial install: %v", err)
	}
	if !strings.Contains(string(lockfile), `skill = "go-project-rules"`) {
		t.Fatalf("expected successful install in lockfile, got:\n%s", lockfile)
	}
}

func TestCoreInstallRefusesUnmanagedDirectorySkill(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	targetDir := filepath.Join(tmp, "target")
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceDir, ruleSelector)
	writeCoreSources(t, configDir, sourceDir)
	installedPath := filepath.Join(targetDir, ruleSelector)
	if err := os.MkdirAll(installedPath, 0o755); err != nil {
		t.Fatalf("mkdir unmanaged skill: %v", err)
	}
	manualContent := []byte("# rules-selector\n\nmanual\n")
	if err := os.WriteFile(filepath.Join(installedPath, "SKILL.md"), manualContent, 0o644); err != nil {
		t.Fatalf("write unmanaged skill: %v", err)
	}
	backend := coreBackend(t, repo, tmp, configDir, cacheDir, nil)

	_, _, err := backend.Install(core.InstallOptions{
		Names:  []string{ruleSelector},
		Target: core.TargetDirectory,
		Dir:    targetDir,
	})
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite unmanaged skill") {
		t.Fatalf("expected unmanaged overwrite refusal, got %v", err)
	}
	content, readErr := os.ReadFile(filepath.Join(installedPath, "SKILL.md"))
	if readErr != nil {
		t.Fatalf("read unmanaged skill: %v", readErr)
	}
	if string(content) != string(manualContent) {
		t.Fatalf("unmanaged skill was overwritten, got:\n%s", content)
	}
}

func TestCoreProjectInstallRefusesUnmanagedSkill(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceDir, ruleSelector)
	writeCoreSources(t, configDir, sourceDir)
	installedPath := filepath.Join(projectDir, ".agents", "skills", ruleSelector)
	if err := os.MkdirAll(installedPath, 0o755); err != nil {
		t.Fatalf("mkdir unmanaged project skill: %v", err)
	}
	manualContent := []byte("# rules-selector\n\nproject manual\n")
	if err := os.WriteFile(filepath.Join(installedPath, "SKILL.md"), manualContent, 0o644); err != nil {
		t.Fatalf("write unmanaged project skill: %v", err)
	}
	sidecar := []byte(`{"source":"local","skill":"rules-selector"}` + "\n")
	if err := os.WriteFile(filepath.Join(installedPath, ".skillhub.json"), sidecar, 0o644); err != nil {
		t.Fatalf("write stale project sidecar: %v", err)
	}
	backend := coreBackend(t, repo, projectDir, configDir, cacheDir, nil)

	_, _, err := backend.Install(core.InstallOptions{
		Names:   []string{ruleSelector},
		Target:  core.TargetCodex,
		Scope:   core.ScopeProject,
		Project: projectDir,
	})
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite unmanaged skill") {
		t.Fatalf("expected unmanaged project overwrite refusal, got %v", err)
	}
	content, readErr := os.ReadFile(filepath.Join(installedPath, "SKILL.md"))
	if readErr != nil {
		t.Fatalf("read unmanaged project skill: %v", readErr)
	}
	if string(content) != string(manualContent) {
		t.Fatalf("unmanaged project skill was overwritten, got:\n%s", content)
	}
}

func TestCoreRestoreFailureReturnsErrorAndPreservesLockfile(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	restoreConfigDir := filepath.Join(tmp, "restore-config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, core.ScopeProject)
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceDir, "rules-selector")
	writeCoreSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	backend := coreBackend(t, repo, projectDir, configDir, cacheDir, nil)
	if _, _, err := backend.Install(core.InstallOptions{
		Names:   []string{ruleSelector},
		Target:  core.TargetCodex,
		Scope:   core.ScopeProject,
		Project: projectDir,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	lockfilePath := filepath.Join(projectDir, "skills.lock.toml")
	lockfile, err := os.ReadFile(lockfilePath)
	if err != nil {
		t.Fatalf("read lockfile: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(projectDir, ".agents")); err != nil {
		t.Fatalf("remove target root parent: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".agents"), 0o755); err != nil {
		t.Fatalf("mkdir target root parent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".agents", "skills"), []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("block target root: %v", err)
	}
	if err := os.MkdirAll(restoreConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir restore config: %v", err)
	}
	restoreBackend := coreBackend(t, repo, projectDir, restoreConfigDir, filepath.Join(tmp, "restore-cache"), nil)

	summary, err := restoreBackend.Restore(core.RestoreOptions{Project: projectDir})
	if err == nil || !strings.Contains(err.Error(), "restore failed for 1 skill") {
		t.Fatalf("expected restore failure, got summary=%#v err=%v", summary, err)
	}
	if summary.Failed != 1 || summary.Installed != 0 || summary.Updated != 0 {
		t.Fatalf("expected one failed restore and no successful writes, got %#v", summary)
	}
	if len(summary.Rows) != 1 || summary.Rows[0].Status != core.ResultFailed {
		t.Fatalf("expected failed row status, got %#v", summary.Rows)
	}
	preserved, err := os.ReadFile(lockfilePath)
	if err != nil {
		t.Fatalf("lockfile should remain after failed restore: %v", err)
	}
	if string(preserved) != string(lockfile) {
		t.Fatalf("failed restore should preserve lockfile, got:\n%s", preserved)
	}
	if _, err := os.Stat(filepath.Join(restoreConfigDir, "installed.tsv")); !os.IsNotExist(err) {
		t.Fatalf("failed restore should not write restore registry, stat err=%v", err)
	}
}

func TestCoreRestoreRejectsLockfilePathsOutsideProject(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, core.ScopeProject)
	outsideSkillDir := filepath.Join(tmp, "outside", "rules-selector")
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceDir, "rules-selector")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.MkdirAll(outsideSkillDir, 0o755); err != nil {
		t.Fatalf("mkdir outside skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outsideSkillDir, "KEEP"), []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("write outside marker: %v", err)
	}
	lockfile := "lockfile_version = 1\n\n" +
		"[[skills]]\n" +
		"source = \"local\"\n" +
		"source_type = \"path\"\n" +
		"skill = \"rules-selector\"\n" +
		"qualified_skill = \"local/rules-selector\"\n" +
		"target = \"codex\"\n" +
		"scope = \"project\"\n" +
		"project_path = \".\"\n" +
		"target_root = \"../outside\"\n" +
		"installed_path = \"../outside/rules-selector\"\n" +
		"source_ref = \"-\"\n" +
		"source_location = \"" + filepath.ToSlash(sourceDir) + "\"\n" +
		"catalog = \"catalog/skills.tsv\"\n" +
		"content_hash = \"-\"\n" +
		"installed_at = \"-\"\n" +
		"updated_at = \"-\"\n"
	if err := os.WriteFile(filepath.Join(projectDir, "skills.lock.toml"), []byte(lockfile), 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}
	backend := coreBackend(t, repo, projectDir, configDir, cacheDir, nil)

	summary, err := backend.Restore(core.RestoreOptions{Project: projectDir})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if summary.Skipped != 1 || summary.Failed != 0 || len(summary.Rows) != 1 {
		t.Fatalf("expected skipped unsafe lockfile row, got %#v", summary)
	}
	if !strings.Contains(summary.Rows[0].Reason, "escapes project") {
		t.Fatalf("expected escape reason, got %#v", summary.Rows[0])
	}
	if _, err := os.Stat(filepath.Join(outsideSkillDir, "KEEP")); err != nil {
		t.Fatalf("outside marker should remain untouched: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outsideSkillDir, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("outside skill should not be overwritten, stat err=%v", err)
	}
}

func TestCoreTargetRootRejectsInvalidScope(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	writeCoreRepo(t, repo)
	backend := coreBackend(t, repo, tmp, configDir, cacheDir, nil)

	_, err := backend.TargetRoot(core.TargetRootOptions{Target: core.TargetCodex, Scope: "nope"})
	if err == nil || !strings.Contains(err.Error(), "invalid scope") {
		t.Fatalf("expected invalid scope error, got %v", err)
	}
}

func TestCoreTargetRootResolvesAntigravityPaths(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	projectDir := filepath.Join(tmp, "project")
	homeDir := filepath.Join(tmp, "home")
	writeCoreRepo(t, repo)
	backend := coreBackend(t, repo, tmp, configDir, cacheDir, map[string]string{testEnvHome: homeDir})

	globalRoot, err := backend.TargetRoot(core.TargetRootOptions{
		Target: core.TargetAntigravity,
		Scope:  core.ScopeGlobal,
	})
	if err != nil {
		t.Fatalf("resolve antigravity global root: %v", err)
	}
	wantGlobal := filepath.Join(homeDir, ".gemini", "antigravity", "skills")
	if globalRoot != wantGlobal {
		t.Fatalf("expected antigravity global root %q, got %q", wantGlobal, globalRoot)
	}

	projectRoot, err := backend.TargetRoot(core.TargetRootOptions{
		Target:  core.TargetAntigravity,
		Scope:   core.ScopeProject,
		Project: projectDir,
	})
	if err != nil {
		t.Fatalf("resolve antigravity project root: %v", err)
	}
	wantProject := filepath.Join(projectDir, ".agents", "skills")
	if projectRoot != wantProject {
		t.Fatalf("expected antigravity project root %q, got %q", wantProject, projectRoot)
	}
}

func TestCoreStandaloneBackendUsesEmbeddedRegistries(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	callerDir := filepath.Join(tmp, "caller")
	homeDir := filepath.Join(tmp, "home")
	if err := os.MkdirAll(callerDir, 0o755); err != nil {
		t.Fatalf("mkdir caller: %v", err)
	}
	backend := coreBackend(t, "", callerDir, configDir, cacheDir, map[string]string{testEnvHome: homeDir})

	sources, err := backend.ListDefaultSources()
	if err != nil {
		t.Fatalf("list embedded default sources: %v", err)
	}
	if len(sources) == 0 || sources[0].Name != core.SourceNameAgentRules {
		t.Fatalf("expected embedded default sources, got %#v", sources)
	}
	targets, err := backend.ListTargets()
	if err != nil {
		t.Fatalf("list embedded targets: %v", err)
	}
	foundAntigravity := false
	for _, target := range targets {
		if target.ID == core.TargetAntigravity {
			foundAntigravity = true
		}
	}
	if !foundAntigravity {
		t.Fatalf("expected embedded antigravity target, got %#v", targets)
	}
	path, err := backend.SourcePath(core.Source{
		Name:     "relative",
		Type:     core.SourceTypePath,
		Location: "relative-source",
		Ref:      "-",
		Catalog:  "catalog/skills.tsv",
	})
	if err != nil {
		t.Fatalf("resolve relative source path: %v", err)
	}
	wantPath := filepath.Join(callerDir, "relative-source")
	if path != wantPath {
		t.Fatalf("expected relative source path %q, got %q", wantPath, path)
	}
}

func TestCoreUpdateReportsProjectLockfileWriteFailure(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, core.ScopeProject)
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceDir, "rules-selector")
	writeCoreSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	backend := coreBackend(t, repo, projectDir, configDir, cacheDir, nil)
	if _, _, err := backend.Install(core.InstallOptions{
		Names:   []string{"rules-selector"},
		Target:  core.TargetCodex,
		Scope:   core.ScopeProject,
		Project: projectDir,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	lockfile := filepath.Join(projectDir, "skills.lock.toml")
	if err := os.Remove(lockfile); err != nil {
		t.Fatalf("remove lockfile: %v", err)
	}
	if err := os.Mkdir(lockfile, 0o755); err != nil {
		t.Fatalf("replace lockfile with directory: %v", err)
	}

	summary, err := backend.UpdateInstalled(core.InstalledUpdateOptions{
		Target:  core.TargetCodex,
		Scope:   core.ScopeProject,
		Project: projectDir,
	})
	if err == nil {
		t.Fatalf("expected update failure, got summary %#v", summary)
	}
	if summary.Failed != 1 || summary.Unchanged != 0 {
		t.Fatalf("expected failed metadata persistence, got %#v", summary)
	}
	if !strings.Contains(summary.Output, "could not persist install metadata") {
		t.Fatalf("expected metadata persistence failure output, got:\n%s", summary.Output)
	}
}

func TestCoreTargetsDetectCountsRegistryManagedProjectSkill(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, core.ScopeProject)
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceDir, "rules-selector")
	writeCoreSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	backend := coreBackend(t, repo, projectDir, configDir, cacheDir, nil)
	if _, _, err := backend.Install(core.InstallOptions{
		Names:   []string{"rules-selector"},
		Target:  core.TargetCodex,
		Scope:   core.ScopeProject,
		Project: projectDir,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	rows, err := backend.DetectTargets(projectDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	wantPath := filepath.Join(projectDir, ".agents", "skills")
	for _, row := range rows {
		if row.Target == core.TargetCodex && row.Scope == core.ScopeProject && row.Path == wantPath {
			if row.Skills != "1" || row.Managed != "1" {
				t.Fatalf("expected one managed project skill, got %#v", row)
			}
			return
		}
	}
	t.Fatalf("codex project detection missing: %#v", rows)
}

func TestCoreListInstalledDoesNotUseLegacyDirForExplicitTarget(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	projectDir := filepath.Join(tmp, core.ScopeProject)
	legacyDir := filepath.Join(tmp, "legacy-skills")
	writeCoreRepo(t, repo)
	if err := os.MkdirAll(filepath.Join(legacyDir, "legacy-only"), 0o755); err != nil {
		t.Fatalf("mkdir legacy skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "legacy-only", "SKILL.md"), []byte("# Legacy\n"), 0o644); err != nil {
		t.Fatalf("write legacy skill: %v", err)
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	backend := coreBackend(t, repo, projectDir, configDir, cacheDir, map[string]string{"AGENT_SKILLS_DIR": legacyDir})

	legacyRows, err := backend.ListInstalled(core.InstalledListOptions{})
	if err != nil {
		t.Fatalf("list legacy installed: %v", err)
	}
	if len(legacyRows) != 1 || legacyRows[0].Target != core.TargetCodex || legacyRows[0].Scope != core.ScopeGlobal {
		t.Fatalf("expected default list to use legacy Codex dir, got %#v", legacyRows)
	}

	rows, err := backend.ListInstalled(core.InstalledListOptions{
		Target:  core.TargetClaude,
		Scope:   core.ScopeProject,
		Project: projectDir,
	})
	if err != nil {
		t.Fatalf("list explicit installed: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("explicit target/scope must not scan AGENT_SKILLS_DIR, got %#v", rows)
	}

	explicitCodexRows, err := backend.ListInstalled(core.InstalledListOptions{
		Target: core.TargetCodex,
		Scope:  core.ScopeGlobal,
	})
	if err != nil {
		t.Fatalf("list explicit codex installed: %v", err)
	}
	if len(explicitCodexRows) != 1 || explicitCodexRows[0].Path != filepath.Join(legacyDir, "legacy-only") {
		t.Fatalf("explicit Codex global must use AGENT_SKILLS_DIR, got %#v", explicitCodexRows)
	}
}

func TestCoreRecommendRanksProjectSignals(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, core.ScopeProject)
	writeCoreRepo(t, repo)
	writeCoreCatalog(t, sourceDir, "go-project-rules", "docs-project-rules")
	catalog := "name\tcategory\ttriggers\tdescription\n" +
		"go-project-rules\tgo\tgo,golang,go.mod\tGo rules\n" +
		"docs-project-rules\tdocumentation\tdocs,readme\tDocs rules\n"
	if err := os.WriteFile(filepath.Join(sourceDir, "catalog", "skills.tsv"), []byte(catalog), 0o644); err != nil {
		t.Fatalf("rewrite catalog: %v", err)
	}
	writeCoreSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/project\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	backend := coreBackend(t, repo, projectDir, configDir, cacheDir, nil)
	rows, warning, err := backend.Recommend(core.RecommendOptions{Project: projectDir})
	if err != nil {
		t.Fatalf("recommend: %v", err)
	}
	if warning != "" {
		t.Fatalf("expected no recommendation warning, got %q", warning)
	}
	if len(rows) == 0 || rows[0].Skill != "go-project-rules" || !strings.Contains(rows[0].Reason, "go.mod") {
		t.Fatalf("expected go recommendation first, got %#v", rows)
	}
	if rows[0].Score != 116 {
		t.Fatalf("expected unchanged go recommendation score, got %#v", rows[0])
	}
}
