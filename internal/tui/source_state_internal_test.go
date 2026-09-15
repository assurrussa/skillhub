package tui

import (
	"errors"
	"testing"

	"github.com/assurrussa/skillhub/internal/core"
)

func TestCompleteSuccessfulSourceRenameRewritesQueuedSkills(t *testing.T) {
	m := model{
		selected: map[string]bool{
			"alpha/one": true,
			"alpha/two": true,
			"other/one": true,
		},
		pendingRenameSource: SourcePreset{Name: "alpha"},
		sourceRenameInput:   "beta",
	}

	m.completeSuccessfulSourceAction("Rename source")

	if m.selected["alpha/one"] || m.selected["alpha/two"] {
		t.Fatalf("old source keys remained in queue: %#v", m.selected)
	}
	if !m.selected["beta/one"] || !m.selected["beta/two"] || !m.selected["other/one"] {
		t.Fatalf("renamed queue keys are incomplete: %#v", m.selected)
	}
	if m.pendingRenameSource.Name != "" || m.sourceRenameInput != "" {
		t.Fatalf("successful rename state was not cleared: %#v %q", m.pendingRenameSource, m.sourceRenameInput)
	}
}

func TestCompleteSuccessfulSourceRenameRewritesRootSkillWithSummary(t *testing.T) {
	m := model{
		selected: map[string]bool{
			"alpha/alpha": true,
			"alpha/other": true,
			"other/one":   true,
		},
		pendingRenameSource: SourcePreset{Name: "alpha"},
		sourceRenameInput:   "beta",
	}

	summary := core.SourceRenameSummary{
		OldName:         "alpha",
		NewName:         "beta",
		RenamedSkillOld: "alpha",
		RenamedSkillNew: "beta",
	}

	m.completeSuccessfulSourceAction("Rename source", summary)

	if m.selected["alpha/alpha"] || m.selected["alpha/other"] || m.selected["beta/alpha"] {
		t.Fatalf("stale source or unmigrated skill keys remained in queue: %#v", m.selected)
	}
	if !m.selected["beta/beta"] {
		t.Fatalf("root skill key was not rewritten to beta/beta: %#v", m.selected)
	}
	if !m.selected["beta/other"] {
		t.Fatalf("non-root skill key was not rewritten to beta/other: %#v", m.selected)
	}
	if !m.selected["other/one"] {
		t.Fatalf("unrelated source key was modified: %#v", m.selected)
	}
}

func TestCompleteSuccessfulSourceRenameRewritesRootSkillWithoutSummary(t *testing.T) {
	m := model{
		selected: map[string]bool{
			"alpha/alpha": true,
			"alpha/other": true,
			"other/one":   true,
		},
		pendingRenameSource: SourcePreset{Name: "alpha"},
		sourceRenameInput:   "beta",
	}

	m.completeSuccessfulSourceAction("Rename source")

	if m.selected["alpha/alpha"] || m.selected["beta/alpha"] {
		t.Fatalf("stale root skill keys remained in queue: %#v", m.selected)
	}
	if !m.selected["beta/beta"] {
		t.Fatalf("root skill key was not rewritten to beta/beta in fallback: %#v", m.selected)
	}
	if !m.selected["beta/other"] || !m.selected["other/one"] {
		t.Fatalf("expected other keys preserved: %#v", m.selected)
	}
}

func TestCompleteSuccessfulSourceRenamePreservesSkillWhenSummaryHasNoSkillRename(t *testing.T) {
	m := model{
		selected: map[string]bool{
			"alpha/alpha": true,
			"other/one":   true,
		},
		pendingRenameSource: SourcePreset{Name: "alpha"},
		sourceRenameInput:   "beta",
	}

	summary := core.SourceRenameSummary{
		OldName: "alpha",
		NewName: "beta",
		// RenamedSkillOld and RenamedSkillNew are empty
	}

	m.completeSuccessfulSourceAction("Rename source", summary)

	if m.selected["alpha/alpha"] || m.selected["beta/beta"] {
		t.Fatalf("unexpected keys in queue: %#v", m.selected)
	}
	if !m.selected["beta/alpha"] {
		t.Fatalf("expected skill named alpha to stay beta/alpha when no root migration occurred: %#v", m.selected)
	}
}

func TestCompleteSuccessfulSourceRemovalDropsQueuedSkills(t *testing.T) {
	m := model{
		selected: map[string]bool{
			"alpha/one": true,
			"alpha/two": true,
			"other/one": true,
		},
		pendingRemoveSource: SourcePreset{Name: "alpha"},
	}

	m.completeSuccessfulSourceAction("Remove source")

	if m.selected["alpha/one"] || m.selected["alpha/two"] {
		t.Fatalf("removed source keys remained in queue: %#v", m.selected)
	}
	if !m.selected["other/one"] {
		t.Fatalf("unrelated queue key was removed: %#v", m.selected)
	}
	if m.pendingRemoveSource.Name != "" {
		t.Fatalf("successful removal state was not cleared: %#v", m.pendingRemoveSource)
	}
}

func TestRenameCommandFailureKeepsEditableState(t *testing.T) {
	m := model{
		busy:                    true,
		viewMode:                viewRenameSource,
		pendingRenameSource:     SourcePreset{Name: "alpha"},
		pendingRenameSourceDeps: 2,
		sourceRenameInput:       "beta",
	}

	updated, _ := m.updateCommandDone(commandDoneMsg{
		action: "Rename source",
		err:    errors.New("source already exists: beta"),
	})
	next := updated.(model)

	if next.viewMode != viewRenameSource {
		t.Fatalf("expected rename form to remain open, got %q", next.viewMode)
	}
	if next.pendingRenameSource.Name != "alpha" || next.sourceRenameInput != "beta" {
		t.Fatalf("rename state was lost after error: %#v %q", next.pendingRenameSource, next.sourceRenameInput)
	}
	if next.pendingRenameSourceDeps != 2 {
		t.Fatalf("dependency count was lost after error: %d", next.pendingRenameSourceDeps)
	}
	if next.busy {
		t.Fatal("rename form remained busy after command failure")
	}
}

func TestRemoveCommandFailureKeepsConfirmationState(t *testing.T) {
	m := model{
		busy:                    true,
		viewMode:                viewConfirmRemoveSource,
		pendingRemoveSource:     SourcePreset{Name: "alpha"},
		pendingRemoveSourceDeps: 3,
	}

	updated, _ := m.updateCommandDone(commandDoneMsg{
		action: "Remove source",
		err:    errors.New("cannot write sources.tsv"),
	})
	next := updated.(model)

	if next.viewMode != viewConfirmRemoveSource {
		t.Fatalf("expected remove confirmation to remain open, got %q", next.viewMode)
	}
	if next.pendingRemoveSource.Name != "alpha" || next.pendingRemoveSourceDeps != 3 {
		t.Fatalf("remove state was lost after error: %#v deps=%d", next.pendingRemoveSource, next.pendingRemoveSourceDeps)
	}
	if next.busy {
		t.Fatal("remove confirmation remained busy after command failure")
	}
}
