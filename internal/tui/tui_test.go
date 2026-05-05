package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func stripANSI(value string) string {
	return ansiRE.ReplaceAllString(value, "")
}

func TestParseSkillsTSV(t *testing.T) {
	input := "source\tname\tcategory\ttriggers\tdescription\n" +
		"agent-rules\tgo-project-rules\tgo\tgo,golang\tGo project rules\n"

	skills, err := parseSkillsTSV(input)
	if err != nil {
		t.Fatalf("parseSkillsTSV returned error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "go-project-rules" || skills[0].Category != "go" {
		t.Fatalf("unexpected skill: %#v", skills[0])
	}
}

func TestApplyFilterMatchesDescriptionAndTriggers(t *testing.T) {
	m := initialModel(".")
	m.skills = []Skill{
		{Source: "agent-rules", Name: "go-project-rules", Category: "go", Triggers: "go,golang", Description: "Go project rules"},
		{Source: "agent-rules", Name: "docs-project-rules", Category: "documentation", Triggers: "docs", Description: "Documentation rules"},
	}
	m.search = "golang"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered skill, got %d", len(m.filtered))
	}
	if m.skills[m.filtered[0]].Name != "go-project-rules" {
		t.Fatalf("unexpected filtered skill: %#v", m.skills[m.filtered[0]])
	}
}

func TestSelectionUsesQualifiedSkillNames(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.skills = []Skill{
		{Source: "alpha", Name: "same-name", Category: "go", Description: "Alpha"},
		{Source: "beta", Name: "same-name", Category: "go", Description: "Beta"},
	}
	m.applyFilter()

	m.toggleCurrent()
	m.cursor = 1
	m.toggleCurrent()

	got := m.selectedNames()
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
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 20
	m.skills = []Skill{
		{Source: "agent-rules", Name: "go-project-rules", Category: "go", Description: "Go project rules"},
	}
	m.applyFilter()

	view := m.View()
	if !strings.Contains(view, "agent-rules") {
		t.Fatalf("expected view to include source name, got:\n%s", view)
	}
}

func TestViewShowsTaskOrientedSkillList(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 30
	m.skills = []Skill{
		{
			Source:      "agent-rules",
			Name:        "go-project-rules",
			Category:    "go",
			Triggers:    "go,golang,go.mod",
			Description: "Global Go project rules for architecture and contracts.",
		},
	}
	m.applyFilter()

	view := stripANSI(m.View())
	for _, want := range []string{
		"Skillhub",
		"Sources: 1",
		"Skills",
		"[ ] go-project-rules",
		"• go",
		"source: agent-rules",
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

func TestDashboardRendersSectionNavigation(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 30
	m.skills = []Skill{
		{Source: "agent-rules", Name: "go-project-rules", Category: "go", Description: "Go rules"},
	}
	m.applyFilter()

	view := stripANSI(m.View())
	for _, want := range []string{
		"1 Skills",
		"2 Installed",
		"3 Usage",
		"4 Sources",
		"5 Targets",
		"6 Update",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected dashboard navigation to contain %q, got:\n%s", want, view)
		}
	}
}

func TestEnterOpensSkillDetails(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 80
	m.skills = []Skill{
		{
			Source:      "agent-rules",
			Name:        "go-project-rules",
			Category:    "go",
			Triggers:    "go,golang,go.mod",
			Description: "Global Go project rules for architecture and contracts.",
		},
	}
	m.applyFilter()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.viewMode != viewDetails {
		t.Fatalf("expected details view, got %q", m.viewMode)
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Skill details",
		"agent-rules/go-project-rules",
		"Triggers",
		"Default path",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected details view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestParseInstalledTSV(t *testing.T) {
	input := "target\tscope\tskill\tmanaged\tsource\tqualified_skill\tinstalled_path\tcontent_hash\tinstalled_at\tpath\n" +
		"claude\tproject\trules-selector\tyes\tagent-rules\tagent-rules/rules-selector\t/tmp/project/.claude/skills/rules-selector\tabc\t2026-05-05T00:00:00Z\t/tmp/project/.claude/skills/rules-selector\n" +
		"claude\tproject\tmanual-skill\tno\t-\t-\t/tmp/project/.claude/skills/manual-skill\t-\t-\t/tmp/project/.claude/skills/manual-skill\n"

	rows, err := parseInstalledTSV(input)
	if err != nil {
		t.Fatalf("parseInstalledTSV returned error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 installed rows, got %d", len(rows))
	}
	if rows[0].Target != "claude" || rows[0].Scope != "project" || rows[0].Managed != "yes" || rows[0].Source != "agent-rules" {
		t.Fatalf("unexpected managed row: %#v", rows[0])
	}
	if rows[1].Managed != "no" {
		t.Fatalf("expected unmanaged row, got %#v", rows[1])
	}
}

func TestParseInstalledUsageTSV(t *testing.T) {
	input := "source\tskill\ttarget\tscope\tproject_path\ttarget_root\tinstalled_path\tsource_ref\tsource_location\tcatalog\tcontent_hash\tinstalled_at\tupdated_at\n" +
		"agent-rules\trules-selector\tcodex\tproject\t/tmp/project\t/tmp/project/.agents/skills\t/tmp/project/.agents/skills/rules-selector\tmain\tgit@example.com:rules.git\tcatalog/skills.tsv\tabc\t2026-05-05T00:00:00Z\t2026-05-05T01:00:00Z\n"

	rows, err := parseInstalledUsageTSV(input)
	if err != nil {
		t.Fatalf("parseInstalledUsageTSV returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 usage row, got %d", len(rows))
	}
	row := rows[0]
	if row.Target != "codex" || row.Scope != "project" || row.ProjectPath != "/tmp/project" || !row.RegistryOnly {
		t.Fatalf("unexpected usage row: %#v", row)
	}
}

func TestBuildUsageSummariesGroupsRowsBySkill(t *testing.T) {
	rows := []InstalledSkill{
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "codex", Scope: "global", ProjectPath: "-", UpdatedAt: "2026-05-05T01:00:00Z"},
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "codex", Scope: "project", ProjectPath: "/tmp/project-a", UpdatedAt: "2026-05-05T02:00:00Z"},
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "claude", Scope: "project", ProjectPath: "/tmp/project-b", UpdatedAt: "2026-05-05T03:00:00Z"},
		{Source: "agent-rules", Skill: "go-project-rules", Managed: "yes", Target: "codex", Scope: "project", ProjectPath: "/tmp/project-a", UpdatedAt: "2026-05-05T00:00:00Z"},
	}

	summaries := buildUsageSummaries(rows)
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %#v", summaries)
	}
	var rules UsageSummary
	for _, summary := range summaries {
		if summary.Key == "agent-rules/rules-selector" {
			rules = summary
		}
	}
	if rules.InstallCount != 3 || rules.ProjectCount != 2 || rules.TargetCount != 3 || rules.LatestUpdated != "2026-05-05T03:00:00Z" {
		t.Fatalf("unexpected rules-selector summary: %#v", rules)
	}
}

