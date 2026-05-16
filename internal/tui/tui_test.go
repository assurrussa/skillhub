package tui_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/assurrussa/skillhub/internal/core"
	tui "github.com/assurrussa/skillhub/internal/tui"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

const (
	testSkillGoProjectRules  = "go-project-rules"
	testSkillRulesSelector   = "rules-selector"
	testSkillDocsProject     = "docs-project-rules"
	testSkillManual          = "manual-skill"
	testSkillActon           = "acton"
	testSkillGrillMe         = "productivity_grill-me"
	testSourceAgentRules     = "agent-rules"
	testSourceMattPocock     = "mattpocock"
	testSourceActon          = "acton"
	testSourceAlpha          = "alpha"
	testSourceBeta           = "beta"
	testProjectPath          = "/tmp/project"
	testProjectA             = "/tmp/project-a"
	testProjectB             = "/tmp/project-b"
	testProjectTongoldy      = "/tmp/tongoldy"
	testProjectSkillsRoot    = "/tmp/project/.agents/skills"
	testSkillsRoot           = "/tmp/skills"
	testTimestampInstalled   = "2026-05-05T00:00:00Z"
	testTimestampUsage       = "2026-05-05T01:00:00Z"
	testTimestampLatest      = "2026-05-05T03:00:00Z"
	testDescGoProjectRules   = "Go project rules"
	testDescGoRules          = "Go rules"
	testDescGrillMe          = "Grill me"
	testDocsCategory         = "documentation"
	testDocsTrigger          = "docs"
	testQualifiedGoRules     = "agent-rules/go-project-rules"
	testQualifiedRules       = "agent-rules/rules-selector"
	testQualifiedGrillMe     = "mattpocock/productivity_grill-me"
	testGoRulesPath          = "/tmp/skills/go-project-rules"
	testManualSkillPath      = "/tmp/project/.claude/skills/manual-skill"
	testProjectBGeminiPath   = "/tmp/project-b/.gemini/skills/rules-selector"
	testGrillMeClaudePath    = "/home/me/.claude/skills/productivity_grill-me"
	testLabelCodexGlobal     = "Codex global"
	testLabelClaudeGlobal    = "Claude global"
	testFlagTarget           = "--target"
	testCommandAdd           = "add"
	testCodexProjectKey      = "codex:project"
	testCodexGlobalKey       = "codex:global"
	testClaudeGlobalKey      = "claude:global"
	testCodexSkillsDesc      = "Codex skills"
	testClaudeSkillsDesc     = "Claude skills"
	testTargetOpenCode       = "opencode"
	testLabelCursor          = "Cursor"
	testAdapterSkillDir      = "skill-dir"
	testStatusSupported      = "supported"
	testStatusPlanned        = "planned"
	testDescPlanned          = "Planned"
	testBulletGo             = "• go"
	testHashABC              = "abc"
	testProjectRulesPath     = "/tmp/project/.agents/skills/rules-selector"
	testGlobalRulesPath      = "/tmp/global/rules-selector"
	testRecordedProject      = "/tmp/recorded-project"
	testLegacyRulesPath      = "/tmp/legacy-skills/rules-selector"
	testSourceURL            = "https://github.com/mattpocock/skills"
	testTriggerGrill         = "grill"
	testTriggerGoGolang      = "go,golang"
	testCategoryProductivity = "productivity"
	testSourceTypeGit        = "git"
	testSourceRefMain        = "main"
	testCatalogSkillsPath    = "catalog/skills.tsv"
	testQueueStateRunning    = "running"
	testQueueStatePending    = "pending"
	testPermissionDeniedLine = "permission denied\n"
	testTargetCursor         = "cursor"
	testLabelOpenCode        = "OpenCode"
	testGeminiSkillsDesc     = "Gemini skills"
	testOpenCodeSkillsDesc   = "OpenCode skills"
	commandUpdate            = "update"
)

func stripANSI(value string) string {
	return ansiRE.ReplaceAllString(value, "")
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func asModel(t *testing.T, tm tea.Model) tui.TestModel {
	t.Helper()
	m, ok := tm.(tui.TestModel)
	if !ok {
		t.Fatalf("expected tui.TestModel, got %T", tm)
	}
	return m
}

func actonInstalledRows() []tui.InstalledSkill {
	return []tui.InstalledSkill{
		{
			Target: tui.TargetGemini, Scope: tui.ScopeGlobal, Skill: "productivity_caveman",
			Managed: tui.ManagedYes, Source: testSourceMattPocock, Path: "/tmp/gemini/productivity_caveman",
		},
		{
			Target: tui.TargetCodex, Scope: tui.ScopeProject, Skill: testSourceActon, QualifiedSkill: "acton/acton",
			Managed: tui.ManagedYes, Source: testSourceActon, ProjectPath: testProjectTongoldy,
			Path: "/tmp/tongoldy/.agents/skills/acton",
		},
		{
			Target: tui.TargetCodex, Scope: tui.ScopeProject, Skill: "func2tolk", QualifiedSkill: "acton/func2tolk",
			Managed: tui.ManagedYes, Source: testSourceActon, ProjectPath: testProjectTongoldy,
			Path: "/tmp/tongoldy/.agents/skills/func2tolk",
		},
		{
			Target: tui.TargetCodex, Scope: tui.ScopeProject, Skill: "tolk", QualifiedSkill: "acton/tolk",
			Managed: tui.ManagedYes, Source: testSourceActon, ProjectPath: testProjectTongoldy,
			Path: "/tmp/tongoldy/.agents/skills/tolk",
		},
		{
			Target: tui.TargetCodex, Scope: tui.ScopeProject, Skill: "ton-blockchain",
			QualifiedSkill: "acton/ton-blockchain", Managed: tui.ManagedYes, Source: testSourceActon,
			ProjectPath: testProjectTongoldy, Path: "/tmp/tongoldy/.agents/skills/ton-blockchain",
		},
	}
}

func testManagedInstalled(path string) tui.InstalledSkill {
	return tui.InstalledSkill{
		Target: tui.TargetCodex, Scope: tui.ScopeGlobal, Skill: testSkillRulesSelector,
		Managed: tui.ManagedYes, Source: testSourceAgentRules, Path: path,
	}
}

func testTargetChoice(key, target, label, scope, path string) tui.InstallTargetChoice {
	return tui.InstallTargetChoice{
		Key: key, Target: target, Label: label,
		Scope: scope, Status: tui.TargetStatusSupported, Path: path, Supported: true,
	}
}

func testSupportedTarget(id, label, description string) tui.Target {
	return tui.Target{
		ID: id, Label: label, Status: testStatusSupported,
		Adapter: testAdapterSkillDir, Description: description,
	}
}

func testPlannedTarget(id, label string) tui.Target {
	return tui.Target{
		ID: id, Label: label, Status: testStatusPlanned,
		Adapter: testStatusPlanned, Description: testDescPlanned,
	}
}

func writeTUIGitSourceConfig(t *testing.T, configDir, cacheDir, name string, syncedAt time.Time) {
	t.Helper()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	sources := "name\ttype\tlocation\tref\tcatalog\n" +
		name + "\tgit\t" + filepath.Join(configDir, "missing-remote") + "\tmain\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(sources), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}

	catalogDir := filepath.Join(cacheDir, "sources", name, "catalog")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatalf("mkdir catalog: %v", err)
	}
	catalog := "name\tcategory\ttriggers\tdescription\n" +
		testSkillGoProjectRules + "\tgo\tgo,golang,go.mod\tGo rules\n"
	if err := os.WriteFile(filepath.Join(catalogDir, "skills.tsv"), []byte(catalog), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	stateDir := filepath.Join(cacheDir, "source-state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir source state: %v", err)
	}
	state := strconv.FormatInt(syncedAt.Unix(), 10) + "\n"
	if err := os.WriteFile(filepath.Join(stateDir, name+".synced_at"), []byte(state), 0o644); err != nil {
		t.Fatalf("write source state: %v", err)
	}
}

func writeTUINestedPathSource(t *testing.T, configDir, sourceDir string) {
	t.Helper()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	sources := "name\ttype\tlocation\tref\tcatalog\n" +
		"nested\tpath\t" + sourceDir + "\t-\tcatalog/skills.tsv\n"
	if err := os.WriteFile(filepath.Join(configDir, "sources.tsv"), []byte(sources), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
	skillDir := filepath.Join(sourceDir, "skills", "engineering", "tdd")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir nested skill: %v", err)
	}
	content := "---\nname: tdd\ndescription: Test-driven development\n---\n\n# TDD\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write nested skill: %v", err)
	}
}

