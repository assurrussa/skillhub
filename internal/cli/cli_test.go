package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/assurrussa/skillhub/internal/cli"
	"github.com/assurrussa/skillhub/internal/core"
)

const (
	testArgAdd         = "add"
	testCommandSkills  = "skills"
	testCommandSources = "sources"
	testCommandTargets = "targets"
	testDirDefaults    = "defaults"
	testFlagTSV        = "--tsv"
	testSubcommandList = "list"
	testRepoTargetsDir = testCommandTargets
)

func testRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func runCLIForTest(t *testing.T, env []string, command string, args ...string) (string, error) {
	t.Helper()
	stdout, stderr, err := runCLISplitForTest(t, env, command, args...)
	return stdout + stderr, err
}

func skillhubConfigEnv(configDir string) []string {
	return []string{"SKILLHUB_CONFIG_DIR=" + configDir}
}

func runCLIWithConfig(t *testing.T, configDir string, command string, args ...string) (string, error) {
	t.Helper()
	return runCLIForTest(t, skillhubConfigEnv(configDir), command, args...)
}

func installedUsageFixtureRow(source, skill, target, projectPath, sourceLocation string) string {
	targetRoot := filepath.Join(projectPath, "."+target, "skills")
	if target == "codex" {
		targetRoot = filepath.Join(projectPath, ".agents", "skills")
	}
	return strings.Join([]string{
		source,
		skill,
		target,
		"project",
		projectPath,
		targetRoot,
		filepath.Join(targetRoot, skill),
		"-",
		sourceLocation,
		"catalog/skills.tsv",
		"old",
		"2026-05-05T00:00:00Z",
		"2026-05-05T00:00:00Z",
	}, "\t")
}

func overwriteFile(t *testing.T, path string, data string) {
	t.Helper()
	file, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		t.Fatalf("open file for overwrite: %v", err)
	}
	if _, err := file.WriteString(data); err != nil {
		_ = file.Close()
		t.Fatalf("overwrite file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close overwritten file: %v", err)
	}
}

func runCLISplitForTest(t *testing.T, env []string, command string, args ...string) (stdout string, stderr string, err error) {
	t.Helper()
	root := testRepoRoot(t)
	cliArgs, err := commandCLIArgs(command, args...)
	if err != nil {
		return "", err.Error() + "\n", err
	}
	restoreEnv := applyTestEnv(append([]string{
		"SKILLHUB_REPO=" + root,
		"SKILLHUB_CALLER_CWD=" + root,
	}, env...))
	defer restoreEnv()

	var outBuf, errBuf bytes.Buffer
	cmd := cli.NewRootCommand()
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(cliArgs)
	err = cmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintln(&errBuf, err)
	}
	return outBuf.String(), errBuf.String(), err
}

func commandCLIArgs(command string, args ...string) ([]string, error) {
	switch command {
	case testCommandSkills:
		return append([]string{testCommandSkills}, args...), nil
	case testCommandSources:
		return append([]string{testCommandSources}, args...), nil
	case "installed":
		return append([]string{"installed"}, args...), nil
	case testCommandTargets:
		return append([]string{testCommandTargets}, args...), nil
	case "recommend":
		return append([]string{"recommend"}, args...), nil
	default:
		return nil, fmt.Errorf("unsupported CLI test command: %s", command)
	}
}

