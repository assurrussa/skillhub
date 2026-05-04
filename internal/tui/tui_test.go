package tui

import "testing"

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
		{Name: "go-project-rules", Category: "go", Triggers: "go,golang", Description: "Go project rules"},
		{Name: "docs-project-rules", Category: "documentation", Triggers: "docs", Description: "Documentation rules"},
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
