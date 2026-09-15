package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/assurrussa/skillhub/internal/core"
)

func (m *model) completeSuccessfulSourceAction(action string, summaries ...core.SourceRenameSummary) {
	switch action {
	case actionRenameSource:
		oldName := strings.TrimSpace(m.pendingRenameSource.Name)
		newName := strings.ToLower(strings.TrimSpace(m.sourceRenameInput))
		if oldName != "" && newName != "" && oldName != newName {
			if len(summaries) > 0 {
				m.renameSelectedSource(oldName, newName, summaries[0].RenamedSkillOld, summaries[0].RenamedSkillNew)
			} else {
				m.renameSelectedSource(oldName, newName, oldName, newName)
			}
		}
		m.pendingRenameSource = SourcePreset{}
		m.pendingRenameSourceDeps = 0
		m.sourceRenameInput = ""
	case actionRemoveSource:
		name := strings.TrimSpace(m.pendingRemoveSource.Name)
		if name != "" {
			m.removeSelectedSource(name)
		}
		m.pendingRemoveSource = SourcePreset{}
		m.pendingRemoveSourceDeps = 0
	}
}

func (m model) sourceDependencyCount(sourceName string) (int, error) {
	count := 0
	for _, row := range m.installedRows {
		if row.Source == sourceName {
			count++
		}
	}
	if count > 0 {
		return count, nil
	}
	backend, err := newBackend(m.repoRoot)
	if err != nil {
		return 0, err
	}
	deps, err := backend.SourceDependencies(sourceName)
	if err != nil {
		return 0, err
	}
	return len(deps), nil
}

// updateSourcesKeySafe mirrors the Sources key flow but refuses rename/remove
// when dependency state cannot be read. Destructive actions must never render
// an unknown dependency count as zero.
func (m model) updateSourcesKeySafe(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "x":
		source, ok := m.currentSource()
		if !ok {
			m.status = statusNoSourceSelected
			return m, nil
		}
		deps, err := m.sourceDependencyCount(source.Name)
		if err != nil {
			m.status = "Cannot inspect source dependencies: " + err.Error()
			return m, nil
		}
		m.pendingRemoveSource = source
		m.pendingRemoveSourceDeps = deps
		m.viewMode = viewConfirmRemoveSource
		m.status = "Confirm source removal."
		return m, nil
	case "e", "R":
		source, ok := m.currentSource()
		if !ok {
			m.status = statusNoSourceSelected
			return m, nil
		}
		deps, err := m.sourceDependencyCount(source.Name)
		if err != nil {
			m.status = "Cannot inspect source dependencies: " + err.Error()
			return m, nil
		}
		m.pendingRenameSource = source
		m.pendingRenameSourceDeps = deps
		m.sourceRenameInput = source.Name
		m.viewMode = viewRenameSource
		m.status = "Enter new name for source " + source.Name + "."
		return m, nil
	default:
		return m.updateSourcesKey(msg)
	}
}

func (m *model) renameSelectedSource(oldName, newName, oldSkill, newSkill string) {
	if len(m.selected) == 0 {
		return
	}
	oldPrefix := oldName + "/"
	newPrefix := newName + "/"
	for key := range m.selected {
		if key == oldName {
			delete(m.selected, key)
			if newSkill != "" {
				m.selected[newSkill] = true
			} else {
				m.selected[newName] = true
			}
			continue
		}
		if !strings.HasPrefix(key, oldPrefix) {
			continue
		}
		skill := strings.TrimPrefix(key, oldPrefix)
		delete(m.selected, key)
		if oldSkill != "" && newSkill != "" && skill == oldSkill {
			m.selected[newPrefix+newSkill] = true
		} else {
			m.selected[newPrefix+skill] = true
		}
	}
}

func (m *model) removeSelectedSource(name string) {
	if len(m.selected) == 0 {
		return
	}
	prefix := name + "/"
	for key := range m.selected {
		if strings.HasPrefix(key, prefix) {
			delete(m.selected, key)
		}
	}
}
