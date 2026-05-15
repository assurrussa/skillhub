package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type installFlowModel struct {
	root  model
	state installFlowState
}

func (m model) installFlow() installFlowModel {
	return installFlowModel{root: m, state: m.install}
}

func (f installFlowModel) updateScopeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return f.commit(), tea.Quit
	case keyEsc, "b":
		m := f.commit()
		m.viewMode = viewSkills
		m.status = "Install cancelled."
		return m, nil
	case "up", "k", keyDown, "j", " ":
		if f.state.scopeCursor == 0 {
			f.state.scopeCursor = 1
			f.state.scope = scopeGlobal
		} else {
			f.state.scopeCursor = 0
			f.state.scope = scopeProject
		}
		f.state.selectedTargets = map[string]bool{}
		m := f.commit()
		m.status = "Install scope: " + installScopeDisplay(f.state.scope) + "."
		return m, nil
	case "1":
		f.state.scopeCursor = 0
		f.state.scope = scopeProject
		f.state.selectedTargets = map[string]bool{}
		m := f.commit()
		m.status = "Install scope: Project."
		return m, nil
	case "2":
		f.state.scopeCursor = 1
		f.state.scope = scopeGlobal
		f.state.selectedTargets = map[string]bool{}
		m := f.commit()
		m.status = "Install scope: User."
		return m, nil
	case keyEnter:
		return f.commit().openTargetSelection()
	default:
		return f.commit(), nil
	}
}

func (f installFlowModel) updateConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return f.commit(), tea.Quit
	case keyEsc, "b", "n":
		m := f.commit()
		m.viewMode = viewTargets
		m.status = "Returned to install agents."
		return m, nil
	case keyEnter, "y", "i":
		return f.commit().installToSelectedTargets()
	default:
		return f.commit(), nil
	}
}

func (f installFlowModel) updateResultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return f.commit(), tea.Quit
	case "t":
		m := f.commit()
		m.viewMode = viewTargets
		m.status = "Returned to install targets."
		return m, nil
	case keyEsc, keyEnter, "b":
		m := f.commit()
		m.viewMode = viewInstalled
		m.status = "Returned to installed skills."
		return m, nil
	default:
		return f.commit(), nil
	}
}

func (f installFlowModel) updateFailedProgressKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return f.commit(), tea.Quit
	case keyEsc, keyEnter, "b":
		f.state.progress = installProgressState{}
		m := f.commit()
		m.viewMode = viewTargets
		m.status = "Returned to install agents."
		return m, nil
	default:
		return f.commit(), nil
	}
}

func (f installFlowModel) updateTargetsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m := f.commit()
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case keyEsc, "b":
		m.viewMode = viewInstallScope
		m.status = "Returned to install scope."
		return m, nil
	case "up", "k":
		if m.targetCursor > 0 {
			m.targetCursor--
			m.ensureTargetCursorVisible()
		}
		return m, nil
	case keyDown, "j":
		if m.targetCursor < len(m.install.targetChoices)-1 {
			m.targetCursor++
			m.ensureTargetCursorVisible()
		}
		return m, nil
	case " ":
		m.toggleCurrentTarget()
		return m, nil
	case "a":
		if m.install.selectedTargets == nil {
			m.install.selectedTargets = map[string]bool{}
		}
		for _, choice := range m.install.targetChoices {
			if choice.Supported {
				m.install.selectedTargets[choice.Key] = true
			}
		}
		m.status = fmt.Sprintf("Selected %d supported target option(s).", len(m.selectedTargetChoices()))
		return m, nil
	case "c":
		m.install.selectedTargets = map[string]bool{}
		m.status = "Target selection cleared."
		return m, nil
	case keyEnter, "i":
		if len(m.selectedTargetChoices()) == 0 {
			m.status = statusSelectTarget
			return m, nil
		}
		m.viewMode = viewInstallConfirm
		m.status = "Review install plan."
		return m, nil
	case "r":
		m.loading = true
		m.status = "Reloading targets..."
		return m, loadTargets(m.repoRoot)
	default:
		return m, nil
	}
}

func (f installFlowModel) commit() model {
	m := f.root
	m.install = f.state
	return m
}

func (f installFlowModel) scopeContent(width int) string {
	return f.root.installScopeContent(width)
}

func (f installFlowModel) confirmContent(width int) string {
	return f.root.installConfirmContent(width)
}

func (f installFlowModel) resultContent(width int) string {
	return f.root.installResultContent(width)
}

func (f installFlowModel) progressContent(width int) string {
	return f.root.installProgressContent(width)
}
