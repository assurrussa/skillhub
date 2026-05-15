package tui

import tea "github.com/charmbracelet/bubbletea"

func Run(repoRoot string) error {
	p := tea.NewProgram(initialModel(repoRoot), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func initialModel(repoRoot string) model {
	return model{
		repoRoot:         repoRoot,
		selected:         map[string]bool{},
		loading:          true,
		loadingSkills:    true,
		loadingInstalled: true,
		status:           statusLoadingInstalled,
		viewMode:         viewInstalled,

		projectDir: callerCwd(),
		install: installFlowState{
			scope:           scopeProject,
			scopeCursor:     0,
			selectedTargets: map[string]bool{},
		},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadInstalled(m.repoRoot), loadSkills(m.repoRoot))
}
