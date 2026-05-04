package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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