func applyTestEnv(env []string) func() {
	type oldValue struct {
		value string
		ok    bool
	}
	old := map[string]oldValue{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if _, seen := old[key]; !seen {
			previous, existed := os.LookupEnv(key)
			old[key] = oldValue{value: previous, ok: existed}
		}
		_ = os.Setenv(key, value)
	}
	return func() {
		for key, previous := range old {
			if previous.ok {
				_ = os.Setenv(key, previous.value)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	}
}

func writeFakeSkillhubRepo(t *testing.T, root string) {
	t.Helper()
	writeFakeSkillhubRuntimeFiles(t, root)
	for _, dir := range []string{testDirDefaults, testRepoTargetsDir} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir fake repo dir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(
		filepath.Join(root, testDirDefaults, "sources.tsv"),
		[]byte("name\ttype\tlocation\tref\tcatalog\n"),
		0o644,
	); err != nil {
		t.Fatalf("write fake defaults: %v", err)
	}
	targets := "id\tlabel\tstatus\tadapter\tdescription\n" +
		"codex\tCodex\tsupported\tskill-dir\tCodex skills\n" +
		"directory\tDirectory\tsupported\tskill-dir\tDirectory skills\n"
	if err := os.WriteFile(filepath.Join(root, testRepoTargetsDir, "targets.tsv"), []byte(targets), 0o644); err != nil {
		t.Fatalf("write fake targets: %v", err)
	}
}

func writeFakeSkillhubRuntimeFiles(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "skillhub"), 0o755); err != nil {
		t.Fatalf("mkdir fake runtime dir: %v", err)
	}
	goMod := []byte("module example.com/fake-skillhub\n\ngo 1.26.0\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), goMod, 0o644); err != nil {
		t.Fatalf("write fake go.mod: %v", err)
	}
	mainGo := []byte("package main\nfunc main() {}\n")
	if err := os.WriteFile(filepath.Join(root, "cmd", "skillhub", "main.go"), mainGo, 0o644); err != nil {
		t.Fatalf("write fake main.go: %v", err)
	}
}

