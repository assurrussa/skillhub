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