func writeFakeTUIRepo(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{"defaults", "targets"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir fake repo dir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(
		filepath.Join(root, "defaults", "sources.tsv"),
		[]byte("name\ttype\tlocation\tref\tcatalog\n"),
		0o644,
	); err != nil {
		t.Fatalf("write fake defaults: %v", err)
	}
	targets := "id\tlabel\tstatus\tadapter\tdescription\n" +
		"codex\tCodex\tsupported\tskill-dir\tCodex skills\n" +
		"claude\tClaude\tsupported\tskill-dir\tClaude skills\n"
	if err := os.WriteFile(filepath.Join(root, "targets", "targets.tsv"), []byte(targets), 0o644); err != nil {
		t.Fatalf("write fake targets: %v", err)
	}
}

func TestParseSkillsTSV(t *testing.T) {
	input := "source\tname\tcategory\ttriggers\tdescription\n" +
		testSourceAgentRules + "\t" + testSkillGoProjectRules + "\tgo\tgo,golang\tGo project rules\n"

	skills, err := tui.ParseSkillsTSV(input)
	if err != nil {
		t.Fatalf("tui.ParseSkillsTSV returned error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != testSkillGoProjectRules || skills[0].Category != "go" {
		t.Fatalf("unexpected skill: %#v", skills[0])
	}
}

func TestLoadSkillsUsesGoBackend(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	sourceDir := filepath.Join(tmp, "source")
	writeFakeTUIRepo(t, repo)
	writeTUINestedPathSource(t, configDir, sourceDir)
	t.Setenv("SKILLHUB_CONFIG_DIR", configDir)
	t.Setenv("SKILLHUB_CACHE_DIR", filepath.Join(tmp, "cache"))

	msg := tui.LoadSkills(repo)()
	loaded, ok := msg.(tui.SkillsLoadedMsg)
	if !ok {
		t.Fatalf("expected tui.SkillsLoadedMsg, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("tui.LoadSkills should use Go backend, got error: %v", loaded.Err)
	}
	if len(loaded.Skills) != 1 || loaded.Skills[0].Name != "engineering_tdd" {
		t.Fatalf("unexpected loaded Skills: %#v", loaded.Skills)
	}
}

func TestSkillsLoadedNoSourcesFromGoBackendShowsPresetPrompt(t *testing.T) {
	model := tui.InitialModel(testRepoRoot(t))
	model.Loading = true
	model.LoadingSkills = true
	model.ViewMode = tui.ViewSkills

	updated, _ := model.Update(tui.SkillsLoadedMsg{
		Err: errors.New("no sources configured. Run: skillhub sources defaults list"),
	})
	got, ok := updated.(tui.TestModel)
	if !ok {
		t.Fatalf("expected tui.TestModel, got %T", updated)
	}
	if !got.NoSources {
		t.Fatalf("expected no-sources state")
	}
	if got.Loading || got.LoadingSkills {
		t.Fatalf("expected loading flags to clear, loading=%v loadingSkills=%v", got.Loading, got.LoadingSkills)
	}
	if !strings.Contains(got.Status, "Press d for presets") {
		t.Fatalf("expected preset prompt status, got %q", got.Status)
	}
}

func TestLoadSkillsUsesCachedCatalog(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	writeTUIGitSourceConfig(t, configDir, cacheDir, "cached", time.Now())
	t.Setenv("SKILLHUB_CONFIG_DIR", configDir)
	t.Setenv("SKILLHUB_CACHE_DIR", cacheDir)

	msg := tui.LoadSkills(testRepoRoot(t))()
	loaded, ok := msg.(tui.SkillsLoadedMsg)
	if !ok {
		t.Fatalf("expected tui.SkillsLoadedMsg, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("tui.LoadSkills returned error: %v", loaded.Err)
	}
	if len(loaded.Skills) != 1 || loaded.Skills[0].Name != testSkillGoProjectRules {
		t.Fatalf("unexpected loaded Skills: %#v", loaded.Skills)
	}
}

func TestLoadSkillsShowsStaleCacheWarningInStatus(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	writeTUIGitSourceConfig(t, configDir, cacheDir, "cached", time.Now().Add(-11*time.Minute))
	t.Setenv("SKILLHUB_CONFIG_DIR", configDir)
	t.Setenv("SKILLHUB_CACHE_DIR", cacheDir)

	msg := tui.LoadSkills(testRepoRoot(t))()
	loaded, ok := msg.(tui.SkillsLoadedMsg)
	if !ok {
		t.Fatalf("expected tui.SkillsLoadedMsg, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("tui.LoadSkills should use stale cache: %v", loaded.Err)
	}

	m := tui.InitialModel(testRepoRoot(t))
	m.ViewMode = tui.ViewSkills
	updated, _ := m.Update(loaded)
	m = asModel(t, updated)
	if !strings.Contains(m.Status, "stale cache") {
		t.Fatalf("expected stale cache warning in status, got %q", m.Status)
	}
}

func TestLoadSkillsUsesGeneratedNestedCatalog(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	writeTUINestedPathSource(t, configDir, sourceDir)
	t.Setenv("SKILLHUB_CONFIG_DIR", configDir)
	t.Setenv("SKILLHUB_CACHE_DIR", cacheDir)

	msg := tui.LoadSkills(testRepoRoot(t))()
	loaded, ok := msg.(tui.SkillsLoadedMsg)
	if !ok {
		t.Fatalf("expected tui.SkillsLoadedMsg, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("tui.LoadSkills returned error: %v", loaded.Err)
	}
	if len(loaded.Skills) != 1 ||
		loaded.Skills[0].Name != "engineering_tdd" ||
		loaded.Skills[0].Category != "engineering" {
		t.Fatalf("unexpected generated nested Skills: %#v", loaded.Skills)
	}
}

func TestApplyFilterMatchesDescriptionAndTriggers(t *testing.T) {
	m := tui.InitialModel(".")
	m.Skills = []tui.Skill{
		{
			Source: testSourceAgentRules, Name: testSkillGoProjectRules,
			Category: "go", Triggers: testTriggerGoGolang,
			Description: testDescGoProjectRules,
		},
		{
			Source:      testSourceAgentRules,
			Name:        testSkillDocsProject,
			Category:    testDocsCategory,
			Triggers:    testDocsTrigger,
			Description: "Documentation rules",
		},
	}
	m.Search = "golang"
	m.ApplyFilter()

	if len(m.Filtered) != 1 {
		t.Fatalf("expected 1 filtered skill, got %d", len(m.Filtered))
	}
	if m.Skills[m.Filtered[0]].Name != testSkillGoProjectRules {
		t.Fatalf("unexpected filtered skill: %#v", m.Skills[m.Filtered[0]])
	}
}

func TestSelectionUsesQualifiedSkillNames(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Skills = []tui.Skill{
		{Source: testSourceAlpha, Name: "same-name", Category: "go", Description: "Alpha"},
		{Source: testSourceBeta, Name: "same-name", Category: "go", Description: "Beta"},
	}
	m.ApplyFilter()

	m.ToggleCurrent()
	m.Cursor = 1
	m.ToggleCurrent()

	got := m.SelectedNames()
	want := []string{"alpha/same-name", "beta/same-name"}
	if len(got) != len(want) {
		t.Fatalf("expected %d selected names, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("selectedNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestViewShowsSourceColumn(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Width = 120
	m.Height = 20
	m.Skills = []tui.Skill{
		{Source: testSourceAgentRules, Name: testSkillGoProjectRules, Category: "go", Description: testDescGoProjectRules},
	}
	m.ApplyFilter()

	view := m.View()
	if !strings.Contains(view, testSourceAgentRules) {
		t.Fatalf("expected view to include source name, got:\n%s", view)
	}
}

func TestViewShowsTaskOrientedSkillList(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Width = 120
	m.Height = 30
	m.Skills = []tui.Skill{
		{
			Source:      testSourceAgentRules,
			Name:        testSkillGoProjectRules,
			Category:    "go",
			Triggers:    "go,golang,go.mod",
			Description: "Global Go project rules for architecture and contracts.",
		},
	}
	m.ApplyFilter()

	view := stripANSI(m.View())
	for _, want := range []string{
		"Skillhub",
		"Sources: 1",
		"Skills",
		"[ ] go-project-rules",
		"Source agent-rules",
		testBulletGo,
		"Global Go project rules for",
		"architecture and contracts.",
		"d presets",
		"n source",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q, got:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Preview") {
		t.Fatalf("default selector should not render the old preview split, got:\n%s", view)
	}
}

func TestSourcesViewShowsStatusSummaryAndCursor(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSources
	m.Width = 120
	m.Height = 30
	m.Sources = []tui.SourcePreset{
		{
			Name:         testSourceAgentRules,
			Type:         testSourceTypeGit,
			Status:       "fresh",
			LastSyncedAt: "2026-05-15T10:00:00Z",
			CachePath:    "/tmp/cache/sources/agent-rules",
			Ref:          testSourceRefMain,
			Catalog:      testCatalogSkillsPath,
			Location:     "git@example.com:agent-rules.git",
			Message:      "cache is fresh",
		},
		{
			Name:     testSourceMattPocock,
			Type:     testSourceTypeGit,
			Status:   core.SourceStatusMissing,
			Ref:      testSourceRefMain,
			Catalog:  testCatalogSkillsPath,
			Location: "git@example.com:mattpocock.git",
			Message:  "cache missing. Run: skillhub sources sync mattpocock",
		},
	}
	m.SourceCursor = 1

	view := stripANSI(m.View())
	for _, want := range []string{
		"Sources: 2",
		"fresh: 1",
		"missing: 1",
		"agent-rules [fresh]",
		"last sync: 2026-05-15T10:00:00Z",
		"mattpocock [missing]",
		"skillhub sources sync mattpocock",
		"u update source",
		"U/s update all",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected sources view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestSourcesCursorMovesWithJK(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSources
	m.Sources = []tui.SourcePreset{
		{Name: testSourceAgentRules},
		{Name: testSourceMattPocock},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = asModel(t, updated)
	if m.SourceCursor != 1 {
		t.Fatalf("expected source cursor to move down, got %d", m.SourceCursor)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = asModel(t, updated)
	if m.SourceCursor != 0 {
		t.Fatalf("expected source cursor to move up, got %d", m.SourceCursor)
	}
}

func TestSourcesUpdateCurrentStartsOneStepProgress(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSources
	m.Sources = []tui.SourcePreset{
		{Name: testSourceAgentRules},
		{Name: testSourceMattPocock},
	}
	m.SourceCursor = 1

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = asModel(t, updated)
	if cmd == nil || !m.Busy || m.SourceProgress.Total != 1 {
		t.Fatalf("expected one-step source update progress, busy=%v progress=%#v cmd=%v", m.Busy, m.SourceProgress, cmd)
	}
	if m.SourceProgress.Items[0].Name != testSourceMattPocock {
		t.Fatalf("expected highlighted source update, got %#v", m.SourceProgress)
	}
}

func TestSourcesUpdateAllStartsProgress(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSources
	m.Sources = []tui.SourcePreset{
		{Name: testSourceAgentRules},
		{Name: testSourceMattPocock},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("U")})
	m = asModel(t, updated)
	if cmd == nil || !m.Busy || m.SourceProgress.Total != 2 {
		t.Fatalf("expected all-source update progress, busy=%v progress=%#v cmd=%v", m.Busy, m.SourceProgress, cmd)
	}
	view := stripANSI(m.View())
	for _, want := range []string{
		"Source update progress",
		"Updating 1/2",
		"Queue",
		testQueueStateRunning,
		testQueueStatePending,
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected source update progress view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestSourceProgressAdvancesAndStopsOnFailure(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSources
	m.Sources = []tui.SourcePreset{
		{Name: testSourceAgentRules},
		{Name: testSourceMattPocock},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = asModel(t, updated)
	updated, _ = m.Update(tui.SourceSyncStepDoneMsg{Output: "Synced agent-rules\n"})
	m = asModel(t, updated)
	if m.SourceProgress.Completed != 1 || m.SourceProgress.Current != 2 || !m.Busy {
		t.Fatalf("expected source progress 2/2 after first step, busy=%v progress=%#v", m.Busy, m.SourceProgress)
	}

	updated, _ = m.Update(tui.SourceSyncStepDoneMsg{Output: testPermissionDeniedLine, Err: errors.New("exit status 1")})
	m = asModel(t, updated)
	if m.Busy || !m.SourceProgress.Failed {
		t.Fatalf("expected failed source progress to stop busy state, busy=%v progress=%#v", m.Busy, m.SourceProgress)
	}
	view := stripANSI(m.View())
	for _, want := range []string{"Source update failed", "Updating 2/2", "permission denied", "exit status 1"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected failed source progress view to contain %q, got:\n%s", want, view)
		}
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	if m.SourceProgress.Failed || m.SourceProgress.Total != 0 || m.ViewMode != tui.ViewSources {
		t.Fatalf("expected failed source progress to clear in sources, got view=%q progress=%#v", m.ViewMode, m.SourceProgress)
	}
	if !strings.Contains(m.Status, "exit status 1") {
		t.Fatalf("expected failure message to remain in status, got %q", m.Status)
	}
}

func TestSkillsScreenGroupsBySourceThenCategory(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Width = 140
	m.Height = 50
	m.Skills = []tui.Skill{
		{Source: testSourceBeta, Name: "zeta", Category: "go", Triggers: "go", Description: "Beta Go skill"},
		{
			Source: testSourceAlpha, Name: testDocsTrigger,
			Category: testDocsCategory, Triggers: testDocsTrigger,
			Description: "Alpha docs skill",
		},
		{Source: testSourceAlpha, Name: "go-project", Category: "go", Triggers: "go", Description: "Alpha Go skill"},
	}
	m.ApplyFilter()

	view := stripANSI(m.View())
	for _, want := range []string{
		"Source alpha",
		"• documentation",
		"[ ] docs",
		testBulletGo,
		"[ ] go-project",
		"Source beta",
		"[ ] zeta",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected source-grouped skills view to contain %q, got:\n%s", want, view)
		}
	}
	alphaIndex := strings.Index(view, "Source alpha")
	betaIndex := strings.Index(view, "Source beta")
	docsIndex := strings.Index(view, "[ ] docs")
	goIndex := strings.Index(view, "[ ] go-project")
	if alphaIndex < 0 || betaIndex < 0 || alphaIndex > betaIndex {
		t.Fatalf("expected alpha source group before beta, got:\n%s", view)
	}
	if docsIndex < 0 || goIndex < 0 || docsIndex > goIndex {
		t.Fatalf("expected categories to sort inside alpha source, got:\n%s", view)
	}

	m.Search = testSourceBeta
	m.ApplyFilter()
	view = stripANSI(m.View())
	if !strings.Contains(view, "Source beta") || strings.Contains(view, "Source alpha") {
		t.Fatalf("expected filtered skills to preserve only matching source group, got:\n%s", view)
	}
}

func TestDashboardRendersSectionNavigation(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 30
	m.Skills = []tui.Skill{
		{Source: testSourceAgentRules, Name: testSkillGoProjectRules, Category: "go", Description: testDescGoRules},
	}
	m.ApplyFilter()

	view := stripANSI(m.View())
	for _, want := range []string{
		"1 Installed",
		"2 Skills",
		"3 Sources",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected dashboard navigation to contain %q, got:\n%s", want, view)
		}
	}
	if strings.Contains(view, "4 Targets") || strings.Contains(view, "5 Update") || strings.Contains(view, "Usage") {
		t.Fatalf("expected targets/update dashboard sections to be removed, got:\n%s", view)
	}
}

func TestInitialModelDefaultsToInstalled(t *testing.T) {
	m := tui.InitialModel(".")
	if m.ViewMode != tui.ViewInstalled {
		t.Fatalf("expected default view to be installed, got %q", m.ViewMode)
	}
	if m.Status != "Loading installed skills..." {
		t.Fatalf("expected installed loading status, got %q", m.Status)
	}
}

func TestInitialCatalogLoadDoesNotClearInstalledLoading(t *testing.T) {
	m := tui.InitialModel(".")
	updated, _ := m.Update(tui.SkillsLoadedMsg{Skills: []tui.Skill{
		{Source: testSourceAgentRules, Name: testSkillGoProjectRules, Category: "go", Description: testDescGoRules},
	}})
	m = asModel(t, updated)
	if !m.Loading {
		t.Fatalf("expected installed-first screen to keep loading until installed rows load")
	}
	if m.Status != "Loading installed skills..." {
		t.Fatalf("expected installed loading status to remain, got %q", m.Status)
	}

	updated, _ = m.Update(tui.InstalledLoadedMsg{Rows: []tui.InstalledSkill{
		{
			Target:         tui.TargetCodex,
			Scope:          tui.ScopeGlobal,
			Skill:          testSkillGoProjectRules,
			Managed:        tui.ManagedYes,
			Source:         testSourceAgentRules,
			QualifiedSkill: testQualifiedGoRules,
			Path:           testGoRulesPath,
		},
	}})
	m = asModel(t, updated)
	if m.Loading {
		t.Fatalf("expected loading to finish after installed rows load")
	}
}

func TestEnterOpensSkillDetails(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Width = 120
	m.Height = 80
	m.Skills = []tui.Skill{
		{
			Source:      testSourceAgentRules,
			Name:        testSkillGoProjectRules,
			Category:    "go",
			Triggers:    "go,golang,go.mod",
			Description: "Global Go project rules for architecture and contracts.",
		},
	}
	m.ApplyFilter()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewDetails {
		t.Fatalf("expected details view, got %q", m.ViewMode)
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Skill details",
		testQualifiedGoRules,
		"Triggers",
		"Default path",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected details view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestParseInstalledTSV(t *testing.T) {
	input := strings.Join([]string{
		core.InstalledHeader,
		strings.Join([]string{
			tui.TargetClaude, tui.ScopeProject, testSkillRulesSelector, tui.ManagedYes, testSourceAgentRules,
			testQualifiedRules, "/tmp/project/.claude/skills/rules-selector",
			testHashABC, testTimestampInstalled, "/tmp/project/.claude/skills/rules-selector",
		}, "\t"),
		strings.Join([]string{
			tui.TargetClaude, tui.ScopeProject, testSkillManual, "no", "-", "-",
			testManualSkillPath, "-", "-",
			testManualSkillPath,
		}, "\t"),
		"",
	}, "\n")

	rows, err := tui.ParseInstalledTSV(input)
	if err != nil {
		t.Fatalf("tui.ParseInstalledTSV returned error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 installed rows, got %d", len(rows))
	}
	if rows[0].Target != tui.TargetClaude ||
		rows[0].Scope != tui.ScopeProject ||
		rows[0].Managed != tui.ManagedYes ||
		rows[0].Source != testSourceAgentRules {
		t.Fatalf("unexpected managed row: %#v", rows[0])
	}
	if rows[1].Managed != "no" {
		t.Fatalf("expected unmanaged row, got %#v", rows[1])
	}
}

func TestMergeInstalledUsageRowsSkipsScannedDuplicates(t *testing.T) {
	scanned := []tui.InstalledSkill{
		{
			Target: tui.TargetCodex, Scope: tui.ScopeProject, Skill: testSkillRulesSelector,
			InstalledPath: testProjectRulesPath,
			Path:          testProjectRulesPath,
		},
	}
	usageRows := []tui.InstalledSkill{
		{
			Target: tui.TargetCodex, Scope: tui.ScopeProject, Skill: testSkillRulesSelector, ProjectPath: testProjectPath,
			TargetRoot: testProjectSkillsRoot, InstalledPath: testProjectRulesPath,
			Path: testProjectRulesPath, RegistryOnly: true,
		},
		{
			Target: tui.TargetClaude, Scope: tui.ScopeProject, Skill: testSkillRulesSelector,
			InstalledPath: "/tmp/other/.claude/skills/rules-selector",
			Path:          "/tmp/other/.claude/skills/rules-selector",
			RegistryOnly:  true,
		},
	}

	rows := tui.MergeInstalledUsageRows(scanned, usageRows)
	if len(rows) != 2 {
		t.Fatalf("expected scanned duplicate to be skipped, got %#v", rows)
	}
	if !rows[0].RegistryOnly ||
		rows[0].ProjectPath != testProjectPath ||
		rows[0].TargetRoot != testProjectSkillsRoot {
		t.Fatalf("expected scanned duplicate to be enriched from registry, got %#v", rows[0])
	}
	if !rows[1].RegistryOnly || rows[1].Target != tui.TargetClaude {
		t.Fatalf("expected usage-only claude row, got %#v", rows[1])
	}
}

func TestInstalledScreenGroupsLocationsBySkill(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 40
	m.ViewMode = tui.ViewInstalled
	m.InstalledRows = []tui.InstalledSkill{
		testManagedInstalled("/tmp/codex/rules-selector"),
		{
			Target: tui.TargetClaude, Scope: tui.ScopeProject, Skill: testSkillManual, Managed: "no",
			Source: "-", ProjectPath: testProjectPath, Path: testManualSkillPath,
		},
		{
			Target: tui.TargetGemini, Scope: tui.ScopeProject, Skill: testSkillRulesSelector, Managed: tui.ManagedYes,
			Source: testSourceAgentRules, ProjectPath: "/tmp/other-project",
			Path: "/tmp/other-project/.gemini/skills/rules-selector", RegistryOnly: true,
		},
		{
			Target: tui.TargetDirectory, Scope: tui.ScopeCustom, Skill: testSkillDocsProject, Managed: tui.ManagedYes,
			Source: testSourceAgentRules, TargetRoot: testSkillsRoot, Path: "/tmp/skills/docs-project-rules",
		},
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Installed skills",
		"Installed: 4",
		"Skills: 3",
		"Projects: 2",
		"Managed: 3",
		"Showing 1-3/3",
		"[M] managed by Skillhub",
		"[ ] unmanaged: read-only in TUI",
		"agent-rules/rules-selector",
		"2 locations",
		"targets: Codex global, Gemini project",
		"[ ] manual-skill",
		"unmanaged",
		"targets: Claude project",
		"agent-rules/docs-project-rules",
		"targets: directory custom",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected installed screen to contain %q, got:\n%s", want, view)
		}
	}
}

func TestInstalledReloadClampsCursorToGroupedSummaries(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 40
	m.ViewMode = tui.ViewInstalled
	m.InstalledCursor = 2
	m.InstalledOffset = 2

	updated, _ := m.Update(tui.InstalledLoadedMsg{Rows: []tui.InstalledSkill{
		testManagedInstalled("/tmp/codex/rules-selector"),
		{
			Target: tui.TargetClaude, Scope: tui.ScopeGlobal, Skill: testSkillRulesSelector,
			Managed: tui.ManagedYes, Source: testSourceAgentRules, Path: "/tmp/claude/rules-selector",
		},
		{
			Target: tui.TargetGemini, Scope: tui.ScopeGlobal, Skill: testSkillRulesSelector,
			Managed: tui.ManagedYes, Source: testSourceAgentRules, Path: "/tmp/gemini/rules-selector",
		},
	}})
	m = asModel(t, updated)
	if m.InstalledCursor != 0 || m.InstalledOffset != 0 {
		t.Fatalf(
			"expected reload to clamp cursor and offset to grouped summary, cursor=%d offset=%d",
			m.InstalledCursor,
			m.InstalledOffset,
		)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewInstalledDetails {
		t.Fatalf("expected clamped installed selection to open details, got view=%q status=%q", m.ViewMode, m.Status)
	}
}

func TestInstalledScreenCanScrollToLastProjectSkill(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 130
	m.Height = 28
	m.ViewMode = tui.ViewInstalled
	m.InstalledRows = actonInstalledRows()
	tui.SortInstalledRows(m.InstalledRows)
	m.EnsureInstalledCursorVisible()

	view := stripANSI(m.View())
	if !strings.Contains(view, "Showing ") {
		t.Fatalf("expected constrained installed view to expose a range indicator, got:\n%s", view)
	}

	for i := 0; i < 4; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = asModel(t, updated)
	}
	view = stripANSI(m.View())
	if !strings.Contains(view, "ton-blockchain") {
		t.Fatalf("expected scrolling to reveal last project skill, got:\n%s", view)
	}
	if !strings.Contains(view, "Showing ") || !strings.Contains(view, "/5") {
		t.Fatalf("expected scrolled installed view to include row range, got:\n%s", view)
	}
}

func TestInstallReloadFocusesNewProjectRows(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewInstallResult
	m.Width = 130
	m.Height = 40
	m.InstallResult = tui.InstallResult{
		SkillNames: []string{"acton/acton", "acton/func2tolk", "acton/tolk", "acton/ton-blockchain"},
		Targets: []tui.InstallTargetResult{
			{Target: tui.TargetCodex, Scope: tui.ScopeProject, Label: "Codex project", Root: "/tmp/tongoldy/.agents/skills"},
		},
	}

	updated, _ := m.Update(tui.InstalledLoadedMsg{Rows: []tui.InstalledSkill{
		actonInstalledRows()[0],
		actonInstalledRows()[1],
		actonInstalledRows()[2],
		actonInstalledRows()[3],
		actonInstalledRows()[4],
	}})
	m = asModel(t, updated)
	if m.InstalledCursor != 1 || m.InstalledOffset != 1 {
		t.Fatalf(
			"expected installed reload to focus first newly installed project row, cursor=%d offset=%d",
			m.InstalledCursor,
			m.InstalledOffset,
		)
	}

	m.ViewMode = tui.ViewInstalled
	view := stripANSI(m.View())
	if strings.Contains(view, "productivity_caveman") {
		t.Fatalf("expected installed view to focus project install group instead of unrelated global row, got:\n%s", view)
	}
	if !strings.Contains(view, testSourceActon) || !strings.Contains(view, "func2tolk") {
		t.Fatalf("expected installed view to show new project install group, got:\n%s", view)
	}
}

func TestEnterOpensInstalledSkillDetails(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 40
	m.ViewMode = tui.ViewInstalled
	m.InstalledRows = []tui.InstalledSkill{
		testManagedInstalled("/tmp/codex/rules-selector"),
		{
			Target: tui.TargetGemini, Scope: tui.ScopeProject, Skill: testSkillRulesSelector, Managed: tui.ManagedYes,
			Source: testSourceAgentRules, ProjectPath: testProjectB, Path: testProjectBGeminiPath,
			RegistryOnly: true, ContentHash: testHashABC, UpdatedAt: testTimestampUsage,
		},
		{
			Target: tui.TargetClaude, Scope: tui.ScopeProject, Skill: testSkillManual, Managed: "no",
			Source: "-", ProjectPath: testProjectA, Path: "/tmp/project-a/.claude/skills/manual-skill",
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewInstalledDetails {
		t.Fatalf("expected installed details view, got %q", m.ViewMode)
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Installed skill details",
		testQualifiedRules,
		"Installed in: 2",
		testLabelCodexGlobal,
		"LOCAL PROJECT",
		"Gemini project",
		"Project root: /tmp/project-b",
		"    source: agent-rules",
		"hash: abc",
		"updated: 2026-05-05T01:00:00Z",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected installed details to contain %q, got:\n%s", want, view)
		}
	}
}

func TestInstalledDetailsActionsUseHighlightedLocation(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 40
	m.ViewMode = tui.ViewInstalled
	m.InstalledRows = []tui.InstalledSkill{
		testManagedInstalled("/tmp/codex/rules-selector"),
		{
			Target: tui.TargetGemini, Scope: tui.ScopeProject, Skill: testSkillRulesSelector, Managed: tui.ManagedYes,
			Source: testSourceAgentRules, ProjectPath: testProjectB,
			Path: testProjectBGeminiPath, RegistryOnly: true,
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = asModel(t, updated)

	row, ok := m.CurrentInstalledDetail()
	if !ok {
		t.Fatalf("expected highlighted installed details row")
	}
	want := []string{commandUpdate, testFlagTarget, tui.TargetGemini, tui.FlagScope, tui.ScopeProject, tui.FlagProject, testProjectB}
	if got := tui.InstalledUpdateArgsForRow(row, "/tmp/fallback"); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("expected details update args %#v, got %#v", want, got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewConfirmDelete {
		t.Fatalf("expected confirm delete, got %q", m.ViewMode)
	}
	if m.PendingUninstall.ProjectPath != testProjectB {
		t.Fatalf("expected pending uninstall to use highlighted details row, got %#v", m.PendingUninstall)
	}
}

func TestEscReturnsFromInstalledDetails(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewInstalledDetails
	m.InstalledDetailKey = testQualifiedRules

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewInstalled {
		t.Fatalf("expected installed view after esc, got %q", m.ViewMode)
	}
}

func TestInstalledUpdateAndUninstallArgs(t *testing.T) {
	row := tui.InstalledSkill{Target: tui.TargetClaude, Scope: tui.ScopeProject, Skill: testSkillRulesSelector}
	project := testProjectPath

	updateWant := []string{
		commandUpdate, testFlagTarget, tui.TargetClaude,
		tui.FlagScope, tui.ScopeProject, tui.FlagProject, project,
	}
	if got := tui.InstalledUpdateArgsForRow(row, project); strings.Join(got, " ") != strings.Join(updateWant, " ") {
		t.Fatalf("expected update args %#v, got %#v", updateWant, got)
	}

	uninstallWant := []string{
		"uninstall", testSkillRulesSelector, testFlagTarget, tui.TargetClaude,
		tui.FlagScope, tui.ScopeProject, tui.FlagProject, project,
	}
	if got := tui.InstalledUninstallArgsForRow(row, project); strings.Join(got, " ") != strings.Join(uninstallWant, " ") {
		t.Fatalf("expected uninstall args %#v, got %#v", uninstallWant, got)
	}

	registryRow := tui.InstalledSkill{
		Target: tui.TargetGemini, Scope: tui.ScopeProject, Skill: testSkillRulesSelector,
		ProjectPath: testRecordedProject,
	}
	registryWant := []string{
		commandUpdate, testFlagTarget, tui.TargetGemini,
		tui.FlagScope, tui.ScopeProject, tui.FlagProject, testRecordedProject,
	}
	if got := tui.InstalledUpdateArgsForRow(registryRow, project); strings.Join(got, " ") != strings.Join(registryWant, " ") {
		t.Fatalf("expected registry update args %#v, got %#v", registryWant, got)
	}

	directoryRow := tui.InstalledSkill{
		Target: tui.TargetDirectory, Scope: tui.ScopeCustom,
		Skill: testSkillRulesSelector, TargetRoot: testSkillsRoot,
	}
	directoryWant := []string{"uninstall", testSkillRulesSelector, testFlagTarget, tui.TargetDirectory, tui.FlagDir, testSkillsRoot}
	if got := tui.InstalledUninstallArgsForRow(directoryRow, project); strings.Join(got, " ") != strings.Join(directoryWant, " ") {
		t.Fatalf("expected directory uninstall args %#v, got %#v", directoryWant, got)
	}

	registryOnlyRow := tui.InstalledSkill{
		Target: tui.TargetClaude, Scope: tui.ScopeGlobal, Skill: "productivity_grill-me",
		Managed: tui.ManagedYes, Path: "/tmp/home/.claude/skills/productivity_grill-me",
		InstalledPath: "/tmp/home/.claude/skills/productivity_grill-me", RegistryOnly: true,
	}
	registryOnlyWant := "uninstall productivity_grill-me --target claude --scope global"
	if got := tui.InstalledUninstallArgsForRow(registryOnlyRow, project); strings.Join(got, " ") != registryOnlyWant {
		t.Fatalf("expected registry-only uninstall args %q, got %q", registryOnlyWant, strings.Join(got, " "))
	}
}

func TestInstalledArgsWithLegacyEnv(t *testing.T) {
	t.Setenv("AGENT_SKILLS_DIR", "/tmp/legacy-skills")
	row := tui.InstalledSkill{
		Target: tui.TargetCodex, Scope: tui.ScopeGlobal,
		Skill: testSkillRulesSelector, Path: testLegacyRulesPath,
	}

	updateWant := "update --target codex --scope global"
	if got := tui.InstalledUpdateArgsForRow(row, testProjectPath); strings.Join(got, " ") != updateWant {
		t.Fatalf("expected explicit update args %q, got %q", updateWant, strings.Join(got, " "))
	}
	uninstallWant := "uninstall rules-selector --target codex --scope global"
	if got := tui.InstalledUninstallArgsForRow(row, "/tmp/project"); strings.Join(got, " ") != uninstallWant {
		t.Fatalf("expected explicit uninstall args %q, got %q", uninstallWant, strings.Join(got, " "))
	}
}

func TestParseActiveSourcesTSV(t *testing.T) {
	input := "name\ttype\tlocation\tref\tcatalog\n" +
		testSourceAgentRules + "\t" + testSourceTypeGit + "\thttps://github.com/assurrussa/agent-rules.git\t" +
		testSourceRefMain + "\t" + testCatalogSkillsPath + "\n"

	sources, err := tui.ParseSourcesTSV(input)
	if err != nil {
		t.Fatalf("tui.ParseSourcesTSV returned error: %v", err)
	}
	if len(sources) != 1 || sources[0].Name != testSourceAgentRules || sources[0].Catalog != testCatalogSkillsPath {
		t.Fatalf("unexpected sources: %#v", sources)
	}
}

func TestHelpShowsUpdateCommandsWithoutRunningSelfUpdate(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 30
	m.ViewMode = tui.ViewHelp

	view := stripANSI(m.View())
	for _, want := range []string{
		"Update and restore",
		"skillhub update",
		"skillhub update --cascade",
		"skillhub update --cascade -v",
		"skillhub installed update",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected help to contain %q, got:\n%s", want, view)
		}
	}
}

func TestHelpShowsProjectLockfileStatus(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 30
	m.ViewMode = tui.ViewHelp
	m.LockStatus = tui.ProjectLockStatus{
		Present:   true,
		Total:     3,
		Missing:   1,
		Changed:   1,
		Unchanged: 1,
		Skipped:   1,
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Project lockfile",
		"present",
		"rows=3",
		"missing=1",
		"changed=1",
		"skipped=1",
		"r restore project lockfile",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected help to contain %q, got:\n%s", want, view)
		}
	}
}

func TestUpdateRestoreStartsBusyCommand(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewHelp
	m.ProjectDir = testProjectPath

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = asModel(t, updated)
	if cmd == nil || !m.Busy {
		t.Fatalf("expected restore action to start busy command, busy=%v cmd=%v", m.Busy, cmd)
	}
	if !strings.Contains(m.Status, "Restoring project skills") {
		t.Fatalf("expected restore status, got %q", m.Status)
	}
}

func TestRestoreSummarySurvivesUpdateReload(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewHelp

	updated, _ := m.Update(tui.CommandDoneMsg{
		Action: "Restore project",
		Output: "Restored project Skills: installed=1 updated=0 unchanged=0 skipped=0 failed=0\n",
	})
	m = asModel(t, updated)
	if m.Loading || m.ViewMode != tui.ViewHelp {
		t.Fatalf("expected help to reload after restore, view=%q loading=%v", m.ViewMode, m.Loading)
	}

	updated, _ = m.Update(tui.LockStatusLoadedMsg{Status: tui.ProjectLockStatus{Present: true, Total: 1, Unchanged: 1}})
	m = asModel(t, updated)
	if !strings.Contains(m.Status, "Restored project Skills: installed=1 updated=0 unchanged=0 skipped=0 failed=0") {
		t.Fatalf("expected restore summary to remain visible, got %q", m.Status)
	}
}

func TestLockStatusLoadedDoesNotClearOtherViewLoading(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = true
	m.ViewMode = tui.ViewSources
	m.Status = "Loading sources..."
	m.PostReloadStatus = "Restored project Skills: installed=1 updated=0 unchanged=0 skipped=0 failed=0"

	updated, _ := m.Update(tui.LockStatusLoadedMsg{Status: tui.ProjectLockStatus{Present: true, Total: 1}})
	m = asModel(t, updated)
	if !m.Loading {
		t.Fatalf("stale lock status should not clear another view's loading state")
	}
	if m.Status != "Loading sources..." {
		t.Fatalf("stale lock status should not overwrite current status, got %q", m.Status)
	}
	if m.PostReloadStatus != "" {
		t.Fatalf("expected stale restore status to be consumed, got %q", m.PostReloadStatus)
	}
}

func TestAddSourceInputAcceptsDashboardShortcutDigits(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewAddSource

	for _, key := range []string{"r", "u", "l", "e", "s", "-", "v", "2"} {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = asModel(t, updated)
	}

	if m.ViewMode != tui.ViewAddSource {
		t.Fatalf("expected to stay in add-source view, got %q", m.ViewMode)
	}
	if m.SourceInput != "rules-v2" {
		t.Fatalf("expected source input to include digits, got %q", m.SourceInput)
	}
}

func TestAddSourceArgsNormalizeBracketedURLAndName(t *testing.T) {
	args := tui.SourceAddArgs("[https://github.com/mattpocock/skills]", "[mattpocock]")
	want := []string{testCommandAdd, testSourceURL, tui.FlagName, testSourceMattPocock}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("unexpected add source args: got %#v want %#v", args, want)
	}

	args = tui.SourceAddArgs("[tui.Skill repo](https://github.com/mattpocock/skills)", "")
	want = []string{testCommandAdd, testSourceURL}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("unexpected markdown-link add source args: got %#v want %#v", args, want)
	}

	args = tui.SourceAddArgs("<https://github.com/mattpocock/skills>", "<mattpocock>")
	want = []string{testCommandAdd, testSourceURL, tui.FlagName, testSourceMattPocock}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("unexpected angle-bracket add source args: got %#v want %#v", args, want)
	}
}

func TestAddSourceEmptyFieldsDoNotLookPreFilled(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewAddSource
	m.Width = 120

	view := stripANSI(m.AddSourceContent(90))
	if strings.Contains(view, "Location path or git URL") || strings.Contains(view, "Name optional source name") {
		t.Fatalf("empty fields should not render placeholder text as field values, got:\n%s", view)
	}
	if !strings.Contains(view, "Location  _") {
		t.Fatalf("expected empty active location field to show only cursor, got:\n%s", view)
	}
	if !strings.Contains(view, "Location expects a local path, git URL, or GitHub tree URL") {
		t.Fatalf("expected field help to explain source location, got:\n%s", view)
	}
}

func TestAddSourceInputCapturesOptionalNameField(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewAddSource

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(testSourceURL)})
	m = asModel(t, updated)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = asModel(t, updated)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(testSourceMattPocock)})
	m = asModel(t, updated)

	if m.SourceInput != testSourceURL {
		t.Fatalf("expected source location input to be captured, got %q", m.SourceInput)
	}
	if m.SourceNameInput != testSourceMattPocock {
		t.Fatalf("expected source name input to be captured, got %q", m.SourceNameInput)
	}
}

func TestInstalledCommandSummarySurvivesReload(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewInstalled

	updated, _ := m.Update(tui.CommandDoneMsg{
		Action: "Update installed",
		Output: "Summary codex/global: updated=0 unchanged=1 skipped=0 failed=0\nUpdated: 0, unchanged: 1, skipped: 0, failed: 0\n",
	})
	m = asModel(t, updated)
	if !m.Loading {
		t.Fatalf("expected installed reload after update")
	}

	updated, _ = m.Update(tui.InstalledLoadedMsg{Rows: []tui.InstalledSkill{
		testManagedInstalled("/tmp/skills/rules-selector"),
	}})
	m = asModel(t, updated)

	if !strings.Contains(m.Status, "Updated: 0, unchanged: 1, skipped: 0, failed: 0") {
		t.Fatalf("expected update summary to remain visible, got %q", m.Status)
	}
}

func TestQuestionMarkOpensHelpOverlay(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 30

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewHelp {
		t.Fatalf("expected help view, got %q", m.ViewMode)
	}
	view := stripANSI(m.View())
	for _, want := range []string{
		"Help",
		"1/2/3",
		"left/right",
		"skillhub update",
		"skillhub installed update",
		"skillhub installed usage update --projects",
		"r restore project lockfile",
		"Installed: enter locations",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected help overlay to contain %q, got:\n%s", want, view)
		}
	}
}

func TestQuestionMarkClosesHelpOverlay(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 30
	m.ViewMode = tui.ViewSkills

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewHelp {
		t.Fatalf("expected help view, got %q", m.ViewMode)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewSkills {
		t.Fatalf("expected second question mark to return to skills, got %q", m.ViewMode)
	}
}

func TestLeftRightSwitchDashboardSections(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 30
	m.ViewMode = tui.ViewInstalled

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewSkills || !m.Loading {
		t.Fatalf("expected right from installed to load skills section, got view=%q loading=%v", m.ViewMode, m.Loading)
	}

	m.Loading = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewSources || !m.Loading {
		t.Fatalf("expected right from skills to load sources section, got view=%q loading=%v", m.ViewMode, m.Loading)
	}

	m.Loading = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewSkills || !m.Loading {
		t.Fatalf("expected left from sources to load skills section, got view=%q loading=%v", m.ViewMode, m.Loading)
	}
}

func TestNumberKeysUseInstalledFirstDashboardOrder(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 30
	m.ViewMode = tui.ViewSkills

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewInstalled || !m.Loading {
		t.Fatalf("expected 1 to load installed section, got view=%q loading=%v", m.ViewMode, m.Loading)
	}

	m.Loading = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewSkills || !m.Loading {
		t.Fatalf("expected 2 to load skills section, got view=%q loading=%v", m.ViewMode, m.Loading)
	}

	m.Loading = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewSources || !m.Loading {
		t.Fatalf("expected 3 to load sources section, got view=%q loading=%v", m.ViewMode, m.Loading)
	}

	m.Loading = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewSources || m.Loading {
		t.Fatalf("expected 4 to be ignored, got view=%q loading=%v", m.ViewMode, m.Loading)
	}

	m.Loading = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewSources || m.Loading {
		t.Fatalf("expected 5 to be ignored, got view=%q loading=%v", m.ViewMode, m.Loading)
	}

	m.Loading = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("6")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewSources || m.Loading {
		t.Fatalf("expected 6 to be ignored, got view=%q loading=%v", m.ViewMode, m.Loading)
	}
}

func TestLeavingSkillsClearsSearch(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Skills = []tui.Skill{
		{
			Source: testSourceAgentRules, Name: testSkillGoProjectRules,
			Category: "go", Triggers: testTriggerGoGolang, Description: testDescGoRules,
		},
		{
			Source: testSourceMattPocock, Name: testSkillGrillMe,
			Category: testCategoryProductivity, Triggers: testTriggerGrill,
			Description: testDescGrillMe,
		},
	}
	m.Search = testTriggerGrill
	m.ApplyFilter()
	if len(m.Filtered) != 1 {
		t.Fatalf("expected search to filter skills before navigation, got %d", len(m.Filtered))
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = asModel(t, updated)
	if m.Search != "" || m.SearchMode {
		t.Fatalf("expected skill search to reset when leaving Skills, got search=%q mode=%v", m.Search, m.SearchMode)
	}
	if len(m.Filtered) != 2 {
		t.Fatalf("expected skill list filter to reset, got %d", len(m.Filtered))
	}
}

func TestHeaderHighlightsActiveSearch(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Search = testTriggerGrill

	view := m.RenderHeader(120)
	if !strings.Contains(view, "Search: ") || !strings.Contains(view, testTriggerGrill) {
		t.Fatalf("expected header to show active search, got:\n%s", stripANSI(view))
	}
	if got := tui.SearchBadgeStyle.GetBackground(); got != lipgloss.Color("63") {
		t.Fatalf("expected active search badge background, got %#v", got)
	}
	if got := tui.SearchBadgeStyle.GetBold(); !got {
		t.Fatalf("expected active search badge to be bold")
	}
}

func TestLeftRightDoesNotLeaveInstallTargetPicker(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewTargets
	m.TargetPurpose = tui.TargetPurposeInstall

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewTargets || m.Loading {
		t.Fatalf("expected install target picker to ignore right arrow, got view=%q loading=%v", m.ViewMode, m.Loading)
	}
}

func TestSkillSelectionUsesSoftActiveAndGreenQueueStyles(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Width = 120
	m.Height = 30
	m.Skills = []tui.Skill{
		{
			Source: testSourceAgentRules, Name: testSkillGoProjectRules,
			Category: "go", Triggers: testTriggerGoGolang,
			Description: testDescGoProjectRules,
		},
	}
	m.ApplyFilter()
	m.ToggleCurrent()

	view := m.View()
	if got := tui.ActiveRowStyle.GetBackground(); got != (lipgloss.NoColor{}) {
		t.Fatalf("active row should not use a strong background, got %#v", got)
	}
	if got := tui.SelectedRowStyle.GetForeground(); got != lipgloss.Color("42") {
		t.Fatalf("selected row should use green foreground, got %#v", got)
	}
	if !strings.Contains(stripANSI(view), "[+] go-project-rules") {
		t.Fatalf("queued row should keep an explicit queue marker, got:\n%s", stripANSI(view))
	}
}

func TestViewGroupsSkillsByCategoryTree(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Width = 120
	m.Height = 30
	m.Skills = []tui.Skill{
		{Source: testSourceAgentRules, Name: testSkillGoProjectRules, Category: "go", Description: testDescGoProjectRules},
		{Source: testSourceAgentRules, Name: testSkillDocsProject, Category: testDocsCategory, Description: "Documentation rules"},
	}
	m.ApplyFilter()

	view := stripANSI(m.View())
	for _, want := range []string{
		"• documentation",
		"└─ [ ] docs-project-rules",
		testBulletGo,
		"└─ [ ] go-project-rules",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected grouped tree view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestSkillsListShowsManagedInstalledBadge(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Width = 120
	m.Height = 30
	m.Skills = []tui.Skill{
		{Source: testSourceMattPocock, Name: testSkillGrillMe, Category: testCategoryProductivity, Description: testDescGrillMe},
		{Source: testSourceAgentRules, Name: testSkillGoProjectRules, Category: "go", Description: testDescGoRules},
	}
	m.InstalledRows = []tui.InstalledSkill{
		{
			Target: tui.TargetClaude, Scope: tui.ScopeGlobal, Source: testSourceMattPocock, Skill: testSkillGrillMe,
			QualifiedSkill: testQualifiedGrillMe, Managed: tui.ManagedYes,
			Path: testGrillMeClaudePath,
		},
		{
			Target: tui.TargetCodex, Scope: tui.ScopeProject, Source: testSourceMattPocock, Skill: testSkillGrillMe,
			QualifiedSkill: testQualifiedGrillMe, Managed: tui.ManagedYes,
			ProjectPath: "/repo", Path: "/repo/.agents/skills/productivity_grill-me",
		},
		{
			Target: tui.TargetCodex, Scope: tui.ScopeGlobal, Source: "-", Skill: testSkillGoProjectRules,
			Managed: "no", Path: "/home/me/.agents/skills/go-project-rules",
		},
		{
			Target: tui.TargetCodex, Scope: tui.ScopeGlobal, Source: testSourceAgentRules, Skill: testSkillGoProjectRules,
			QualifiedSkill: testQualifiedGoRules, Managed: tui.ManagedYes,
			Path: "/home/me/.agents/skills/go-project-rules", PathMissing: true,
		},
	}
	m.ApplyFilter()

	view := stripANSI(m.View())
	for _, want := range []string{
		testSkillGrillMe,
		"installed: 2 locations",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected skill list to contain %q, got:\n%s", want, view)
		}
	}
	if strings.Contains(view, "go-project-rules   installed:") {
		t.Fatalf("unmanaged or missing installed rows should not create catalog badges, got:\n%s", view)
	}
}

func TestSkillDetailsShowsInstalledLocations(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Width = 120
	m.Height = 40
	m.Skills = []tui.Skill{
		{Source: testSourceMattPocock, Name: testSkillGrillMe, Category: testCategoryProductivity, Description: testDescGrillMe},
	}
	m.InstalledRows = []tui.InstalledSkill{
		{
			Target:         tui.TargetClaude,
			Scope:          tui.ScopeGlobal,
			Source:         testSourceMattPocock,
			Skill:          testSkillGrillMe,
			QualifiedSkill: testQualifiedGrillMe,
			Managed:        tui.ManagedYes,
			Path:           testGrillMeClaudePath,
			RegistryOnly:   true,
		},
	}
	m.ApplyFilter()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	view := stripANSI(m.View())
	for _, want := range []string{
		"Installed locations",
		testLabelClaudeGlobal,
		testGrillMeClaudePath,
		"managed registry",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected skill details to contain %q, got:\n%s", want, view)
		}
	}
}

func TestSmallHeightViewKeepsDashboardHeaderVisible(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Width = 120
	m.Height = 18
	m.Skills = []tui.Skill{
		{
			Source:   testSourceAgentRules,
			Name:     "reusable-module-rules",
			Category: "architecture",
			Triggers: "reusable module,library,public surface,external consumer,release readiness,replace",
			Description: "Global reusable-module rules for public facades, supported import surfaces, " +
				"clean consumer probes, and host/library ownership.",
		},
		{
			Source:   testSourceMattPocock,
			Name:     "deprecated_design-an-interface",
			Category: "deprecated",
			Triggers: "deprecated,design-an-interface",
			Description: "Generate multiple radically different interface designs for a module using parallel sub-agents. " +
				"Use when user wants to design an API, explore interface options, compare module shapes, or mentions design it twice.",
		},
		{
			Source:   testSourceMattPocock,
			Name:     "deprecated_qa",
			Category: "deprecated",
			Triggers: "deprecated,qa",
			Description: "Interactive QA session where user reports bugs or issues conversationally, and the agent files GitHub issues. " +
				"Explores the codebase in the background for context and domain language.",
		},
	}
	m.ApplyFilter()

	view := stripANSI(m.View())
	for _, want := range []string{"Skillhub", "Sources: 2", "1 Installed", "2 Skills", "3 Sources"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected small-height view to keep dashboard text %q visible, got:\n%s", want, view)
		}
	}
	if got := lipgloss.Height(view); got > m.Height {
		t.Fatalf("expected rendered view to fit height %d, got %d lines:\n%s", m.Height, got, view)
	}
}

func TestParseDefaultSourcesTSV(t *testing.T) {
	input := "name\ttype\tlocation\tref\tcatalog\n" +
		testSourceAgentRules + "\t" + testSourceTypeGit + "\thttps://github.com/assurrussa/agent-rules.git\t" +
		testSourceRefMain + "\t" + testCatalogSkillsPath + "\n"

	sources, err := tui.ParseDefaultSourcesTSV(input)
	if err != nil {
		t.Fatalf("tui.ParseDefaultSourcesTSV returned error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Name != testSourceAgentRules || sources[0].Type != testSourceTypeGit {
		t.Fatalf("unexpected source preset: %#v", sources[0])
	}
}

func TestParseTargetsTSV(t *testing.T) {
	input := "id\tlabel\tstatus\tadapter\tdescription\n" +
		"claude\tClaude\tsupported\tskill-dir\tClaude skills\n"

	targets, err := tui.ParseTargetsTSV(input)
	if err != nil {
		t.Fatalf("tui.ParseTargetsTSV returned error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].ID != tui.TargetClaude || targets[0].Adapter != testAdapterSkillDir {
		t.Fatalf("unexpected target: %#v", targets[0])
	}
}

func TestLoadTargetsUsesGoBackend(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	writeFakeTUIRepo(t, repo)
	t.Setenv("HOME", filepath.Join(tmp, "home"))
	t.Setenv("SKILLHUB_CONFIG_DIR", filepath.Join(tmp, "config"))

	msg := tui.LoadTargets(repo)()
	loaded, ok := msg.(tui.TargetsLoadedMsg)
	if !ok {
		t.Fatalf("expected tui.TargetsLoadedMsg, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("tui.LoadTargets should use Go backend, got error: %v", loaded.Err)
	}
	if len(loaded.Targets) != 2 || loaded.Targets[0].ID != tui.TargetCodex {
		t.Fatalf("unexpected Targets: %#v", loaded.Targets)
	}
	if len(loaded.Detections) == 0 || loaded.Detections[0].Target != tui.TargetCodex {
		t.Fatalf("unexpected Detections: %#v", loaded.Detections)
	}
}

func TestParseTargetDetectionsTSV(t *testing.T) {
	input := "target\tscope\tstatus\tpath\texists\tskills\tmanaged\n" +
		"claude\tproject\tsupported\t/tmp/project/.claude/skills\tyes\t3\t2\n"

	detections, err := tui.ParseTargetDetectionsTSV(input)
	if err != nil {
		t.Fatalf("tui.ParseTargetDetectionsTSV returned error: %v", err)
	}
	if len(detections) != 1 {
		t.Fatalf("expected 1 detection, got %d", len(detections))
	}
	got := detections[0]
	if got.Target != tui.TargetClaude ||
		got.Scope != tui.ScopeProject ||
		got.Exists != tui.ManagedYes ||
		got.Skills != "3" ||
		got.Managed != "2" {
		t.Fatalf("unexpected detection: %#v", got)
	}
}

func TestInstallStartsScopeWizard(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSkills
	m.Width = 120
	m.Height = 80
	m.Skills = []tui.Skill{
		{Source: testSourceAgentRules, Name: testSkillGoProjectRules, Category: "go", Description: testDescGoProjectRules},
	}
	m.ApplyFilter()
	m.ToggleCurrent()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = asModel(t, updated)
	if m.Loading {
		t.Fatalf("expected scope wizard to render without loading targets")
	}
	if m.ViewMode != tui.ViewInstallScope {
		t.Fatalf("expected install scope wizard, got %q", m.ViewMode)
	}
	if m.InstallScope != tui.ScopeProject {
		t.Fatalf("expected project default scope, got %q", m.InstallScope)
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Install scope",
		"Project",
		"Current project",
		"User",
		"Global skills directory",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected scope wizard to contain %q, got:\n%s", want, view)
		}
	}
}

func TestInstallFlowStateRoundTripsThroughTestModelFacade(t *testing.T) {
	m := tui.InitialModel(".")
	m.ViewMode = tui.ViewInstallScope
	m.InstallScope = tui.ScopeGlobal
	m.InstallScopeCursor = 1
	m.TargetPurpose = tui.TargetPurposeInstall
	m.TargetChoices = []tui.InstallTargetChoice{
		testTargetChoice(testCodexGlobalKey, tui.TargetCodex, testLabelCodexGlobal, tui.ScopeGlobal, testSkillsRoot),
	}
	m.SelectedTargets = map[string]bool{testCodexGlobalKey: true}
	m.PendingInstall = tui.InstallResult{SkillNames: []string{testQualifiedGoRules}}
	m.InstallResult = tui.InstallResult{
		SkillNames: []string{testQualifiedGoRules},
		Targets: []tui.InstallTargetResult{
			{Label: testLabelCodexGlobal, Target: tui.TargetCodex, Scope: tui.ScopeGlobal, Root: testSkillsRoot},
		},
	}

	const permissionDenied = "permission denied"
	m.InstallProgress = tui.TestInstallProgress{
		Current: 1, Total: 2, Failed: true, Error: permissionDenied,
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = asModel(t, updated)
	if m.InstallScope != tui.ScopeGlobal || m.InstallScopeCursor != 1 {
		t.Fatalf("expected scope state to round-trip, got scope=%q cursor=%d", m.InstallScope, m.InstallScopeCursor)
	}
	if m.TargetPurpose != tui.TargetPurposeInstall ||
		len(m.TargetChoices) != 1 ||
		!m.SelectedTargets[testCodexGlobalKey] {
		t.Fatalf("expected target selection state to round-trip, got purpose=%q choices=%#v selected=%#v",
			m.TargetPurpose, m.TargetChoices, m.SelectedTargets)
	}
	if len(m.PendingInstall.SkillNames) != 1 || len(m.InstallResult.Targets) != 1 {
		t.Fatalf("expected install result state to round-trip, pending=%#v result=%#v", m.PendingInstall, m.InstallResult)
	}
	if !m.InstallProgress.Failed || m.InstallProgress.Error != permissionDenied || m.InstallProgress.Total != 2 {
		t.Fatalf("expected progress state to round-trip, got %#v", m.InstallProgress)
	}
}

func TestInstallWizardTargetsSupportedAgentsAndDetectedDefaults(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewInstallScope
	m.InstallScope = tui.ScopeProject
	m.Width = 120
	m.Height = 80

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	if !m.Loading {
		t.Fatalf("expected scope confirm to load targets")
	}
	updated, _ = m.Update(tui.TargetsLoadedMsg{Targets: []tui.Target{
		testSupportedTarget(tui.TargetCodex, tui.LabelCodex, testCodexSkillsDesc),
		testSupportedTarget(tui.TargetClaude, tui.LabelClaude, testClaudeSkillsDesc),
		testSupportedTarget(tui.TargetGemini, tui.LabelGemini, testGeminiSkillsDesc),
		testSupportedTarget(testTargetOpenCode, testLabelOpenCode, testOpenCodeSkillsDesc),
		testPlannedTarget(testTargetCursor, testLabelCursor),
	}, Detections: []tui.TargetDetection{
		{
			Target: tui.TargetClaude, Scope: tui.ScopeProject, Status: tui.TargetStatusSupported,
			Path: "/tmp/project/.claude/skills", Exists: tui.ManagedYes, Skills: "2", Managed: "1",
		},
		{
			Target: tui.TargetGemini, Scope: tui.ScopeProject, Status: tui.TargetStatusSupported,
			Path: "/tmp/project/.gemini/skills", Exists: tui.ManagedYes, Skills: "1", Managed: "0",
		},
	}})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewTargets {
		t.Fatalf("expected agent target step, got %q", m.ViewMode)
	}
	if m.SelectedTargets[testCodexProjectKey] || !m.SelectedTargets["claude:project"] || !m.SelectedTargets["gemini:project"] {
		t.Fatalf("expected detected project targets only, got %#v", m.SelectedTargets)
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Install agents",
		"[ ] Codex",
		"[✓] Claude",
		"[✓] Gemini",
		"[ ] OpenCode",
		"scope: project",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected target view to contain %q, got:\n%s", want, view)
		}
	}
	if strings.Contains(view, testLabelCursor) {
		t.Fatalf("planned targets should not appear in install agent step, got:\n%s", view)
	}
}

func TestInstallWizardTargetFallbackSelectsCodexProject(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewInstallScope
	m.InstallScope = tui.ScopeProject
	m.TargetPurpose = tui.TargetPurposeInstall

	updated, _ := m.Update(tui.TargetsLoadedMsg{Targets: []tui.Target{
		testSupportedTarget(tui.TargetCodex, tui.LabelCodex, testCodexSkillsDesc),
		testSupportedTarget(tui.TargetClaude, tui.LabelClaude, testClaudeSkillsDesc),
	}})
	m = asModel(t, updated)
	if !m.SelectedTargets[testCodexProjectKey] || m.SelectedTargets["claude:project"] {
		t.Fatalf("expected Codex project fallback, got %#v", m.SelectedTargets)
	}
}

func TestInstallWizardUserScopeBuildsGlobalAgentChoices(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewInstallScope
	m.InstallScope = tui.ScopeProject
	m.InstallScopeCursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = asModel(t, updated)
	if m.InstallScope != tui.ScopeGlobal {
		t.Fatalf("expected User selection to map to global scope, got %q", m.InstallScope)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	updated, _ = m.Update(tui.TargetsLoadedMsg{Targets: []tui.Target{
		testSupportedTarget(tui.TargetCodex, tui.LabelCodex, testCodexSkillsDesc),
		testSupportedTarget(tui.TargetClaude, tui.LabelClaude, testClaudeSkillsDesc),
	}})
	m = asModel(t, updated)
	for _, choice := range m.TargetChoices {
		if choice.Scope != tui.ScopeGlobal {
			t.Fatalf("expected only global choices for User scope, got %#v", m.TargetChoices)
		}
	}
	if !m.SelectedTargets[testCodexGlobalKey] {
		t.Fatalf("expected Codex global fallback, got %#v", m.SelectedTargets)
	}
}

func TestInstallWizardConfirmRendersSelectionAndStartsInstall(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewTargets
	m.TargetPurpose = tui.TargetPurposeInstall
	m.Width = 120
	m.Height = 40
	m.ProjectDir = testProjectPath
	m.Selected = map[string]bool{
		testQualifiedGoRules: true,
	}
	m.TargetChoices = []tui.InstallTargetChoice{
		testTargetChoice(testCodexProjectKey, tui.TargetCodex, tui.LabelCodex, tui.ScopeProject, testProjectSkillsRoot),
		testTargetChoice("claude:project", tui.TargetClaude, tui.LabelClaude, tui.ScopeProject, "/tmp/project/.claude/skills"),
	}
	m.SelectedTargets = map[string]bool{
		testCodexProjectKey: true,
		"claude:project":    true,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewInstallConfirm {
		t.Fatalf("expected confirm screen, got %q", m.ViewMode)
	}
	view := stripANSI(m.View())
	for _, want := range []string{
		"Confirm install",
		testQualifiedGoRules,
		"Scope",
		"Project",
		tui.LabelCodex,
		"/tmp/project/.agents/skills/go-project-rules",
		tui.LabelClaude,
		"/tmp/project/.claude/skills/go-project-rules",
		"enter/y install",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected confirm screen to contain %q, got:\n%s", want, view)
		}
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	if cmd == nil || !m.Busy || m.InstallProgress.Total != 2 {
		t.Fatalf("expected confirmed install to start 2-step queue, busy=%v progress=%#v cmd=%v", m.Busy, m.InstallProgress, cmd)
	}
}

func TestInstallWizardBackNavigation(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewTargets
	m.TargetPurpose = tui.TargetPurposeInstall
	m.TargetChoices = []tui.InstallTargetChoice{
		{Key: testCodexProjectKey, Target: tui.TargetCodex, Label: tui.LabelCodex, Scope: tui.ScopeProject, Supported: true},
	}
	m.SelectedTargets = map[string]bool{testCodexProjectKey: true}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewInstallScope {
		t.Fatalf("expected esc from agents to return to scope, got %q", m.ViewMode)
	}

	m.ViewMode = tui.ViewInstallConfirm
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewTargets {
		t.Fatalf("expected esc from confirm to return to agents, got %q", m.ViewMode)
	}
}

func TestTargetSelectionShowsDetectionStats(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 30
	m.ProjectDir = testProjectPath

	updated, _ := m.Update(tui.TargetsLoadedMsg{
		Targets: []tui.Target{
			testSupportedTarget(tui.TargetClaude, tui.LabelClaude, testClaudeSkillsDesc),
		},
		Detections: []tui.TargetDetection{
			{
				Target: tui.TargetClaude, Scope: tui.ScopeGlobal, Status: tui.TargetStatusSupported,
				Path: "/home/me/.claude/skills", Exists: tui.ManagedYes, Skills: "4", Managed: "2",
			},
			{
				Target: tui.TargetClaude, Scope: tui.ScopeProject, Status: tui.TargetStatusSupported,
				Path: "/tmp/project/.claude/skills", Exists: "no", Skills: "0", Managed: "0",
			},
		},
	})
	m = asModel(t, updated)

	view := stripANSI(m.View())
	for _, want := range []string{
		testLabelClaudeGlobal,
		"/home/me/.claude/skills",
		"exists: yes",
		"skills: 4",
		"managed: 2",
		"Claude project",
		"exists: no",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected target stats view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestInstallSuccessOpensResultScreen(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 40
	m.Selected = map[string]bool{
		testQualifiedGoRules: true,
		testQualifiedRules:   true,
	}
	m.TargetChoices = []tui.InstallTargetChoice{
		testTargetChoice(testCodexGlobalKey, tui.TargetCodex, testLabelCodexGlobal, tui.ScopeGlobal, testSkillsRoot),
		testTargetChoice("claude:project", tui.TargetClaude, "Claude project", tui.ScopeProject, "/tmp/project/.claude/skills"),
	}
	m.SelectedTargets = map[string]bool{
		testCodexGlobalKey: true,
	}

	updated, _ := m.InstallToSelectedTargets()
	m = asModel(t, updated)
	if !m.Busy {
		t.Fatalf("expected install to mark TUI busy")
	}
	if m.InstallProgress.Total != 2 || m.InstallProgress.Current != 1 {
		t.Fatalf("expected queued install progress 1/2, got %#v", m.InstallProgress)
	}

	updated, _ = m.Update(tui.InstallStepDoneMsg{
		Output: "Installed go-project-rules from agent-rules to /tmp/skills/go-project-rules\n",
	})
	m = asModel(t, updated)
	if !m.Busy || m.InstallProgress.Current != 2 || m.InstallProgress.Completed != 1 {
		t.Fatalf("expected queued install progress 2/2 after first step, got busy=%v progress=%#v", m.Busy, m.InstallProgress)
	}

	updated, cmd := m.Update(tui.InstallStepDoneMsg{
		Output: "Installed rules-selector from agent-rules to /tmp/skills/rules-selector\n",
	})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewInstallResult {
		t.Fatalf("expected install result view, got %q", m.ViewMode)
	}
	if cmd == nil {
		t.Fatalf("expected install completion to refresh installed rows")
	}
	updated, _ = m.Update(tui.InstalledLoadedMsg{Rows: []tui.InstalledSkill{
		{
			Target: tui.TargetCodex, Scope: tui.ScopeGlobal, Source: testSourceAgentRules, Skill: testSkillGoProjectRules,
			QualifiedSkill: testQualifiedGoRules, Managed: tui.ManagedYes,
			Path: testGoRulesPath,
		},
	}})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewInstallResult {
		t.Fatalf("expected installed refresh to keep result view, got %q", m.ViewMode)
	}
	if len(m.InstalledRows) != 1 {
		t.Fatalf("expected installed rows to refresh, got %#v", m.InstalledRows)
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Install complete",
		testLabelCodexGlobal,
		testSkillsRoot,
		testGoRulesPath,
		"/tmp/skills/rules-selector",
		"enter back",
		"q quit",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected install result view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestInstallProgressViewShowsCurrentStep(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 32
	m.Selected = map[string]bool{
		testQualifiedGoRules:         true,
		"mattpocock/engineering_tdd": true,
	}
	m.TargetChoices = []tui.InstallTargetChoice{
		testTargetChoice(testCodexGlobalKey, tui.TargetCodex, testLabelCodexGlobal, tui.ScopeGlobal, "/tmp/codex"),
		testTargetChoice(testClaudeGlobalKey, tui.TargetClaude, testLabelClaudeGlobal, tui.ScopeGlobal, "/tmp/claude"),
	}
	m.SelectedTargets = map[string]bool{
		testCodexGlobalKey:  true,
		testClaudeGlobalKey: true,
	}

	updated, cmd := m.InstallToSelectedTargets()
	m = asModel(t, updated)
	if cmd == nil {
		t.Fatalf("expected first install step command")
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Install progress",
		"Installing 1/4",
		"Running",
		"Skill agent-rules/go-project-rules",
		"Target Codex global",
		"Queue",
		testQueueStateRunning,
		testQueueStatePending,
		testQualifiedGoRules,
		"mattpocock/engineering_tdd",
		"[",
		"]",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected install progress view to contain %q, got:\n%s", want, view)
		}
	}

	updated, _ = m.Update(tui.InstallStepDoneMsg{Output: "Installed one\n"})
	m = asModel(t, updated)
	view = stripANSI(m.View())
	for _, want := range []string{
		"done",
		testQueueStateRunning,
		"Installing 2/4",
		"Target Claude global",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected updated queue progress to contain %q, got:\n%s", want, view)
		}
	}
}

func TestGenericBusyViewShowsWorkingPanel(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Busy = true
	m.ViewMode = tui.ViewSources
	m.Status = "Syncing sources..."
	m.Width = 120
	m.Height = 30

	view := stripANSI(m.View())
	for _, want := range []string{
		"Working",
		"Running",
		"Syncing sources...",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected generic busy view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestInstallProgressStopsOnFailedStep(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 32
	m.Selected = map[string]bool{
		testQualifiedGoRules: true,
	}
	m.TargetChoices = []tui.InstallTargetChoice{
		testTargetChoice(testCodexGlobalKey, tui.TargetCodex, testLabelCodexGlobal, tui.ScopeGlobal, "/tmp/codex"),
		testTargetChoice(testClaudeGlobalKey, tui.TargetClaude, testLabelClaudeGlobal, tui.ScopeGlobal, "/tmp/claude"),
	}
	m.SelectedTargets = map[string]bool{
		testCodexGlobalKey:  true,
		testClaudeGlobalKey: true,
	}

	updated, _ := m.InstallToSelectedTargets()
	m = asModel(t, updated)
	updated, _ = m.Update(tui.InstallStepDoneMsg{
		Output: testPermissionDeniedLine,
		Err:    errors.New("exit status 1"),
	})
	m = asModel(t, updated)

	if m.Busy {
		t.Fatalf("expected install failure to stop busy state")
	}
	if !m.InstallProgress.Failed {
		t.Fatalf("expected failed progress state, got %#v", m.InstallProgress)
	}
	view := stripANSI(m.View())
	for _, want := range []string{
		"Install failed",
		"Installing 1/2",
		"Skill agent-rules/go-project-rules",
		"Target Codex global",
		"permission denied",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected failed install progress view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestInstallProgressFailureCanReturnToTargets(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewTargets
	m.TargetPurpose = tui.TargetPurposeInstall
	m.Width = 120
	m.Height = 32
	m.Selected = map[string]bool{
		testQualifiedGoRules: true,
	}
	m.TargetChoices = []tui.InstallTargetChoice{
		testTargetChoice(testCodexGlobalKey, tui.TargetCodex, testLabelCodexGlobal, tui.ScopeGlobal, "/tmp/codex"),
	}
	m.SelectedTargets = map[string]bool{
		testCodexGlobalKey: true,
	}

	updated, _ := m.InstallToSelectedTargets()
	m = asModel(t, updated)
	updated, _ = m.Update(tui.InstallStepDoneMsg{Output: testPermissionDeniedLine, Err: errors.New("exit status 1")})
	m = asModel(t, updated)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)

	if m.InstallProgress.Failed || m.InstallProgress.Total != 0 {
		t.Fatalf("expected failed install progress to clear, got %#v", m.InstallProgress)
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "Install agents") || strings.Contains(view, "Install failed") {
		t.Fatalf("expected enter to return to target picker, got:\n%s", view)
	}
}

func TestInstallResultCanReturnToTargetsOrInstalled(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewInstallResult
	m.Status = "Install complete."

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewTargets {
		t.Fatalf("expected t to return to targets, got %q", m.ViewMode)
	}

	m.ViewMode = tui.ViewInstallResult
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewInstalled {
		t.Fatalf("expected enter to return to installed, got %q", m.ViewMode)
	}
}

func TestTargetSelectionOnlyTogglesSupportedTargets(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.TargetChoices = []tui.InstallTargetChoice{
		{Key: testTargetCursor, Label: testLabelCursor, Supported: false},
		{
			Key: testClaudeGlobalKey, Label: testLabelClaudeGlobal,
			Target: tui.TargetClaude, Scope: tui.ScopeGlobal, Supported: true,
		},
	}
	m.SelectedTargets = map[string]bool{}
	m.TargetCursor = 0

	m.ToggleCurrentTarget()
	if m.SelectedTargets[testTargetCursor] {
		t.Fatalf("planned targets should not be selectable")
	}

	m.TargetCursor = 1
	m.ToggleCurrentTarget()
	if !m.SelectedTargets[testClaudeGlobalKey] {
		t.Fatalf("supported target should be selectable")
	}
}

func TestTargetChoicesShowSupportedBeforePlanned(t *testing.T) {
	m := tui.InitialModel(".")
	choices := m.BuildTargetChoices([]tui.Target{
		testPlannedTarget(testTargetCursor, testLabelCursor),
		testSupportedTarget(tui.TargetGemini, tui.LabelGemini, testGeminiSkillsDesc),
		{ID: "copilot", Label: "Copilot", Status: testStatusPlanned, Adapter: testStatusPlanned, Description: testDescPlanned},
		testSupportedTarget(tui.TargetCodex, tui.LabelCodex, testCodexSkillsDesc),
	})

	got := make([]string, 0, len(choices))
	for _, choice := range choices {
		got = append(got, choice.Key)
	}
	wantPrefix := []string{"gemini:global", "gemini:project", testCodexGlobalKey, testCodexProjectKey}
	if strings.Join(got[:len(wantPrefix)], " ") != strings.Join(wantPrefix, " ") {
		t.Fatalf("expected supported choices first, got %#v", got)
	}
	if got[len(got)-2] != testTargetCursor || got[len(got)-1] != "copilot" {
		t.Fatalf("expected planned choices last, got %#v", got)
	}
}

func TestTargetSelectionScrollsToKeepCursorVisible(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 24
	m.ViewMode = tui.ViewTargets
	m.SelectedTargets = map[string]bool{}
	m.TargetChoices = []tui.InstallTargetChoice{
		{Key: "agent-00", Label: "Agent 00", Status: testStatusSupported, Description: "Agent 00 skills", Supported: true},
		{Key: "agent-01", Label: "Agent 01", Status: testStatusSupported, Description: "Agent 01 skills", Supported: true},
		{Key: "agent-02", Label: "Agent 02", Status: testStatusSupported, Description: "Agent 02 skills", Supported: true},
		{Key: "agent-03", Label: "Agent 03", Status: testStatusSupported, Description: "Agent 03 skills", Supported: true},
		{Key: "agent-04", Label: "Agent 04", Status: testStatusSupported, Description: "Agent 04 skills", Supported: true},
		{Key: "agent-05", Label: "Agent 05", Status: testStatusSupported, Description: "Agent 05 skills", Supported: true},
		{Key: "agent-06", Label: "Agent 06", Status: testStatusSupported, Description: "Agent 06 skills", Supported: true},
	}

	for i := 0; i < 5; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = asModel(t, updated)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "› [ ] Agent 05") {
		t.Fatalf("expected cursor row to stay visible, got:\n%s", view)
	}
	if strings.Contains(view, "Agent 00") {
		t.Fatalf("expected scrolled target view to hide the first row, got:\n%s", view)
	}
	if got := lipgloss.Height(view); got > m.Height {
		t.Fatalf("expected target view to fit height %d, got %d lines:\n%s", m.Height, got, view)
	}
}

func TestSkillsScrollKeepsCursorVisibleAtSmallHeight(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 22
	m.ViewMode = tui.ViewSkills
	for i := 0; i < 7; i++ {
		m.Skills = append(m.Skills, tui.Skill{
			Source:      testSourceAgentRules,
			Name:        fmt.Sprintf("skill-%02d", i),
			Category:    "go",
			Description: "Short skill description.",
		})
	}
	m.ApplyFilter()

	for i := 0; i < 5; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = asModel(t, updated)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "›") || !strings.Contains(view, "skill-05") {
		t.Fatalf("expected selected skill row to stay visible, got:\n%s", view)
	}
	if got := lipgloss.Height(view); got > m.Height {
		t.Fatalf("expected skills view to fit height %d, got %d lines:\n%s", m.Height, got, view)
	}
}

func TestSkillsScrollKeepsCursorVisibleWithWrappedGroupedCards(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 88
	m.Height = 30
	m.ViewMode = tui.ViewSkills
	longDescription := strings.Repeat(
		"Detailed cursor visibility text wraps across several terminal lines so earlier cards can exhaust the panel body. ",
		3,
	)
	m.Skills = []tui.Skill{
		{
			Source:      "alpha",
			Name:        "skill-00",
			Category:    "backend",
			Triggers:    "alpha-backend",
			Description: longDescription,
		},
		{
			Source:      "alpha",
			Name:        "skill-01",
			Category:    "frontend",
			Triggers:    "alpha-frontend",
			Description: longDescription,
		},
		{
			Source:      "beta",
			Name:        "skill-02",
			Category:    "go",
			Triggers:    "beta-go",
			Description: longDescription,
		},
		{
			Source:      "beta",
			Name:        "skill-03",
			Category:    "go",
			Description: "Short skill description.",
		},
	}
	m.ApplyFilter()

	for i := 0; i < 2; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = asModel(t, updated)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "›") || !strings.Contains(view, "skill-02") {
		t.Fatalf("expected selected wrapped skill row to stay visible, got:\n%s", view)
	}
	if strings.Contains(view, "skill-00") {
		t.Fatalf("expected wrapped skills view to scroll away from first row, got:\n%s", view)
	}
	if got := lipgloss.Height(view); got > m.Height {
		t.Fatalf("expected wrapped skills view to fit height %d, got %d lines:\n%s", m.Height, got, view)
	}
}

func TestInstalledDetailsScrollKeepsCursorVisibleAtSmallHeight(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 24
	m.ViewMode = tui.ViewInstalledDetails
	m.InstalledDetailKey = testQualifiedRules
	for i := 0; i < 6; i++ {
		project := fmt.Sprintf("/tmp/project-%02d", i)
		m.InstalledRows = append(m.InstalledRows, tui.InstalledSkill{
			Source:      testSourceAgentRules,
			Skill:       testSkillRulesSelector,
			Managed:     tui.ManagedYes,
			Target:      tui.TargetCodex,
			Scope:       tui.ScopeProject,
			ProjectPath: project,
			Path:        project + "/.agents/skills/rules-selector",
		})
	}

	for i := 0; i < 5; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = asModel(t, updated)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "›") || !strings.Contains(view, "/tmp/project-05") {
		t.Fatalf("expected selected installed detail row to stay visible, got:\n%s", view)
	}
	if got := lipgloss.Height(view); got > m.Height {
		t.Fatalf("expected installed detail view to fit height %d, got %d lines:\n%s", m.Height, got, view)
	}
}

func TestSourcesScrollKeepsCursorVisibleAtSmallHeight(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 24
	m.ViewMode = tui.ViewSources
	for i := 0; i < 7; i++ {
		m.Sources = append(m.Sources, tui.SourcePreset{
			Name:     fmt.Sprintf("source-%02d", i),
			Type:     testSourceTypeGit,
			Status:   "fresh",
			Ref:      testSourceRefMain,
			Catalog:  testCatalogSkillsPath,
			Location: fmt.Sprintf("/tmp/source-%02d", i),
		})
	}

	for i := 0; i < 5; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = asModel(t, updated)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "› source-05") {
		t.Fatalf("expected selected source row to stay visible, got:\n%s", view)
	}
	if got := lipgloss.Height(view); got > m.Height {
		t.Fatalf("expected sources view to fit height %d, got %d lines:\n%s", m.Height, got, view)
	}
}

func TestSourceDefaultsScrollKeepsCursorVisibleAtSmallHeight(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.Width = 120
	m.Height = 18
	m.ViewMode = tui.ViewDefaults
	for i := 0; i < 12; i++ {
		m.Defaults = append(m.Defaults, tui.SourcePreset{
			Name:     fmt.Sprintf("source-%02d", i),
			Type:     testSourceTypeGit,
			Ref:      testSourceRefMain,
			Location: fmt.Sprintf("/tmp/source-%02d", i),
		})
	}

	for i := 0; i < 9; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = asModel(t, updated)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "› source-09") {
		t.Fatalf("expected selected default source row to stay visible, got:\n%s", view)
	}
	if strings.Contains(view, "source-00") {
		t.Fatalf("expected default source view to scroll away from first row, got:\n%s", view)
	}
	if got := lipgloss.Height(view); got > m.Height {
		t.Fatalf("expected default source view to fit height %d, got %d lines:\n%s", m.Height, got, view)
	}
}

func TestSkillVisibleCountLeavesRoomForMultilineCards(t *testing.T) {
	m := tui.InitialModel(".")
	m.Height = 30

	if got := m.VisibleCount(); got != 3 {
		t.Fatalf("expected 3 visible skill cards at height 30, got %d", got)
	}
}

func TestCustomSourceInputModeCapturesText(t *testing.T) {
	m := tui.InitialModel(".")
	m.Loading = false
	m.ViewMode = tui.ViewSources

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = asModel(t, updated)
	if m.ViewMode != tui.ViewAddSource {
		t.Fatalf("expected add source mode, got %q", m.ViewMode)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("../agent-rules")})
	m = asModel(t, updated)
	if m.SourceInput != "../agent-rules" {
		t.Fatalf("expected source input to be captured, got %q", m.SourceInput)
	}
}

func TestInstallTargetPathUsesProjectScope(t *testing.T) {
	m := tui.InitialModel(".")
	m.InstallScope = tui.ScopeProject
	m.ProjectDir = "/tmp/example-project"

	want := filepath.Join("/tmp/example-project", ".agents", "skills")
	if got := m.InstallTargetPath(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestInstallTargetPathUsesLegacyGlobalEnv(t *testing.T) {
	t.Setenv("AGENT_SKILLS_DIR", "/tmp/legacy-skills")
	m := tui.InitialModel(".")
	m.InstallScope = tui.ScopeGlobal

	if got := m.InstallTargetPath(); got != "/tmp/legacy-skills" {
		t.Fatalf("expected legacy target, got %q", got)
	}
}

func TestInstallArgsWithLegacyEnv(t *testing.T) {
	t.Setenv("AGENT_SKILLS_DIR", "/tmp/legacy-skills")

	args := tui.InstallArgsForTargetChoice(
		tui.InstallTargetChoice{Target: tui.TargetCodex, Scope: tui.ScopeGlobal},
		"/tmp/project",
		[]string{testSkillRulesSelector},
	)
	want := []string{"install", tui.FlagTarget, tui.TargetCodex, tui.FlagScope, tui.ScopeGlobal, testSkillRulesSelector}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Fatalf("expected explicit install args %#v, got %#v", want, args)
	}
}

func TestInstallStepWithLegacyEnv(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "source")
	legacyDir := filepath.Join(tmp, "legacy-skills")
	homeDir := filepath.Join(tmp, "home")
	writeFakeTUIRepo(t, repo)
	writeTUINestedPathSource(t, configDir, sourceDir)
	t.Setenv("SKILLHUB_CONFIG_DIR", configDir)
	t.Setenv("SKILLHUB_CACHE_DIR", cacheDir)
	t.Setenv("AGENT_SKILLS_DIR", legacyDir)
	t.Setenv("HOME", homeDir)

	model := tui.InitialModel(repo)
	model.Loading = false
	model.Selected = map[string]bool{"nested/engineering_tdd": true}
	model.TargetChoices = []tui.InstallTargetChoice{
		testTargetChoice(testCodexGlobalKey, tui.TargetCodex, testLabelCodexGlobal, tui.ScopeGlobal, legacyDir),
	}
	model.SelectedTargets = map[string]bool{testCodexGlobalKey: true}

	updated, cmd := model.InstallToSelectedTargets()
	if cmd == nil {
		t.Fatalf("expected install command")
	}
	model = asModel(t, updated)
	batchMsg := cmd()
	batch, ok := batchMsg.(tea.BatchMsg)
	if !ok || len(batch) == 0 {
		t.Fatalf("expected install batch command, got %#v", batchMsg)
	}
	msg := batch[0]()
	updated, _ = model.Update(msg)
	model = asModel(t, updated)
	if strings.Contains(model.Status, "failed") {
		t.Fatalf("install failed: %s", model.Status)
	}
	if _, err := os.Stat(filepath.Join(legacyDir, "engineering_tdd", "SKILL.md")); err != nil {
		t.Fatalf("expected install in AGENT_SKILLS_DIR: %v", err)
	}
	if _, err := os.Stat(filepath.Join(homeDir, ".agents", "skills", "engineering_tdd")); !os.IsNotExist(err) {
		t.Fatalf("expected install not to use default home target, stat err=%v", err)
	}
}