func TestTopLevelAddAliasesInstall(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	targetDir := filepath.Join(tmp, "target")
	writeFakeSkillhubRepo(t, repo)
	writeInstallableTestSource(t, sourceDir, "go-project-rules", "v1")
	writeTestSources(t, configDir, sourceDir)
	t.Setenv("SKILLHUB_REPO", repo)
	t.Setenv("SKILLHUB_CONFIG_DIR", configDir)

	cmd := cli.NewRootCommand()
	cmd.SetArgs([]string{
		testArgAdd,
		"local/go-project-rules",
		"--target",
		"directory",
		"--dir",
		targetDir,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("skillhub add failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "go-project-rules", "SKILL.md")); err != nil {
		t.Fatalf("expected add alias to install skill through Go backend: %v", err)
	}
}

func TestTargetsListUsesGoBackend(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	writeFakeSkillhubRuntimeFiles(t, repo)
	for _, dir := range []string{testDirDefaults, testRepoTargetsDir} {
		if err := os.MkdirAll(filepath.Join(repo, dir), 0o755); err != nil {
			t.Fatalf("mkdir fake repo dir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(
		filepath.Join(repo, testDirDefaults, "sources.tsv"),
		[]byte("name\ttype\tlocation\tref\tcatalog\n"),
		0o644,
	); err != nil {
		t.Fatalf("write defaults: %v", err)
	}
	targets := "id\tlabel\tstatus\tadapter\tdescription\n" +
		"codex\tCodex\tsupported\tskill-dir\tCodex skills\n"
	if err := os.WriteFile(filepath.Join(repo, testRepoTargetsDir, "targets.tsv"), []byte(targets), 0o644); err != nil {
		t.Fatalf("write targets: %v", err)
	}
	t.Setenv("SKILLHUB_REPO", repo)

	cmd := cli.NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{testCommandTargets, testSubcommandList, testFlagTSV})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("targets list should use Go backend: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "codex\tCodex\tsupported\tskill-dir\tCodex skills") {
		t.Fatalf("unexpected targets output:\n%s", got)
	}
}

func TestRepoDiscoveryUsesRuntimeMarkers(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	writeFakeSkillhubRuntimeFiles(t, repo)
	for _, dir := range []string{testDirDefaults, testRepoTargetsDir} {
		if err := os.MkdirAll(filepath.Join(repo, dir), 0o755); err != nil {
			t.Fatalf("mkdir fake repo dir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(
		filepath.Join(repo, testDirDefaults, "sources.tsv"),
		[]byte(core.SourcesHeader+"\n"),
		0o644,
	); err != nil {
		t.Fatalf("write defaults: %v", err)
	}
	targets := core.TargetsHeader + "\n" +
		"codex\tCodex\tsupported\tskill-dir\tCodex skills\n"
	if err := os.WriteFile(filepath.Join(repo, testRepoTargetsDir, "targets.tsv"), []byte(targets), 0o644); err != nil {
		t.Fatalf("write targets: %v", err)
	}
	t.Setenv("SKILLHUB_REPO", repo)

	cmd := cli.NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{testCommandTargets, testSubcommandList, testFlagTSV})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("targets list should work from runtime markers: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "codex\tCodex\tsupported\tskill-dir\tCodex skills") {
		t.Fatalf("unexpected targets output:\n%s", got)
	}
}

func TestBinSkillhubFailsWithoutGo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bin/skillhub is a POSIX shell wrapper")
	}
	root := testRepoRoot(t)
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	writeFakeSkillhubRuntimeFiles(t, repo)
	for _, dir := range []string{"bin", testDirDefaults, testRepoTargetsDir} {
		if err := os.MkdirAll(filepath.Join(repo, dir), 0o755); err != nil {
			t.Fatalf("mkdir fake repo dir %s: %v", dir, err)
		}
	}
	wrapper, err := os.ReadFile(filepath.Join(root, "bin", "skillhub"))
	if err != nil {
		t.Fatalf("read bin/skillhub: %v", err)
	}
	wrapperPath := filepath.Join(repo, "bin", "skillhub")
	// #nosec G306,G703 -- wrapperPath is inside a test temp directory.
	if err := os.WriteFile(wrapperPath, wrapper, 0o755); err != nil {
		t.Fatalf("write fake wrapper: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(repo, testDirDefaults, "sources.tsv"),
		[]byte(core.SourcesHeader+"\n"),
		0o644,
	); err != nil {
		t.Fatalf("write defaults: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(repo, testRepoTargetsDir, "targets.tsv"),
		[]byte(core.TargetsHeader+"\n"),
		0o644,
	); err != nil {
		t.Fatalf("write targets: %v", err)
	}

	pathDir := filepath.Join(tmp, "path")
	if err := os.MkdirAll(pathDir, 0o755); err != nil {
		t.Fatalf("mkdir PATH dir: %v", err)
	}
	dirnamePath, err := exec.LookPath("dirname")
	if err != nil {
		t.Fatalf("dirname is required for shell wrapper test: %v", err)
	}
	if err := os.Symlink(dirnamePath, filepath.Join(pathDir, "dirname")); err != nil {
		t.Fatalf("symlink dirname: %v", err)
	}
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Fatalf("sh is required for shell wrapper test: %v", err)
	}
	cmd := exec.CommandContext(context.Background(), shPath, wrapperPath, testSubcommandList)
	cmd.Env = append(os.Environ(), "PATH="+pathDir)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected wrapper to fail without go, got:\n%s", output)
	}
	if !strings.Contains(string(output), "Go is required to run skillhub from a checkout") {
		t.Fatalf("expected Go-required error, got:\n%s", output)
	}
}

func TestSourcesDefaultsPluralCommandUsesGoBackend(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	writeFakeSkillhubRepo(t, repo)
	writeInstallableTestSource(t, sourceDir, "rules-selector", "v1")
	defaults := core.SourcesHeader + "\n" +
		"local-default\tpath\t" + sourceDir + "\t-\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(repo, "defaults", "sources.tsv"), []byte(defaults), 0o644); err != nil {
		t.Fatalf("write defaults: %v", err)
	}
	t.Setenv("SKILLHUB_REPO", repo)
	t.Setenv("SKILLHUB_CONFIG_DIR", configDir)
	t.Setenv("SKILLHUB_CACHE_DIR", cacheDir)

	var out bytes.Buffer
	cmd := cli.NewRootCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{testCommandSources, testDirDefaults, testSubcommandList, testFlagTSV})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("sources defaults list failed: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "local-default\tpath\t"+sourceDir+"\t-\tcatalog/skills.tsv") {
		t.Fatalf("unexpected defaults output:\n%s", got)
	}

	out.Reset()
	cmd = cli.NewRootCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{testCommandSources, testDirDefaults, testArgAdd, "local-default"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("sources defaults add failed: %v", err)
	}
	sources, err := os.ReadFile(filepath.Join(configDir, "sources.tsv"))
	if err != nil {
		t.Fatalf("read user sources: %v", err)
	}
	if !strings.Contains(string(sources), "local-default\tpath\t"+sourceDir+"\t-\tcatalog/skills.tsv") {
		t.Fatalf("expected default source to be added, got:\n%s", sources)
	}
}

