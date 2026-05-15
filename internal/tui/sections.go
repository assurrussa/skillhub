package tui

import tea "github.com/charmbracelet/bubbletea"

type skillsSectionModel struct {
	root model
}

type installedSectionModel struct {
	root model
}

type targetsSectionModel struct {
	root model
}

type sourcesSectionModel struct {
	root model
}

func (m model) skillsSection() skillsSectionModel {
	return skillsSectionModel{root: m}
}

func (m model) installedSectionModel() installedSectionModel {
	return installedSectionModel{root: m}
}

func (m model) targetsSection() targetsSectionModel {
	return targetsSectionModel{root: m}
}

func (m model) sourcesSection() sourcesSectionModel {
	return sourcesSectionModel{root: m}
}

func (s skillsSectionModel) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return s.root.updateSkillsKey(msg)
}

func (s skillsSectionModel) updateDetailsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return s.root.updateDetailsKey(msg)
}

func (s skillsSectionModel) content(width int) string {
	return s.root.skillsContent(width)
}

func (s skillsSectionModel) detailsContent(width int) string {
	return s.root.detailsContent(width)
}

func (s installedSectionModel) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return s.root.updateInstalledKey(msg)
}

func (s installedSectionModel) updateDetailsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return s.root.updateInstalledDetailsKey(msg)
}

func (s installedSectionModel) content(width int) string {
	return s.root.installedContent(width)
}

func (s installedSectionModel) detailsContent(width int) string {
	return s.root.installedDetailsContent(width)
}

func (s targetsSectionModel) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return s.root.updateTargetsKey(msg)
}

func (s targetsSectionModel) content(width int) string {
	return s.root.targetsContent(width)
}

func (s targetsSectionModel) panelTitle() string {
	return s.root.targetsPanelTitle()
}

func (s sourcesSectionModel) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return s.root.updateSourcesKey(msg)
}

func (s sourcesSectionModel) updateDefaultsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return s.root.updateDefaultsKey(msg)
}

func (s sourcesSectionModel) updateAddKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return s.root.updateAddSourceKey(msg)
}

func (s sourcesSectionModel) content(width int) string {
	return s.root.sourcesContent(width)
}

func (s sourcesSectionModel) defaultsContent(width int) string {
	return s.root.defaultsContent(width)
}

func (s sourcesSectionModel) addContent(width int) string {
	return s.root.addSourceContent(width)
}