func TestMergeInstalledUsageRowsSkipsScannedDuplicates(t *testing.T) {
	scanned := []InstalledSkill{
		{Target: "codex", Scope: "project", Skill: "rules-selector", InstalledPath: "/tmp/project/.agents/skills/rules-selector", Path: "/tmp/project/.agents/skills/rules-selector"},
	}
	usageRows := []InstalledSkill{
		{Target: "codex", Scope: "project", Skill: "rules-selector", ProjectPath: "/tmp/project", TargetRoot: "/tmp/project/.agents/skills", InstalledPath: "/tmp/project/.agents/skills/rules-selector", Path: "/tmp/project/.agents/skills/rules-selector", RegistryOnly: true},
		{Target: "claude", Scope: "project", Skill: "rules-selector", InstalledPath: "/tmp/other/.claude/skills/rules-selector", Path: "/tmp/other/.claude/skills/rules-selector", RegistryOnly: true},
	}

	rows := mergeInstalledUsageRows(scanned, usageRows)
	if len(rows) != 2 {
		t.Fatalf("expected scanned duplicate to be skipped, got %#v", rows)
	}
	if !rows[0].RegistryOnly || rows[0].ProjectPath != "/tmp/project" || rows[0].TargetRoot != "/tmp/project/.agents/skills" {
		t.Fatalf("expected scanned duplicate to be enriched from registry, got %#v", rows[0])
	}
	if !rows[1].RegistryOnly || rows[1].Target != "claude" {
		t.Fatalf("expected usage-only claude row, got %#v", rows[1])
	}
}

func TestUsageScreenShowsSummaryRows(t *testing.T) {
	rows := []InstalledSkill{
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "codex", Scope: "global", ProjectPath: "-", Path: "/tmp/global/rules-selector", UpdatedAt: "2026-05-05T01:00:00Z"},
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "codex", Scope: "project", ProjectPath: "/tmp/project-a", Path: "/tmp/project-a/.agents/skills/rules-selector", UpdatedAt: "2026-05-05T02:00:00Z"},
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "claude", Scope: "project", ProjectPath: "/tmp/project-b", Path: "/tmp/project-b/.claude/skills/rules-selector", UpdatedAt: "2026-05-05T03:00:00Z"},
	}
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 40
	m.viewMode = viewUsage
	m.usageRows = rows
	m.usageSummaries = buildUsageSummaries(rows)

	view := stripANSI(m.View())
	for _, want := range []string{
		"Usage",
		"Installs: 3",
		"Skills: 1",
		"Projects: 2",
		"agent-rules/rules-selector",
		"installs: 3",
		"projects: 2",
		"targets: 3",
		"latest: 2026-05-05T03:00:00Z",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected usage screen to contain %q, got:\n%s", want, view)
		}
	}
}