func TestTopLevelRestoreChecksLockfileWithGoBackend(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	project := filepath.Join(tmp, "project")
	writeFakeSkillhubRepo(t, repo)
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	t.Setenv("SKILLHUB_REPO", repo)

	cmd := cli.NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"restore", "--project", project, "--check", "--tsv", "-v"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("skillhub restore failed: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "source\tskill\ttarget\tstatus\tinstalled_path\tcontent_hash\treason") {
		t.Fatalf("expected restore check TSV header, got:\n%s", got)
	}
}

func TestSkillsRestoreChecksLockfileWithGoBackend(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	project := filepath.Join(tmp, "project")
	writeFakeSkillhubRepo(t, repo)
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	t.Setenv("SKILLHUB_REPO", repo)

	cmd := cli.NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"skills", "restore", "--project", project, "--check"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("skillhub skills restore failed: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "No project lockfile found: "+filepath.Join(project, "skills.lock.toml")) {
		t.Fatalf("expected no-lockfile restore output, got:\n%s", got)
	}
}

func writeTestSourceCatalog(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "catalog"), 0o755); err != nil {
		t.Fatalf("mkdir source catalog: %v", err)
	}
	catalog := strings.Join([]string{
		"name\tcategory\ttriggers\tdescription",
		"reusable-module-rules\tarchitecture\t" +
			"reusable module,library,public surface,external consumer,release readiness,replace\tReusable rules",
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

func writeCachedGitCatalog(t *testing.T, cacheDir, name, extraSkill string) {
	t.Helper()
	sourceDir := filepath.Join(cacheDir, "sources", name)
	writeTestSourceCatalogWithExtraSkill(t, sourceDir, extraSkill)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func gitIgnored(t *testing.T, dir string, path string) bool {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", "check-ignore", "-q", path)
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "sync", "cached")
	if err != nil {
		t.Fatalf("sources sync failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	firstStamp := readSourceSyncedAt(t, cacheDir, "cached")

	writeTestSourceCatalogWithExtraSkill(t, sourceDir, "synced-rules")
	commitGitSource(t, sourceDir, "add synced rules")

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "sync", "cached")
	if err != nil {
		t.Fatalf("second sources sync failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if got := readSourceSyncedAt(t, cacheDir, "cached"); got < firstStamp {
		t.Fatalf("expected sync timestamp to advance from %d, got %d", firstStamp, got)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "add", sourceDir, "--name", "added", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Added source added") {
		t.Fatalf("expected add output, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	_ = readSourceSyncedAt(t, cacheDir, "added")

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list failed after add: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "added\tadded-rules\t") {
		t.Fatalf("expected initially synced catalog, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestSourcesAddGitHubTreeURLUsesRepositoryAndBranch(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "acton-contracts")
	writeNestedSkill(t, sourceDir, "func2tolk", "FunC to Tolk migration", "Func2tolk")
	writeNestedSkill(t, sourceDir, "tolk", "Tolk contracts", "Tolk")
	runGit(t, sourceDir, "init")
	runGit(t, sourceDir, "checkout", "-b", "main")
	commitGitSource(t, sourceDir, "initial Acton-like skills")
	writeRootSkill(t, sourceDir, ".acton-skill", "Internal Acton metadata", "Internal")
	commitGitSource(t, sourceDir, "add hidden metadata skill")
	runGit(t, sourceDir, "checkout", "-b", "skills")

	gitConfig := filepath.Join(tmp, "gitconfig")
	gitConfigContent := "[url \"file://" + filepath.ToSlash(sourceDir) + "\"]\n" +
		"\tinsteadOf = https://github.com/ton-blockchain/acton-contracts\n"
	if err := os.WriteFile(gitConfig, []byte(gitConfigContent), 0o644); err != nil {
		t.Fatalf("write git config: %v", err)
	}

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
		"GIT_CONFIG_GLOBAL=" + gitConfig,
	}, "sources", "add", "https://github.com/ton-blockchain/acton-contracts/tree/skills/skills/", "--name", "acton")
	if err != nil {
		t.Fatalf("sources add GitHub tree URL failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Added source acton") {
		t.Fatalf("expected add output, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "list", "--tsv")
	if err != nil {
		t.Fatalf("sources list failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "acton\tgit\thttps://github.com/ton-blockchain/acton-contracts\tskills\tcatalog/skills.tsv") {
		t.Fatalf("expected normalized GitHub tree source row, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	catalog, err := os.ReadFile(filepath.Join(cacheDir, "generated-sources", "acton", "catalog", "skills.tsv"))
	if err != nil {
		t.Fatalf("read generated catalog: %v", err)
	}
	if !strings.Contains(string(catalog), "func2tolk\tfunc2tolk\t") || !strings.Contains(string(catalog), "tolk\ttolk\t") {
		t.Fatalf("expected generated catalog to include func2tolk and tolk without false duplicates, got:\n%s", catalog)
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "add", sourceA, "--name", "shared", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("first sources add failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "remove", "shared")
	if err != nil {
		t.Fatalf("sources remove failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "add", sourceB, "--name", "shared", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("second sources add failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "add", sourceDir, "--name", "removable", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "sources", "removable", "catalog", "skills.tsv")); err != nil {
		t.Fatalf("expected source cache after add: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "source-state", "removable.synced_at")); err != nil {
		t.Fatalf("expected source sync state after add: %v", err)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "remove", "removable")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
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
	if err := os.WriteFile(
		filepath.Join(projectDir, "go.mod"),
		[]byte("module example.com/project\n\ngo 1.26.0\n"),
		0o644,
	); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "recommend", "--project", projectDir, "--tsv")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "add", sourceDir, "--name", "nested", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
	if err != nil {
		t.Fatalf("skills list nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(
		stdout,
		"nested\tengineering_tdd\tengineering\tengineering,tdd\tTest-driven development with tracer bullets",
	) {
		t.Fatalf("expected generated engineering_tdd row, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(
		stdout,
		"nested\tproductivity_grill-me\tproductivity\tproductivity,grill-me\tInterview the user relentlessly",
	) {
		t.Fatalf("expected generated productivity_grill-me row, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "search", "--tsv", "tracer")
	if err != nil {
		t.Fatalf("skills search nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "nested\tengineering_tdd\t") {
		t.Fatalf("expected search to find generated nested skill, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "install", "nested/engineering_tdd", "--target", "directory", "--dir", targetDir)
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "add", sourceDir, "--name", "rooted", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add root skills failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "search", "--tsv", "red-green")
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
		{"sources", "add", sourceDir, "--name", "nested", "--type", "git", "--ref", "main"},
		{"skills", "install", "nested/engineering_tdd", "--target", "directory", "--dir", targetDir},
	} {
		stdout, stderr, err := runCLISplitForTest(t, []string{
			"SKILLHUB_CONFIG_DIR=" + configDir,
			"SKILLHUB_CACHE_DIR=" + cacheDir,
		}, args[0], args[1:]...)
		if err != nil {
			t.Fatalf("%s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout, stderr)
		}
	}

	writeNestedSkill(t, sourceDir, "engineering/tdd", "Test-driven development with tracer bullets", "TDD v2")
	commitGitSource(t, sourceDir, "update tdd")

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "installed", "update", "--target", "directory", "--dir", targetDir, "-v")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "add", sourceDir, "--name", "nested", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	writeNestedSkill(t, sourceDir, "engineering/diagnose", "Disciplined diagnosis loop", "Diagnose v1")
	commitGitSource(t, sourceDir, "add diagnose")

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "sync", "nested")
	if err != nil {
		t.Fatalf("sources sync nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "skills", "list", "--tsv")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "add", sourceDir, "--name", "nested", "--type", "git", "--ref", "main")
	if err != nil {
		t.Fatalf("sources add nested failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "generated-sources", "nested", "catalog", "skills.tsv")); err != nil {
		t.Fatalf("expected generated catalog after add: %v", err)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "remove", "nested")
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

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "add", sourceDir, "--name", "dups", "--type", "path")
	if err == nil {
		t.Fatalf("expected duplicate generated skill names to fail, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "duplicate generated skill name a_b_c") {
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
	if err := os.WriteFile(
		filepath.Join(projectDir, "go.mod"),
		[]byte("module example.com/project\n\ngo 1.26.0\n"),
		0o644,
	); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	stdout, stderr, err := runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "sources", "add", sourceDir, "--name", "nested", "--type", "path")
	if err != nil {
		t.Fatalf("sources add nested path failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	stdout, stderr, err = runCLISplitForTest(t, []string{
		"SKILLHUB_CONFIG_DIR=" + configDir,
		"SKILLHUB_CACHE_DIR=" + cacheDir,
	}, "recommend", "--project", projectDir, "--tsv")
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

	output, err := runCLIWithConfig(t, configDir, "recommend", "--project", projectDir, "--tsv")
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

	output, err := runCLIWithConfig(t, configDir, "recommend", "--project", projectDir)
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
		core.InstalledUsageHeader,
		installedUsageFixtureRow("local", "rules-selector", "codex", projectA, tmp),
		installedUsageFixtureRow("local", "rules-selector", "claude", projectB, tmp),
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(configDir, "installed.tsv"), []byte(registry), 0o644); err != nil {
		t.Fatalf("write installed registry: %v", err)
	}

	output, err := runCLIWithConfig(
		t, configDir, "installed", "usage", "update", "--projects",
		"--target", "codex", "--project", projectA, "rules-selector", "-v")
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

	output, err := runCLIWithConfig(
		t, configDir, "skills", "install", "rules-selector",
		"--target", "codex", "--scope", "project", "--project", projectDir)
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

func TestProjectInstallWritesPortableLockfile(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeInstallableTestSource(t, sourceDir, "rules-selector", "v1")
	writeTestSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	output, err := runCLIWithConfig(
		t, configDir, "skills", "install", "rules-selector",
		"--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}

	lockfile, err := os.ReadFile(filepath.Join(projectDir, "skills.lock.toml"))
	if err != nil {
		t.Fatalf("expected project lockfile: %v", err)
	}
	lock := string(lockfile)
	for _, want := range []string{
		"lockfile_version = 1",
		"[[skills]]",
		`source = "local"`,
		`source_type = "path"`,
		`skill = "rules-selector"`,
		`qualified_skill = "local/rules-selector"`,
		`target = "codex"`,
		`scope = "project"`,
		`project_path = "."`,
		`target_root = ".agents/skills"`,
		`installed_path = ".agents/skills/rules-selector"`,
		`source_location = "` + sourceDir + `"`,
		`catalog = "catalog/skills.tsv"`,
		`content_hash = "`,
		`installed_at = "`,
		`updated_at = "`,
	} {
		if !strings.Contains(lock, want) {
			t.Fatalf("expected lockfile to contain %q, got:\n%s", want, lock)
		}
	}
	if strings.Contains(lock, projectDir) {
		t.Fatalf("lockfile must not contain absolute project path %q:\n%s", projectDir, lock)
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

	output, err := runCLIForTest(
		t,
		[]string{"SKILLHUB_CONFIG_DIR=" + configDir, "AGENT_SKILLS_DIR=" + globalDir},
		"skills",
		"install",
		"rules-selector",
	)
	if err != nil {
		t.Fatalf("global install failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(globalDir, "rules-selector", ".skillhub.json")); err != nil {
		t.Fatalf("expected global install sidecar metadata: %v", err)
	}

	output, err = runCLIWithConfig(
		t, configDir, "skills", "install", "rules-selector",
		"--target", "directory", "--dir", directoryDir)
	if err != nil {
		t.Fatalf("directory install failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(directoryDir, "rules-selector", ".skillhub.json")); err != nil {
		t.Fatalf("expected directory install sidecar metadata: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "skills.lock.toml")); !os.IsNotExist(err) {
		t.Fatalf("global/directory install should not create project lockfile, stat err=%v", err)
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

	output, err := runCLIWithConfig(
		t, configDir, "skills", "install", "rules-selector",
		"--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}
	installedDir := filepath.Join(projectDir, ".agents", "skills", "rules-selector")
	staleMetadata := filepath.Join(installedDir, ".skillhub.json")
	if err := os.WriteFile(staleMetadata, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write stale sidecar metadata: %v", err)
	}

	writeInstallableTestSource(t, sourceDir, "rules-selector", "v2")
	output, err = runCLIWithConfig(
		t, configDir, "installed", "usage", "update", "--projects",
		"--target", "codex", "--project", projectDir, "rules-selector", "-v")
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
	lockfile, err := os.ReadFile(filepath.Join(projectDir, "skills.lock.toml"))
	if err != nil {
		t.Fatalf("expected project lockfile after update: %v", err)
	}
	if !strings.Contains(string(lockfile), `source = "local"`) ||
		!strings.Contains(string(lockfile), `installed_path = ".agents/skills/rules-selector"`) {
		t.Fatalf("expected update to refresh project lockfile, got:\n%s", lockfile)
	}
}

func TestProjectUninstallRemovesLockfileWhenEmpty(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeInstallableTestSource(t, sourceDir, "rules-selector", "v1")
	writeTestSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	output, err := runCLIWithConfig(
		t, configDir, "skills", "install", "rules-selector",
		"--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}
	output, err = runCLIWithConfig(
		t, configDir, "installed", "uninstall", "rules-selector",
		"--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project uninstall failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "skills.lock.toml")); !os.IsNotExist(err) {
		t.Fatalf("expected empty project lockfile to be removed, stat err=%v", err)
	}
}

func TestRestoreInstallsMissingProjectSkillFromLockfileWithoutConfiguredSources(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	restoreConfigDir := filepath.Join(tmp, "restore-config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeInstallableTestSource(t, sourceDir, "rules-selector", "v1")
	writeTestSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	output, err := runCLIWithConfig(
		t, configDir, "skills", "install", "rules-selector",
		"--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}
	installedDir := filepath.Join(projectDir, ".agents", "skills", "rules-selector")
	if err := os.RemoveAll(installedDir); err != nil {
		t.Fatalf("remove installed skill: %v", err)
	}
	if err := os.MkdirAll(restoreConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir restore config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(restoreConfigDir, "sources.tsv"), []byte("broken\n"), 0o644); err != nil {
		t.Fatalf("write malformed restore sources: %v", err)
	}

	output, err = runCLIWithConfig(
		t, restoreConfigDir, "skills", "restore", "--project", projectDir, "-v")
	if err != nil {
		t.Fatalf("restore failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "Restored project skills: installed=1 updated=0 unchanged=0 skipped=0 failed=0") {
		t.Fatalf("expected restore summary, got:\n%s", output)
	}
	content, err := os.ReadFile(filepath.Join(installedDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("expected restored skill content: %v", err)
	}
	if !strings.Contains(string(content), "v1") {
		t.Fatalf("expected restored content from lockfile source, got:\n%s", content)
	}
	registry, err := os.ReadFile(filepath.Join(restoreConfigDir, "installed.tsv"))
	if err != nil {
		t.Fatalf("expected restore to write installed registry: %v", err)
	}
	if !strings.Contains(string(registry), "local\trules-selector\tcodex\tproject\t"+projectDir) {
		t.Fatalf("expected restore registry row with absolute project path, got:\n%s", registry)
	}
	sources, err := os.ReadFile(filepath.Join(restoreConfigDir, "sources.tsv"))
	if err != nil {
		t.Fatalf("read restore sources: %v", err)
	}
	if string(sources) != "broken\n" {
		t.Fatalf("restore must not mutate user sources, got:\n%s", sources)
	}
}

func TestRestoreCheckTSVReportsStatusesWithoutMutation(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	projectDir := filepath.Join(tmp, "project")
	writeInstallableTestSource(t, sourceDir, "rules-selector", "v1")
	writeTestSources(t, configDir, sourceDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	output, err := runCLIWithConfig(
		t, configDir, "skills", "install", "rules-selector",
		"--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}
	skillFile := filepath.Join(projectDir, ".agents", "skills", "rules-selector", "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("# local edit\n"), 0o644); err != nil {
		t.Fatalf("write local edit: %v", err)
	}

	output, err = runCLIWithConfig(
		t, configDir, "skills", "restore", "--check", "--tsv", "--project", projectDir)
	if err != nil {
		t.Fatalf("restore check failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "source\tskill\ttarget\tstatus\tinstalled_path\tcontent_hash\treason") {
		t.Fatalf("expected restore check TSV header, got:\n%s", output)
	}
	if !strings.Contains(output, "local\trules-selector\tcodex\tchanged\t.agents/skills/rules-selector\t") {
		t.Fatalf("expected changed status, got:\n%s", output)
	}
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("read local edit: %v", err)
	}
	if string(content) != "# local edit\n" {
		t.Fatalf("restore --check must not mutate installed skill, got:\n%s", content)
	}

	lockPath := filepath.Join(projectDir, "skills.lock.toml")
	lockfile, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lockfile: %v", err)
	}
	broken := strings.ReplaceAll(
		string(lockfile),
		`source_location = "`+sourceDir+`"`,
		`source_location = "`+filepath.Join(tmp, "missing-source")+`"`,
	)
	overwriteFile(t, lockPath, broken)
	output, err = runCLIWithConfig(
		t, configDir, "skills", "restore", "--check", "--tsv", "--project", projectDir)
	if err != nil {
		t.Fatalf("restore check with missing source should not fail: %v\n%s", err, output)
	}
	if !strings.Contains(output, "local\trules-selector\tcodex\tskipped\t.agents/skills/rules-selector\t") {
		t.Fatalf("expected skipped status for missing lockfile source, got:\n%s", output)
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

	output, err := runCLIWithConfig(
		t, configDir, "skills", "install", "rules-selector",
		"--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}

	output, err = runCLIWithConfig(
		t, configDir, "installed", "list",
		"--target", "codex", "--scope", "project", "--project", projectDir, "--tsv")
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

	output, err := runCLIWithConfig(
		t, configDir, "skills", "install", "rules-selector",
		"--target", "codex", "--scope", "project", "--project", projectDir)
	if err != nil {
		t.Fatalf("project install failed: %v\n%s", err, output)
	}

	output, err = runCLIWithConfig(
		t, configDir, "targets", "detect", "--tsv", "--project", projectDir)
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
	sidecar := []byte(`{"source":"local","skill":"rules-selector"}` + "\n")
	if err := os.WriteFile(filepath.Join(installedDir, ".skillhub.json"), sidecar, 0o644); err != nil {
		t.Fatalf("write stale sidecar: %v", err)
	}

	output, err := runCLIWithConfig(
		t, configDir, "installed", "list",
		"--target", "codex", "--scope", "project", "--project", projectDir, "--tsv")
	if err != nil {
		t.Fatalf("installed list failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "codex\tproject\trules-selector\tno\t-\t-") {
		t.Fatalf("expected stale project sidecar without registry to be unmanaged, got:\n%s", output)
	}

	output, err = runCLIWithConfig(
		t, configDir, "targets", "detect", "--tsv", "--project", projectDir)
	if err != nil {
		t.Fatalf("targets detect failed: %v\n%s", err, output)
	}
	want := "codex\tproject\tsupported\t" + filepath.Join(projectDir, ".agents", "skills") + "\tyes\t1\t0"
	if !strings.Contains(output, want) {
		t.Fatalf("expected targets detect to treat stale project sidecar as unmanaged %q, got:\n%s", want, output)
	}

	output, err = runCLIWithConfig(
		t, configDir, "installed", "uninstall", "rules-selector",
		"--target", "codex", "--scope", "project", "--project", projectDir)
	if err == nil {
		t.Fatalf("expected uninstall to refuse stale project sidecar without registry, got:\n%s", output)
	}
	if !strings.Contains(output, "refusing to uninstall unmanaged skill") {
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
	sidecar := []byte(`{"source":"local","skill":"rules-selector"}` + "\n")
	if err := os.WriteFile(filepath.Join(installedDir, ".skillhub.json"), sidecar, 0o644); err != nil {
		t.Fatalf("write stale sidecar: %v", err)
	}

	output, err := runCLIWithConfig(
		t, configDir, "installed", "update",
		"--target", "codex", "--scope", "project", "--project", projectDir, "-v")
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