func TestEnterOpensUsageDetails(t *testing.T) {
	rows := []InstalledSkill{
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "codex", Scope: "global", ProjectPath: "-", Path: "/tmp/global/rules-selector", ContentHash: "global-hash", UpdatedAt: "2026-05-05T01:00:00Z"},
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "gemini", Scope: "project", ProjectPath: "/tmp/project-b", Path: "/tmp/project-b/.gemini/skills/rules-selector", ContentHash: "project-hash", UpdatedAt: "2026-05-05T03:00:00Z"},
	}
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 40
	m.viewMode = viewUsage
	m.usageRows = rows
	m.usageSummaries = buildUsageSummaries(rows)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.viewMode != viewUsageDetails {
		t.Fatalf("expected usage details view, got %q", m.viewMode)
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Usage details",
		"agent-rules/rules-selector",
		"Installed in: 2",
		"Projects: 1",
		"Codex global",
		"LOCAL PROJECT",
		"Gemini project",
		"Project root: /tmp/project-b",
		"hash: project-hash",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected usage details to contain %q, got:\n%s", want, view)
		}
	}
}

func TestUsageUpdateArgsAndGlobalOnlyRefusal(t *testing.T) {
	if got := usageUpdateArgsForKey("agent-rules/rules-selector"); strings.Join(got, " ") != "usage update --projects agent-rules/rules-selector" {
		t.Fatalf("unexpected usage update args: %#v", got)
	}

	m := initialModel(".")
	m.loading = false
	m.viewMode = viewUsage
	m.usageRows = []InstalledSkill{
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "codex", Scope: "global", ProjectPath: "-", Path: "/tmp/global/rules-selector"},
	}
	m.usageSummaries = buildUsageSummaries(m.usageRows)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = updated.(model)
	if cmd != nil || m.busy {
		t.Fatalf("global-only usage update should not start a command")
	}
	if !strings.Contains(m.status, "has no recorded project installs") {
		t.Fatalf("expected global-only refusal status, got %q", m.status)
	}
}

func TestUsageDetailsUpdateReturnsToDetailsAfterReload(t *testing.T) {
	rows := []InstalledSkill{
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "codex", Scope: "global", ProjectPath: "-", Path: "/tmp/global/rules-selector", UpdatedAt: "2026-05-05T01:00:00Z"},
		{Source: "agent-rules", Skill: "rules-selector", Managed: "yes", Target: "codex", Scope: "project", ProjectPath: "/tmp/project-a", Path: "/tmp/project-a/.agents/skills/rules-selector", UpdatedAt: "2026-05-05T02:00:00Z"},
	}
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 40
	m.viewMode = viewUsage
	m.usageRows = rows
	m.usageSummaries = buildUsageSummaries(rows)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = updated.(model)
	if cmd == nil || !m.busy || !m.returnToUsageDetails {
		t.Fatalf("expected usage details update command and detail-return marker")
	}

	updated, _ = m.Update(commandDoneMsg{action: "Update usage", output: "Updated project usage: updated=0 unchanged=1 skipped=0 failed=0\n"})
	m = updated.(model)
	if !m.loading || m.viewMode != viewUsageDetails {
		t.Fatalf("expected usage details to remain active during reload, got view=%q loading=%v", m.viewMode, m.loading)
	}

	updated, _ = m.Update(usageLoadedMsg{rows: rows})
	m = updated.(model)
	if m.viewMode != viewUsageDetails {
		t.Fatalf("expected usage details after reload, got %q", m.viewMode)
	}
	if !strings.Contains(m.status, "Updated project usage: updated=0 unchanged=1 skipped=0 failed=0") {
		t.Fatalf("expected usage update summary to remain visible, got %q", m.status)
	}
}

func TestInstalledScreenGroupsRowsByTargetScope(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 40
	m.viewMode = viewInstalled
	m.installedRows = []InstalledSkill{
		{Target: "codex", Scope: "global", Skill: "rules-selector", Managed: "yes", Source: "agent-rules", Path: "/tmp/codex/rules-selector"},
		{Target: "claude", Scope: "project", Skill: "manual-skill", Managed: "no", Source: "-", ProjectPath: "/tmp/project", Path: "/tmp/project/.claude/skills/manual-skill"},
		{Target: "gemini", Scope: "project", Skill: "rules-selector", Managed: "yes", Source: "agent-rules", ProjectPath: "/tmp/other-project", Path: "/tmp/other-project/.gemini/skills/rules-selector", RegistryOnly: true},
		{Target: "directory", Scope: "custom", Skill: "docs-project-rules", Managed: "yes", Source: "agent-rules", TargetRoot: "/tmp/skills", Path: "/tmp/skills/docs-project-rules"},
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Installed skills",
		"Installed: 4",
		"Skills: 3",
		"Projects: 2",
		"Managed: 3",
		"[M] managed by Skillhub",
		"[ ] unmanaged: read-only in TUI",
		"Global",
		"[M] rules-selector",
		"source: agent-rules",
		"Projects",
		"Project /tmp/project",
		"[ ] manual-skill",
		"unmanaged",
		"Project /tmp/other-project",
		"registry",
		"Custom directories",
		"/tmp/skills",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected installed screen to contain %q, got:\n%s", want, view)
		}
	}
}

func TestEnterOpensInstalledSkillDetails(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 40
	m.viewMode = viewInstalled
	m.installedRows = []InstalledSkill{
		{Target: "codex", Scope: "global", Skill: "rules-selector", Managed: "yes", Source: "agent-rules", Path: "/tmp/codex/rules-selector"},
		{Target: "gemini", Scope: "project", Skill: "rules-selector", Managed: "yes", Source: "agent-rules", ProjectPath: "/tmp/project-b", Path: "/tmp/project-b/.gemini/skills/rules-selector", RegistryOnly: true, ContentHash: "abc", UpdatedAt: "2026-05-05T01:00:00Z"},
		{Target: "claude", Scope: "project", Skill: "manual-skill", Managed: "no", Source: "-", ProjectPath: "/tmp/project-a", Path: "/tmp/project-a/.claude/skills/manual-skill"},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.viewMode != viewInstalledDetails {
		t.Fatalf("expected installed details view, got %q", m.viewMode)
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Installed skill details",
		"agent-rules/rules-selector",
		"Installed in: 2",
		"Codex global",
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
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 40
	m.viewMode = viewInstalled
	m.installedRows = []InstalledSkill{
		{Target: "codex", Scope: "global", Skill: "rules-selector", Managed: "yes", Source: "agent-rules", Path: "/tmp/codex/rules-selector"},
		{Target: "gemini", Scope: "project", Skill: "rules-selector", Managed: "yes", Source: "agent-rules", ProjectPath: "/tmp/project-b", Path: "/tmp/project-b/.gemini/skills/rules-selector", RegistryOnly: true},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)

	row, ok := m.currentInstalledDetail()
	if !ok {
		t.Fatalf("expected highlighted installed details row")
	}
	want := []string{"update", "--target", "gemini", "--scope", "project", "--project", "/tmp/project-b"}
	if got := installedUpdateArgsForRow(row, "/tmp/fallback"); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("expected details update args %#v, got %#v", want, got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = updated.(model)
	if m.viewMode != viewConfirmDelete {
		t.Fatalf("expected confirm delete, got %q", m.viewMode)
	}
	if m.pendingUninstall.ProjectPath != "/tmp/project-b" {
		t.Fatalf("expected pending uninstall to use highlighted details row, got %#v", m.pendingUninstall)
	}
}

func TestEscReturnsFromInstalledDetails(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.viewMode = viewInstalledDetails
	m.installedDetailKey = "agent-rules/rules-selector"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)
	if m.viewMode != viewInstalled {
		t.Fatalf("expected installed view after esc, got %q", m.viewMode)
	}
}

func TestInstalledUpdateAndUninstallArgs(t *testing.T) {
	row := InstalledSkill{Target: "claude", Scope: "project", Skill: "rules-selector"}
	project := "/tmp/project"

	updateWant := []string{"update", "--target", "claude", "--scope", "project", "--project", project}
	if got := installedUpdateArgsForRow(row, project); strings.Join(got, " ") != strings.Join(updateWant, " ") {
		t.Fatalf("expected update args %#v, got %#v", updateWant, got)
	}

	uninstallWant := []string{"uninstall", "rules-selector", "--target", "claude", "--scope", "project", "--project", project}
	if got := installedUninstallArgsForRow(row, project); strings.Join(got, " ") != strings.Join(uninstallWant, " ") {
		t.Fatalf("expected uninstall args %#v, got %#v", uninstallWant, got)
	}

	registryRow := InstalledSkill{Target: "gemini", Scope: "project", Skill: "rules-selector", ProjectPath: "/tmp/recorded-project"}
	registryWant := []string{"update", "--target", "gemini", "--scope", "project", "--project", "/tmp/recorded-project"}
	if got := installedUpdateArgsForRow(registryRow, project); strings.Join(got, " ") != strings.Join(registryWant, " ") {
		t.Fatalf("expected registry update args %#v, got %#v", registryWant, got)
	}

	directoryRow := InstalledSkill{Target: "directory", Scope: "custom", Skill: "rules-selector", TargetRoot: "/tmp/skills"}
	directoryWant := []string{"uninstall", "rules-selector", "--target", "directory", "--dir", "/tmp/skills"}
	if got := installedUninstallArgsForRow(directoryRow, project); strings.Join(got, " ") != strings.Join(directoryWant, " ") {
		t.Fatalf("expected directory uninstall args %#v, got %#v", directoryWant, got)
	}
}

func TestInstalledArgsUseLegacyCodexEnv(t *testing.T) {
	t.Setenv("AGENT_SKILLS_DIR", "/tmp/legacy-skills")
	row := InstalledSkill{Target: "codex", Scope: "global", Skill: "rules-selector", Path: "/tmp/legacy-skills/rules-selector"}

	if got := installedListArgs("codex", "global", "/tmp/project"); strings.Join(got, " ") != "list --tsv" {
		t.Fatalf("expected legacy list args, got %#v", got)
	}
	if got := installedUpdateArgsForRow(row, "/tmp/project"); strings.Join(got, " ") != "update" {
		t.Fatalf("expected legacy update args, got %#v", got)
	}
	if got := installedUninstallArgsForRow(row, "/tmp/project"); strings.Join(got, " ") != "uninstall rules-selector" {
		t.Fatalf("expected legacy uninstall args, got %#v", got)
	}
}

func TestParseActiveSourcesTSV(t *testing.T) {
	input := "name\ttype\tlocation\tref\tcatalog\n" +
		"agent-rules\tgit\tgit@github.com:assurrussa/agent-rules.git\tmain\tcatalog/skills.tsv\n"

	sources, err := parseSourcesTSV(input)
	if err != nil {
		t.Fatalf("parseSourcesTSV returned error: %v", err)
	}
	if len(sources) != 1 || sources[0].Name != "agent-rules" || sources[0].Catalog != "catalog/skills.tsv" {
		t.Fatalf("unexpected sources: %#v", sources)
	}
}

func TestUpdateScreenShowsCommandsWithoutRunningSelfUpdate(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 30
	m.viewMode = viewUpdate

	view := stripANSI(m.View())
	for _, want := range []string{
		"Update",
		"skillhub update",
		"skillhub update --cascade",
		"skillhub update --cascade -v",
		"skillhub installed update",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected update screen to contain %q, got:\n%s", want, view)
		}
	}
}

func TestAddSourceInputAcceptsDashboardShortcutDigits(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.viewMode = viewAddSource

	for _, key := range []string{"r", "u", "l", "e", "s", "-", "v", "2"} {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = updated.(model)
	}

	if m.viewMode != viewAddSource {
		t.Fatalf("expected to stay in add-source view, got %q", m.viewMode)
	}
	if m.sourceInput != "rules-v2" {
		t.Fatalf("expected source input to include digits, got %q", m.sourceInput)
	}
}

func TestInstalledCommandSummarySurvivesReload(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.viewMode = viewInstalled

	updated, _ := m.Update(commandDoneMsg{
		action: "Update installed",
		output: "Summary codex/global: updated=0 unchanged=1 skipped=0 failed=0\nUpdated: 0, unchanged: 1, skipped: 0, failed: 0\n",
	})
	m = updated.(model)
	if !m.loading {
		t.Fatalf("expected installed reload after update")
	}

	updated, _ = m.Update(installedLoadedMsg{rows: []InstalledSkill{
		{Target: "codex", Scope: "global", Skill: "rules-selector", Managed: "yes", Source: "agent-rules", Path: "/tmp/skills/rules-selector"},
	}})
	m = updated.(model)

	if !strings.Contains(m.status, "Updated: 0, unchanged: 1, skipped: 0, failed: 0") {
		t.Fatalf("expected update summary to remain visible, got %q", m.status)
	}
}

func TestQuestionMarkOpensHelpOverlay(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 30

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(model)
	if m.viewMode != viewHelp {
		t.Fatalf("expected help view, got %q", m.viewMode)
	}
	view := stripANSI(m.View())
	for _, want := range []string{
		"Help",
		"1/2/3/4/5",
		"left/right",
		"u           update",
		"x           uninstall",
		"[M]         managed by Skillhub",
		"[ ]         unmanaged local skill",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected help overlay to contain %q, got:\n%s", want, view)
		}
	}
}

func TestLeftRightSwitchDashboardSections(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 30
	m.viewMode = viewSkills

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(model)
	if m.viewMode != viewInstalled || !m.loading {
		t.Fatalf("expected right from skills to load installed section, got view=%q loading=%v", m.viewMode, m.loading)
	}

	m.loading = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(model)
	if m.viewMode != viewUsage || !m.loading {
		t.Fatalf("expected right from installed to load usage section, got view=%q loading=%v", m.viewMode, m.loading)
	}

	m.loading = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(model)
	if m.viewMode != viewInstalled || !m.loading {
		t.Fatalf("expected left from usage to load installed section, got view=%q loading=%v", m.viewMode, m.loading)
	}
}

func TestLeftRightDoesNotLeaveInstallTargetPicker(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.viewMode = viewTargets
	m.targetPurpose = "install"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(model)
	if m.viewMode != viewTargets || m.loading {
		t.Fatalf("expected install target picker to ignore right arrow, got view=%q loading=%v", m.viewMode, m.loading)
	}
}

func TestSkillSelectionUsesSoftActiveAndGreenSelectedStyles(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 30
	m.skills = []Skill{
		{Source: "agent-rules", Name: "go-project-rules", Category: "go", Triggers: "go,golang", Description: "Go project rules"},
	}
	m.applyFilter()
	m.toggleCurrent()

	view := m.View()
	if got := activeRowStyle.GetBackground(); got != (lipgloss.NoColor{}) {
		t.Fatalf("active row should not use a strong background, got %#v", got)
	}
	if got := selectedRowStyle.GetForeground(); got != lipgloss.Color("42") {
		t.Fatalf("selected row should use green foreground, got %#v", got)
	}
	if !strings.Contains(stripANSI(view), "[✓] go-project-rules") {
		t.Fatalf("selected row should keep an explicit selected marker, got:\n%s", stripANSI(view))
	}
}

func TestViewGroupsSkillsByCategoryTree(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 30
	m.skills = []Skill{
		{Source: "agent-rules", Name: "go-project-rules", Category: "go", Description: "Go project rules"},
		{Source: "agent-rules", Name: "docs-project-rules", Category: "documentation", Description: "Documentation rules"},
	}
	m.applyFilter()

	view := stripANSI(m.View())
	for _, want := range []string{
		"• documentation",
		"└─ [ ] docs-project-rules",
		"• go",
		"└─ [ ] go-project-rules",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected grouped tree view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestParseDefaultSourcesTSV(t *testing.T) {
	input := "name\ttype\tlocation\tref\tcatalog\n" +
		"agent-rules\tgit\tgit@github.com:assurrussa/agent-rules.git\tmain\tcatalog/skills.tsv\n"

	sources, err := parseDefaultSourcesTSV(input)
	if err != nil {
		t.Fatalf("parseDefaultSourcesTSV returned error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Name != "agent-rules" || sources[0].Type != "git" {
		t.Fatalf("unexpected source preset: %#v", sources[0])
	}
}

func TestParseTargetsTSV(t *testing.T) {
	input := "id\tlabel\tstatus\tadapter\tdescription\n" +
		"claude\tClaude\tsupported\tskill-dir\tClaude skills\n"

	targets, err := parseTargetsTSV(input)
	if err != nil {
		t.Fatalf("parseTargetsTSV returned error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].ID != "claude" || targets[0].Adapter != "skill-dir" {
		t.Fatalf("unexpected target: %#v", targets[0])
	}
}

func TestParseTargetDetectionsTSV(t *testing.T) {
	input := "target\tscope\tstatus\tpath\texists\tskills\tmanaged\n" +
		"claude\tproject\tsupported\t/tmp/project/.claude/skills\tyes\t3\t2\n"

	detections, err := parseTargetDetectionsTSV(input)
	if err != nil {
		t.Fatalf("parseTargetDetectionsTSV returned error: %v", err)
	}
	if len(detections) != 1 {
		t.Fatalf("expected 1 detection, got %d", len(detections))
	}
	got := detections[0]
	if got.Target != "claude" || got.Scope != "project" || got.Exists != "yes" || got.Skills != "3" || got.Managed != "2" {
		t.Fatalf("unexpected detection: %#v", got)
	}
}

func TestInstallOpensTargetSelection(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 80
	m.skills = []Skill{
		{Source: "agent-rules", Name: "go-project-rules", Category: "go", Description: "Go project rules"},
	}
	m.applyFilter()
	m.toggleCurrent()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = updated.(model)
	if !m.loading {
		t.Fatalf("expected install to load targets")
	}

	updated, _ = m.Update(targetsLoadedMsg{targets: []Target{
		{ID: "codex", Label: "Codex", Status: "supported", Adapter: "skill-dir", Description: "Codex skills"},
		{ID: "claude", Label: "Claude", Status: "supported", Adapter: "skill-dir", Description: "Claude skills"},
		{ID: "gemini", Label: "Gemini", Status: "supported", Adapter: "skill-dir", Description: "Gemini skills"},
		{ID: "opencode", Label: "OpenCode", Status: "supported", Adapter: "skill-dir", Description: "OpenCode skills"},
		{ID: "cursor", Label: "Cursor", Status: "planned", Adapter: "planned", Description: "Planned"},
	}})
	m = updated.(model)
	if m.viewMode != viewTargets {
		t.Fatalf("expected target selection view, got %q", m.viewMode)
	}
	if !m.selectedTargets["codex:global"] {
		t.Fatalf("expected Codex global to be selected by default")
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Install targets",
		"[✓] Codex global",
		"[ ] Claude global",
		"[ ] Gemini global",
		"[ ] OpenCode global",
		"[-] Cursor",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected target view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestTargetSelectionShowsDetectionStats(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 30
	m.projectDir = "/tmp/project"

	updated, _ := m.Update(targetsLoadedMsg{
		targets: []Target{
			{ID: "claude", Label: "Claude", Status: "supported", Adapter: "skill-dir", Description: "Claude skills"},
		},
		detections: []TargetDetection{
			{Target: "claude", Scope: "global", Status: "supported", Path: "/home/me/.claude/skills", Exists: "yes", Skills: "4", Managed: "2"},
			{Target: "claude", Scope: "project", Status: "supported", Path: "/tmp/project/.claude/skills", Exists: "no", Skills: "0", Managed: "0"},
		},
	})
	m = updated.(model)

	view := stripANSI(m.View())
	for _, want := range []string{
		"Claude global",
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
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 40
	m.selected = map[string]bool{
		"agent-rules/go-project-rules": true,
		"agent-rules/rules-selector":   true,
	}
	m.targetChoices = []InstallTargetChoice{
		{Key: "codex:global", Target: "codex", Label: "Codex global", Scope: "global", Status: "supported", Path: "/tmp/skills", Supported: true},
		{Key: "claude:project", Target: "claude", Label: "Claude project", Scope: "project", Status: "supported", Path: "/tmp/project/.claude/skills", Supported: true},
	}
	m.selectedTargets = map[string]bool{
		"codex:global": true,
	}

	updated, _ := m.installToSelectedTargets()
	m = updated.(model)
	if !m.busy {
		t.Fatalf("expected install to mark TUI busy")
	}

	updated, _ = m.Update(commandDoneMsg{action: "Install"})
	m = updated.(model)
	if m.viewMode != viewInstallResult {
		t.Fatalf("expected install result view, got %q", m.viewMode)
	}

	view := stripANSI(m.View())
	for _, want := range []string{
		"Install complete",
		"Codex global",
		"/tmp/skills",
		"/tmp/skills/go-project-rules",
		"/tmp/skills/rules-selector",
		"enter back",
		"q quit",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected install result view to contain %q, got:\n%s", want, view)
		}
	}
}

func TestInstallResultCanReturnToTargetsOrSkills(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.viewMode = viewInstallResult
	m.status = "Install complete."

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = updated.(model)
	if m.viewMode != viewTargets {
		t.Fatalf("expected t to return to targets, got %q", m.viewMode)
	}

	m.viewMode = viewInstallResult
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.viewMode != viewSkills {
		t.Fatalf("expected enter to return to skills, got %q", m.viewMode)
	}
}

func TestTargetSelectionOnlyTogglesSupportedTargets(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.targetChoices = []InstallTargetChoice{
		{Key: "cursor", Label: "Cursor", Supported: false},
		{Key: "claude:global", Label: "Claude global", Target: "claude", Scope: "global", Supported: true},
	}
	m.selectedTargets = map[string]bool{}
	m.targetCursor = 0

	m.toggleCurrentTarget()
	if m.selectedTargets["cursor"] {
		t.Fatalf("planned targets should not be selectable")
	}

	m.targetCursor = 1
	m.toggleCurrentTarget()
	if !m.selectedTargets["claude:global"] {
		t.Fatalf("supported target should be selectable")
	}
}

func TestTargetChoicesShowSupportedBeforePlanned(t *testing.T) {
	m := initialModel(".")
	choices := m.buildTargetChoices([]Target{
		{ID: "cursor", Label: "Cursor", Status: "planned", Adapter: "planned", Description: "Planned"},
		{ID: "gemini", Label: "Gemini", Status: "supported", Adapter: "skill-dir", Description: "Gemini skills"},
		{ID: "copilot", Label: "Copilot", Status: "planned", Adapter: "planned", Description: "Planned"},
		{ID: "codex", Label: "Codex", Status: "supported", Adapter: "skill-dir", Description: "Codex skills"},
	})

	got := []string{}
	for _, choice := range choices {
		got = append(got, choice.Key)
	}
	wantPrefix := []string{"gemini:global", "gemini:project", "codex:global", "codex:project"}
	if strings.Join(got[:len(wantPrefix)], " ") != strings.Join(wantPrefix, " ") {
		t.Fatalf("expected supported choices first, got %#v", got)
	}
	if got[len(got)-2] != "cursor" || got[len(got)-1] != "copilot" {
		t.Fatalf("expected planned choices last, got %#v", got)
	}
}

func TestTargetSelectionScrollsToKeepCursorVisible(t *testing.T) {
	m := initialModel(".")
	m.loading = false
	m.width = 120
	m.height = 24
	m.viewMode = viewTargets
	m.selectedTargets = map[string]bool{}
	m.targetChoices = []InstallTargetChoice{
		{Key: "agent-00", Label: "Agent 00", Status: "supported", Description: "Agent 00 skills", Supported: true},
		{Key: "agent-01", Label: "Agent 01", Status: "supported", Description: "Agent 01 skills", Supported: true},
		{Key: "agent-02", Label: "Agent 02", Status: "supported", Description: "Agent 02 skills", Supported: true},
		{Key: "agent-03", Label: "Agent 03", Status: "supported", Description: "Agent 03 skills", Supported: true},
		{Key: "agent-04", Label: "Agent 04", Status: "supported", Description: "Agent 04 skills", Supported: true},
		{Key: "agent-05", Label: "Agent 05", Status: "supported", Description: "Agent 05 skills", Supported: true},
		{Key: "agent-06", Label: "Agent 06", Status: "supported", Description: "Agent 06 skills", Supported: true},
	}

	for i := 0; i < 5; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "› [ ] Agent 05") {
		t.Fatalf("expected cursor row to stay visible, got:\n%s", view)
	}
	if strings.Contains(view, "Agent 00") {
		t.Fatalf("expected scrolled target view to hide the first row, got:\n%s", view)
	}
}

func TestSkillVisibleCountLeavesRoomForMultilineCards(t *testing.T) {
	m := initialModel(".")
	m.height = 30

	if got := m.visibleCount(); got != 3 {
		t.Fatalf("expected 3 visible skill cards at height 30, got %d", got)
	}
}

func TestCustomSourceInputModeCapturesText(t *testing.T) {
	m := initialModel(".")
	m.loading = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = updated.(model)
	if m.viewMode != viewAddSource {
		t.Fatalf("expected add source mode, got %q", m.viewMode)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("../agent-rules")})
	m = updated.(model)
	if m.sourceInput != "../agent-rules" {
		t.Fatalf("expected source input to be captured, got %q", m.sourceInput)
	}
}

func TestInstallTargetPathUsesProjectScope(t *testing.T) {
	m := initialModel(".")
	m.installScope = "project"
	m.projectDir = "/tmp/example-project"

	want := filepath.Join("/tmp/example-project", ".agents", "skills")
	if got := m.installTargetPath(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestInstallTargetPathUsesLegacyGlobalEnv(t *testing.T) {
	t.Setenv("AGENT_SKILLS_DIR", "/tmp/legacy-skills")
	m := initialModel(".")
	m.installScope = "global"

	if got := m.installTargetPath(); got != "/tmp/legacy-skills" {
		t.Fatalf("expected legacy target, got %q", got)
	}
}

func TestTargetChoiceUsesLegacyCodexEnv(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "rules-selector")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, ".skillhub.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	t.Setenv("AGENT_SKILLS_DIR", tmp)

	m := initialModel(".")
	m.targetStats = map[string]TargetDetection{
		"codex:global": {Target: "codex", Scope: "global", Path: "/home/me/.agents/skills", Exists: "yes", Skills: "9", Managed: "0"},
	}
	choices := m.buildTargetChoices([]Target{
		{ID: "codex", Label: "Codex", Status: "supported", Adapter: "skill-dir", Description: "Codex skills"},
	})

	if len(choices) == 0 {
		t.Fatalf("expected codex target choices")
	}
	got := choices[0]
	if got.Key != "codex:global" {
		t.Fatalf("expected first choice to be codex global, got %q", got.Key)
	}
	if got.Path != tmp || got.Exists != "yes" || got.Skills != "1" || got.Managed != "1" {
		t.Fatalf("expected legacy codex env stats, got %#v", got)
	}
}

func TestInstallArgsUseLegacyCodexEnv(t *testing.T) {
	t.Setenv("AGENT_SKILLS_DIR", "/tmp/legacy-skills")

	args := installArgsForTargetChoice(
		InstallTargetChoice{Target: "codex", Scope: "global"},
		"/tmp/project",
		[]string{"rules-selector"},
	)
	want := []string{"install", "rules-selector"}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Fatalf("expected legacy install args %#v, got %#v", want, args)
	}
}

func TestNativeTargetPaths(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("OPENCODE_CONFIG_DIR", "/tmp/opencode-config")

	cases := []struct {
		target string
		scope  string
		want   string
	}{
		{"claude", "global", filepath.Join("/home/tester", ".claude", "skills")},
		{"claude", "project", filepath.Join("/tmp/project", ".claude", "skills")},
		{"gemini", "global", filepath.Join("/home/tester", ".gemini", "skills")},
		{"gemini", "project", filepath.Join("/tmp/project", ".gemini", "skills")},
		{"opencode", "global", filepath.Join("/tmp/opencode-config", "skills")},
		{"opencode", "project", filepath.Join("/tmp/project", ".opencode", "skills")},
	}
	for _, tc := range cases {
		if got := targetSkillRoot(tc.target, tc.scope, "/tmp/project"); got != tc.want {
			t.Fatalf("targetSkillRoot(%q, %q) = %q, want %q", tc.target, tc.scope, got, tc.want)
		}
	}
}
