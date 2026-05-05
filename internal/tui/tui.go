package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Skill struct {
	Source      string
	Name        string
	Category    string
	Triggers    string
	Description string
}

type SourcePreset struct {
	Name     string
	Type     string
	Location string
	Ref      string
	Catalog  string
}

type Target struct {
	ID          string
	Label       string
	Status      string
	Adapter     string
	Description string
}

type TargetDetection struct {
	Target  string
	Scope   string
	Status  string
	Path    string
	Exists  string
	Skills  string
	Managed string
}

type InstallTargetChoice struct {
	Key         string
	Target      string
	Label       string
	Scope       string
	Status      string
	Path        string
	Description string
	Supported   bool
	Exists      string
	Skills      string
	Managed     string
}

const (
	viewSkills        = "skills"
	viewDefaults      = "defaults"
	viewAddSource     = "add-source"
	viewDetails       = "details"
	viewTargets       = "targets"
	viewInstallResult = "install-result"
)

type InstallResult struct {
	SkillNames []string
	Targets    []InstallTargetResult
}

type InstallTargetResult struct {
	Label      string
	Target     string
	Scope      string
	Root       string
	SkillPaths []string
}

type model struct {
	repoRoot string

	skills      []Skill
	filtered    []int
	selected    map[string]bool
	defaults    []SourcePreset
	targets     []Target
	targetStats map[string]TargetDetection

	cursor         int
	defaultCursor  int
	targetCursor   int
	offset         int
	targetOffset   int
	width          int
	height         int
	search         string
	sourceInput    string
	searchMode     bool
	loading        bool
	busy           bool
	noSources      bool
	status         string
	viewMode       string
	reloadOnFinish bool

	installScope    string
	projectDir      string
	targetChoices   []InstallTargetChoice
	selectedTargets map[string]bool
	pendingInstall  InstallResult
	installResult   InstallResult
}

type skillsLoadedMsg struct {
	skills []Skill
	err    error
}

type defaultsLoadedMsg struct {
	defaults []SourcePreset
	err      error
}

type targetsLoadedMsg struct {
	targets    []Target
	detections []TargetDetection
	err        error
}

type commandDoneMsg struct {
	action string
	output string
	err    error
}

var (
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("81"))
	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(1, 2)
	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229"))
	activeRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)
	selectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))
	categoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Bold(true)
	treeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63"))
	checkboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))
	checkedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)
	disabledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
	badgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("24")).
			Padding(0, 1)
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
)

func Run(repoRoot string) error {
	p := tea.NewProgram(initialModel(repoRoot), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func initialModel(repoRoot string) model {
	return model{
		repoRoot: repoRoot,
		selected: map[string]bool{},
		loading:  true,
		status:   "Loading catalog...",
		viewMode: viewSkills,

		installScope:    "global",
		projectDir:      callerCwd(),
		selectedTargets: map[string]bool{},
	}
}

func (m model) Init() tea.Cmd {
	return loadSkills(m.repoRoot)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorVisible()
		m.ensureTargetCursorVisible()
		return m, nil
	case skillsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			if isNoSourcesError(msg.err) {
				m.skills = nil
				m.filtered = nil
				m.noSources = true
				m.status = "No sources configured. Press d for presets or n for a custom source."
				return m, nil
			}
			m.status = "Load failed: " + msg.err.Error()
			return m, nil
		}
		m.noSources = false
		m.skills = msg.skills
		m.applyFilter()
		m.status = fmt.Sprintf("Loaded %d skill(s).", len(m.skills))
		return m, nil
	case defaultsLoadedMsg:
		m.loading = false
		m.viewMode = viewDefaults
		if msg.err != nil {
			m.status = "Load source defaults failed: " + msg.err.Error()
			return m, nil
		}
		m.defaults = msg.defaults
		if m.defaultCursor >= len(m.defaults) {
			m.defaultCursor = len(m.defaults) - 1
		}
		if m.defaultCursor < 0 {
			m.defaultCursor = 0
		}
		m.status = fmt.Sprintf("Loaded %d source default(s).", len(m.defaults))
		return m, nil
	case targetsLoadedMsg:
		m.loading = false
		m.viewMode = viewTargets
		if msg.err != nil {
			m.status = "Load install targets failed: " + msg.err.Error()
			return m, nil
		}
		m.targets = msg.targets
		m.targetStats = indexTargetDetections(msg.detections)
		m.targetChoices = m.buildTargetChoices(msg.targets)
		m.seedSelectedTargets()
		if m.targetCursor >= len(m.targetChoices) {
			m.targetCursor = len(m.targetChoices) - 1
		}
		if m.targetCursor < 0 {
			m.targetCursor = 0
		}
		m.ensureTargetCursorVisible()
		m.status = fmt.Sprintf("Choose install targets for %d selected skill(s).", len(m.selected))
		return m, nil
	case commandDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.pendingInstall = InstallResult{}
			m.reloadOnFinish = false
			m.status = fmt.Sprintf("%s failed: %s", msg.action, msg.err)
			if strings.TrimSpace(msg.output) != "" {
				m.status += " | " + compactOutput(msg.output)
			}
			return m, nil
		}
		if msg.action == "Install" && len(m.pendingInstall.Targets) > 0 {
			m.installResult = m.pendingInstall
			m.pendingInstall = InstallResult{}
			m.viewMode = viewInstallResult
			m.status = fmt.Sprintf("Installed %d skill(s) to %d target(s).", len(m.installResult.SkillNames), len(m.installResult.Targets))
			return m, nil
		}
		m.status = fmt.Sprintf("%s complete.", msg.action)
		if strings.TrimSpace(msg.output) != "" {
			m.status += " " + compactOutput(msg.output)
		}
		if msg.action == "Sync" || m.reloadOnFinish {
			m.reloadOnFinish = false
			m.viewMode = viewSkills
			m.loading = true
			m.status = "Reloading catalog..."
			return m, loadSkills(m.repoRoot)
		}
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	default:
		return m, nil
	}
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if m.searchMode {
		switch msg.String() {
		case "esc":
			m.searchMode = false
			return m, nil
		case "enter":
			m.searchMode = false
			return m, nil
		case "backspace":
			if len(m.search) > 0 {
				m.search = m.search[:len(m.search)-1]
				m.applyFilter()
			}
			return m, nil
		}
		if msg.Type == tea.KeyRunes {
			m.search += msg.String()
			m.applyFilter()
		}
		return m, nil
	}

	if m.loading || m.busy {
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		default:
			return m, nil
		}
	}

	if m.viewMode == viewAddSource {
		return m.updateAddSourceKey(msg)
	}

	if m.viewMode == viewDetails {
		return m.updateDetailsKey(msg)
	}

	if m.viewMode == viewInstallResult {
		return m.updateInstallResultKey(msg)
	}

	if m.viewMode == viewTargets {
		return m.updateTargetsKey(msg)
	}

	if m.viewMode == viewDefaults {
		return m.updateDefaultsKey(msg)
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "/":
		m.searchMode = true
		return m, nil
	case "esc":
		m.search = ""
		m.applyFilter()
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
		return m, nil
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}
		return m, nil
	case " ":
		m.toggleCurrent()
		return m, nil
	case "enter":
		if skill, ok := m.currentSkill(); ok {
			m.viewMode = viewDetails
			m.status = "Viewing " + skill.Key() + "."
		}
		return m, nil
	case "a":
		for _, idx := range m.filtered {
			m.selected[m.skills[idx].Key()] = true
		}
		m.status = fmt.Sprintf("Selected %d visible skill(s).", len(m.filtered))
		return m, nil
	case "c":
		m.selected = map[string]bool{}
		m.status = "Selection cleared."
		return m, nil
	case "i":
		return m.installSelected()
	case "d":
		m.loading = true
		m.status = "Loading source defaults..."
		return m, loadDefaultSources(m.repoRoot)
	case "n":
		m.sourceInput = ""
		m.viewMode = viewAddSource
		m.status = "Enter a source path or git URL."
		return m, nil
	case "t":
		return m.openTargetSelection()
	case "r":
		m.loading = true
		m.status = "Reloading catalog..."
		return m, loadSkills(m.repoRoot)
	case "s":
		m.busy = true
		m.status = "Syncing sources..."
		return m, runSourceCommand(m.repoRoot, "Sync", "sync")
	default:
		return m, nil
	}
}

func (m model) installSelected() (tea.Model, tea.Cmd) {
	names := m.selectedNames()
	if len(names) == 0 {
		m.status = "Select at least one skill before installing."
		return m, nil
	}
	return m.openTargetSelection()
}

func (m model) openTargetSelection() (tea.Model, tea.Cmd) {
	m.loading = true
	m.status = "Loading install targets..."
	return m, loadTargets(m.repoRoot)
}

func (m model) installToSelectedTargets() (tea.Model, tea.Cmd) {
	names := m.selectedNames()
	if len(names) == 0 {
		m.status = "Select at least one skill before installing."
		return m, nil
	}
	choices := m.selectedTargetChoices()
	if len(choices) == 0 {
		m.status = "Select at least one supported install target."
		return m, nil
	}
	m.busy = true
	m.pendingInstall = buildInstallResult(choices, names)
	m.status = fmt.Sprintf("Installing %d skill(s) to %d target(s)...", len(names), len(choices))
	return m, runInstallTargetsCommand(m.repoRoot, choices, m.projectDir, names)
}

func (m model) updateDetailsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc", "enter":
		m.viewMode = viewSkills
		m.status = "Returned to skills."
		return m, nil
	case " ":
		m.toggleCurrent()
		return m, nil
	case "i":
		return m.installSelected()
	default:
		return m, nil
	}
}

func (m model) updateInstallResultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "t":
		m.viewMode = viewTargets
		m.status = "Returned to install targets."
		return m, nil
	case "esc", "enter", "b":
		m.viewMode = viewSkills
		m.status = "Returned to skills."
		return m, nil
	default:
		return m, nil
	}
}

func (m model) updateTargetsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.viewMode = viewSkills
		m.status = "Returned to skills."
		return m, nil
	case "up", "k":
		if m.targetCursor > 0 {
			m.targetCursor--
			m.ensureTargetCursorVisible()
		}
		return m, nil
	case "down", "j":
		if m.targetCursor < len(m.targetChoices)-1 {
			m.targetCursor++
			m.ensureTargetCursorVisible()
		}
		return m, nil
	case " ":
		m.toggleCurrentTarget()
		return m, nil
	case "a":
		if m.selectedTargets == nil {
			m.selectedTargets = map[string]bool{}
		}
		for _, choice := range m.targetChoices {
			if choice.Supported {
				m.selectedTargets[choice.Key] = true
			}
		}
		m.status = fmt.Sprintf("Selected %d supported target option(s).", len(m.selectedTargetChoices()))
		return m, nil
	case "c":
		m.selectedTargets = map[string]bool{}
		m.status = "Target selection cleared."
		return m, nil
	case "enter", "i":
		return m.installToSelectedTargets()
	default:
		return m, nil
	}
}

func (m model) updateAddSourceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.viewMode = viewSkills
		m.status = "Returned to skills."
		return m, nil
	case "enter":
		location := strings.TrimSpace(m.sourceInput)
		if location == "" {
			m.status = "Enter a source path or git URL before adding."
			return m, nil
		}
		m.busy = true
		m.reloadOnFinish = true
		m.status = "Adding source " + location + "..."
		return m, runSourceCommand(m.repoRoot, "Add source", "add", location)
	case "backspace":
		if m.sourceInput != "" {
			runes := []rune(m.sourceInput)
			m.sourceInput = string(runes[:len(runes)-1])
		}
		return m, nil
	}
	if msg.Type == tea.KeyRunes {
		m.sourceInput += msg.String()
		return m, nil
	}
	return m, nil
}

func (m model) updateDefaultsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.viewMode = viewSkills
		m.status = "Returned to skills."
		return m, nil
	case "up", "k":
		if m.defaultCursor > 0 {
			m.defaultCursor--
		}
		return m, nil
	case "down", "j":
		if m.defaultCursor < len(m.defaults)-1 {
			m.defaultCursor++
		}
		return m, nil
	case "enter", "a":
		if len(m.defaults) == 0 {
			m.status = "No source defaults available."
			return m, nil
		}
		source := m.defaults[m.defaultCursor]
		m.busy = true
		m.reloadOnFinish = true
		m.status = "Adding source " + source.Name + "..."
		return m, runSourceCommand(m.repoRoot, "Add source", "defaults", "add", source.Name)
	case "n":
		m.sourceInput = ""
		m.viewMode = viewAddSource
		m.status = "Enter a source path or git URL."
		return m, nil
	default:
		return m, nil
	}
}

func (m model) View() string {
	if m.width == 0 {
		m.width = 100
	}

	contentWidth := m.contentWidth()
	parts := []string{
		m.renderHeader(contentWidth),
		m.renderBody(contentWidth),
		statusStyle.Width(contentWidth).Render(m.status),
		helpStyle.Width(contentWidth).Render(m.helpText()),
	}

	return appStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m model) renderHeader(width int) string {
	search := emptyLabel(m.search, "none")
	if m.searchMode {
		search = m.search + "_"
	}

	title := titleStyle.Render("Skillhub")
	summary := fmt.Sprintf(
		"Sources: %d   Skills: %d   Visible: %d   Selected: %d   Search: %s",
		m.sourceCount(),
		len(m.skills),
		len(m.filtered),
		len(m.selected),
		search,
	)
	target := fmt.Sprintf("Target: %s   Change: t", m.installTargetLabel())
	if m.viewMode == viewTargets {
		target = fmt.Sprintf("Install targets: %d selected   Project: %s", len(m.selectedTargetChoices()), m.projectDir)
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, subtleStyle.Render(summary), subtleStyle.Render(target))
}

func (m model) renderBody(width int) string {
	if m.loading {
		return panel("Skills", statusStyle.Render(m.status), width)
	}
	if m.viewMode == viewAddSource {
		return panel("Add source", m.addSourceContent(width-6), width)
	}
	if m.viewMode == viewDetails {
		return panel("Skill details", m.detailsContent(width-6), width)
	}
	if m.viewMode == viewInstallResult {
		return panel("Install complete", m.installResultContent(width-6), width)
	}
	if m.viewMode == viewTargets {
		return panel("Install targets", m.targetsContent(width-6), width)
	}
	if m.viewMode == viewDefaults {
		return panel("Source presets", m.defaultsContent(width-6), width)
	}
	if m.noSources {
		return panel("Sources", "No sources configured.\n\nPress d for presets or n for a custom source.", width)
	}
	if len(m.filtered) == 0 {
		return panel("Skills", "No skills matched.", width)
	}

	return panel("Skills", m.skillsContent(width-6), width)
}

func (m model) skillsContent(width int) string {
	var b strings.Builder

	visible := m.visibleCount()
	end := m.offset + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	lastInCategory := m.lastVisibleIndexByCategory(m.offset, end)
	previousCategory := ""
	for row, idx := range m.filtered[m.offset:end] {
		skill := m.skills[idx]
		if skill.Category != previousCategory {
			if row != 0 {
				fmt.Fprintln(&b)
			}
			fmt.Fprintln(&b, categoryStyle.Render("• "+skill.Category))
			previousCategory = skill.Category
		}

		cursor := " "
		if m.offset+row == m.cursor {
			cursor = "›"
		}
		branch := "├─"
		if lastInCategory[skill.Category] == idx {
			branch = "└─"
		}
		checkbox := checkboxStyle.Render("[ ]")
		if m.selected[skill.Key()] {
			checkbox = checkedStyle.Render("[✓]")
		}
		title := fmt.Sprintf("%s %s %s %s", cursor, treeStyle.Render(branch), checkbox, skill.Name)
		if m.offset+row == m.cursor {
			title = activeRowStyle.Render(title)
		}
		description := indent(wrapText(skill.Description, width-8), "      ")
		meta := "      source: " + skill.Source
		if strings.TrimSpace(skill.Triggers) != "" {
			meta += "   triggers: " + truncate(skill.Triggers, max(16, width-lipgloss.Width(meta)-13))
		}
		card := strings.Join([]string{title, description, subtleStyle.Render(meta)}, "\n")
		if m.selected[skill.Key()] {
			card = selectedRowStyle.Render(card)
		}
		fmt.Fprintln(&b, card)
		if row != end-m.offset-1 {
			fmt.Fprintln(&b)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) detailsContent(width int) string {
	skill, ok := m.currentSkill()
	if !ok {
		return "No skill selected."
	}
	selected := "no"
	if m.selected[skill.Key()] {
		selected = "yes"
	}
	lines := []string{
		titleStyle.Render(skill.Key()),
		"",
		labelLine("Source", skill.Source),
		labelLine("Category", badgeStyle.Render(skill.Category)),
		labelLine("Triggers", skill.Triggers),
		labelLine("Selected", selected),
		labelLine("Default path", filepath.Join(m.installTargetPath(), skill.Name)),
		"",
		wrapText(skill.Description, width),
	}
	return strings.Join(lines, "\n")
}

func (m model) targetsContent(width int) string {
	if len(m.targetChoices) == 0 {
		return "No install targets available."
	}

	var b strings.Builder
	visible := m.targetVisibleCount()
	end := m.targetOffset + visible
	if end > len(m.targetChoices) {
		end = len(m.targetChoices)
	}
	for row, choice := range m.targetChoices[m.targetOffset:end] {
		i := m.targetOffset + row
		if row > 0 {
			fmt.Fprintln(&b)
		}

		cursor := " "
		if i == m.targetCursor {
			cursor = "›"
		}
		checkbox := checkboxStyle.Render("[ ]")
		if !choice.Supported {
			checkbox = disabledStyle.Render("[-]")
		} else if m.selectedTargets[choice.Key] {
			checkbox = checkedStyle.Render("[✓]")
		}

		title := fmt.Sprintf("%s %s %s", cursor, checkbox, choice.Label)
		if i == m.targetCursor {
			title = activeRowStyle.Render(title)
		}
		if choice.Supported && m.selectedTargets[choice.Key] {
			title = selectedRowStyle.Render(title)
		}
		if !choice.Supported {
			title = disabledStyle.Render(title)
		}

		meta := "    " + choice.Status
		if strings.TrimSpace(choice.Path) != "" {
			meta += " -> " + choice.Path
		}
		if choice.Supported {
			meta += fmt.Sprintf("   exists: %s   skills: %s   managed: %s",
				emptyLabel(choice.Exists, "-"),
				emptyLabel(choice.Skills, "0"),
				emptyLabel(choice.Managed, "0"),
			)
		}
		description := indent(wrapText(choice.Description, width-4), "    ")
		fmt.Fprintln(&b, title)
		fmt.Fprintln(&b, subtleStyle.Render(meta))
		if strings.TrimSpace(description) != "" {
			fmt.Fprintln(&b, description)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) installResultContent(width int) string {
	if len(m.installResult.Targets) == 0 {
		return "No install result available."
	}

	var b strings.Builder
	fmt.Fprintln(&b, titleStyle.Render("Installed successfully"))
	fmt.Fprintln(&b)
	for targetIndex, target := range m.installResult.Targets {
		if targetIndex > 0 {
			fmt.Fprintln(&b)
		}
		fmt.Fprintf(&b, "%s %s\n", checkedStyle.Render("[✓]"), target.Label)
		meta := fmt.Sprintf("    %s/%s -> %s", target.Target, target.Scope, target.Root)
		fmt.Fprintln(&b, subtleStyle.Render(meta))
		for _, path := range target.SkillPaths {
			fmt.Fprintf(&b, "    %s\n", path)
		}
	}
	fmt.Fprintln(&b)
	actions := strings.Join([]string{
		badgeStyle.Render("enter back"),
		badgeStyle.Render("t targets"),
		badgeStyle.Render("q quit"),
	}, "  ")
	fmt.Fprintln(&b, actions)
	return strings.TrimRight(b.String(), "\n")
}

func (m model) addSourceContent(width int) string {
	input := m.sourceInput
	if strings.TrimSpace(input) == "" {
		input = "path or git URL"
	}
	return strings.Join([]string{
		labelLine("Location", input+"_"),
		"",
		wrapText("Name defaults to the path or repository basename. Use CLI flags for custom name, type, ref, or catalog.", width),
	}, "\n")
}

func (m model) defaultsContent(width int) string {
	if len(m.defaults) == 0 {
		return "No source defaults available."
	}
	var b strings.Builder
	for i, source := range m.defaults {
		cursor := " "
		if i == m.defaultCursor {
			cursor = ">"
		}
		line := fmt.Sprintf("%s %-20s %-8s %-12s %s",
			cursor,
			truncate(source.Name, 20),
			truncate(source.Type, 8),
			truncate(source.Ref, 12),
			truncate(source.Location, width-46),
		)
		if i == m.defaultCursor {
			line = activeRowStyle.Width(width).Render(line)
		}
		fmt.Fprintln(&b, line)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) helpText() string {
	if m.viewMode == viewDefaults {
		return "j/k move  enter add preset  n add custom  esc back  q quit"
	}
	if m.viewMode == viewAddSource {
		return "enter add  esc back  q quit"
	}
	if m.viewMode == viewDetails {
		return "space select  i targets  enter/esc back  q quit"
	}
	if m.viewMode == viewInstallResult {
		return "enter/b back  t targets  q quit"
	}
	if m.viewMode == viewTargets {
		return "j/k move  space toggle  enter/i install  a all  c clear  esc back  q quit"
	}
	return "j/k move  space select  enter details  / search  a all  c clear  d presets  n source  t targets  i install  q quit"
}

func panel(title, body string, width int) string {
	if width < 24 {
		width = 24
	}
	contentWidth := width - 6
	if contentWidth < 10 {
		contentWidth = 10
	}
	content := lipgloss.JoinVertical(lipgloss.Left, panelTitleStyle.Render(title), "", body)
	return panelStyle.Width(contentWidth).Render(content)
}

func (m *model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.search))
	m.filtered = m.filtered[:0]
	for i, skill := range m.skills {
		haystack := strings.ToLower(strings.Join([]string{
			skill.Source,
			skill.Name,
			skill.Category,
			skill.Triggers,
			skill.Description,
		}, " "))
		if q == "" || strings.Contains(haystack, q) {
			m.filtered = append(m.filtered, i)
		}
	}
	sort.SliceStable(m.filtered, func(i, j int) bool {
		left := m.skills[m.filtered[i]]
		right := m.skills[m.filtered[j]]
		if left.Category != right.Category {
			return left.Category < right.Category
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.Source < right.Source
	})
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.offset = 0
	m.ensureCursorVisible()
}

func (m model) lastVisibleIndexByCategory(start, end int) map[string]int {
	result := map[string]int{}
	for _, idx := range m.filtered[start:end] {
		result[m.skills[idx].Category] = idx
	}
	return result
}

func (m *model) ensureCursorVisible() {
	visible := m.visibleCount()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	maxOffset := len(m.filtered) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
}

func (m model) visibleCount() int {
	if m.height <= 0 {
		return 5
	}
	count := (m.height - 14) / 5
	if count < 1 {
		return 1
	}
	return count
}

func (m *model) ensureTargetCursorVisible() {
	visible := m.targetVisibleCount()
	if m.targetCursor < m.targetOffset {
		m.targetOffset = m.targetCursor
	}
	if m.targetCursor >= m.targetOffset+visible {
		m.targetOffset = m.targetCursor - visible + 1
	}
	if m.targetOffset < 0 {
		m.targetOffset = 0
	}
	maxOffset := len(m.targetChoices) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.targetOffset > maxOffset {
		m.targetOffset = maxOffset
	}
}

func (m model) targetVisibleCount() int {
	if m.height <= 0 {
		return 5
	}
	count := (m.height - 14) / 5
	if count < 1 {
		return 1
	}
	return count
}

func (m *model) toggleCurrent() {
	if len(m.filtered) == 0 {
		return
	}
	skill := m.skills[m.filtered[m.cursor]]
	key := skill.Key()
	if m.selected[key] {
		delete(m.selected, key)
		m.status = "Unselected " + key + "."
	} else {
		m.selected[key] = true
		m.status = "Selected " + key + "."
	}
}

func (m *model) toggleCurrentTarget() {
	if len(m.targetChoices) == 0 || m.targetCursor < 0 || m.targetCursor >= len(m.targetChoices) {
		return
	}
	choice := m.targetChoices[m.targetCursor]
	if !choice.Supported {
		m.status = choice.Label + " is not supported by the TUI installer yet."
		return
	}
	if m.selectedTargets == nil {
		m.selectedTargets = map[string]bool{}
	}
	if m.selectedTargets[choice.Key] {
		delete(m.selectedTargets, choice.Key)
		m.status = "Unselected " + choice.Label + "."
	} else {
		m.selectedTargets[choice.Key] = true
		m.status = "Selected " + choice.Label + "."
	}
}

func (m model) currentSkill() (Skill, bool) {
	if len(m.filtered) == 0 || m.cursor < 0 || m.cursor >= len(m.filtered) {
		return Skill{}, false
	}
	return m.skills[m.filtered[m.cursor]], true
}

func (m model) sourceCount() int {
	seen := map[string]bool{}
	for _, skill := range m.skills {
		if strings.TrimSpace(skill.Source) != "" {
			seen[skill.Source] = true
		}
	}
	return len(seen)
}

func (m model) contentWidth() int {
	width := m.width - 4
	if width < 72 {
		return 72
	}
	if width > 150 {
		return 150
	}
	return width
}

func (s Skill) Key() string {
	if strings.TrimSpace(s.Source) == "" {
		return s.Name
	}
	return s.Source + "/" + s.Name
}

func (m model) buildTargetChoices(targets []Target) []InstallTargetChoice {
	choices := []InstallTargetChoice{}
	for _, target := range sortedTargetsForDisplay(targets) {
		if target.ID == "directory" {
			choices = append(choices, InstallTargetChoice{
				Key:         "directory",
				Target:      target.ID,
				Label:       target.Label,
				Status:      "CLI only",
				Path:        "use --dir <path>",
				Description: target.Description,
				Supported:   false,
			})
			continue
		}

		if target.Status == "supported" && target.Adapter == "skill-dir" && isNativeScopedTarget(target.ID) {
			for _, scope := range []string{"global", "project"} {
				choices = append(choices, InstallTargetChoice{
					Key:         target.ID + ":" + scope,
					Target:      target.ID,
					Label:       target.Label + " " + scope,
					Scope:       scope,
					Status:      "supported",
					Path:        targetSkillRoot(target.ID, scope, m.projectDir),
					Description: target.Description,
					Supported:   true,
				})
				choice := &choices[len(choices)-1]
				if detection, ok := m.targetStats[choice.Key]; ok {
					choice.Status = detection.Status
					choice.Path = detection.Path
					choice.Exists = detection.Exists
					choice.Skills = detection.Skills
					choice.Managed = detection.Managed
				}
				applyLegacyCodexEnv(choice)
			}
			continue
		}

		choices = append(choices, InstallTargetChoice{
			Key:         target.ID,
			Target:      target.ID,
			Label:       target.Label,
			Status:      "planned",
			Description: target.Description,
			Supported:   false,
		})
	}
	return choices
}

func sortedTargetsForDisplay(targets []Target) []Target {
	ordered := append([]Target(nil), targets...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return targetStatusRank(ordered[i].Status) < targetStatusRank(ordered[j].Status)
	})
	return ordered
}

func targetStatusRank(status string) int {
	if status == "planned" {
		return 1
	}
	return 0
}

func indexTargetDetections(detections []TargetDetection) map[string]TargetDetection {
	result := map[string]TargetDetection{}
	for _, detection := range detections {
		result[detection.Target+":"+detection.Scope] = detection
	}
	return result
}

func (m *model) seedSelectedTargets() {
	previous := m.selectedTargets
	m.selectedTargets = map[string]bool{}
	for _, choice := range m.targetChoices {
		if previous[choice.Key] && choice.Supported {
			m.selectedTargets[choice.Key] = true
		}
	}
	if len(m.selectedTargets) > 0 {
		return
	}

	defaultKey := "codex:" + m.installScope
	for _, choice := range m.targetChoices {
		if choice.Key == defaultKey && choice.Supported {
			m.selectedTargets[choice.Key] = true
			return
		}
	}
	for _, choice := range m.targetChoices {
		if choice.Supported {
			m.selectedTargets[choice.Key] = true
			return
		}
	}
}

func (m model) selectedTargetChoices() []InstallTargetChoice {
	choices := []InstallTargetChoice{}
	for _, choice := range m.targetChoices {
		if choice.Supported && m.selectedTargets[choice.Key] {
			choices = append(choices, choice)
		}
	}
	return choices
}

func buildInstallResult(choices []InstallTargetChoice, names []string) InstallResult {
	result := InstallResult{
		SkillNames: append([]string(nil), names...),
		Targets:    make([]InstallTargetResult, 0, len(choices)),
	}
	for _, choice := range choices {
		target := InstallTargetResult{
			Label:  choice.Label,
			Target: choice.Target,
			Scope:  choice.Scope,
			Root:   choice.Path,
		}
		for _, name := range names {
			if target.Root == "" {
				target.SkillPaths = append(target.SkillPaths, installSkillDirName(name))
				continue
			}
			target.SkillPaths = append(target.SkillPaths, filepath.Join(target.Root, installSkillDirName(name)))
		}
		result.Targets = append(result.Targets, target)
	}
	return result
}

func installSkillDirName(name string) string {
	if index := strings.LastIndex(name, "/"); index >= 0 {
		return name[index+1:]
	}
	return name
}

func (m model) selectedNames() []string {
	names := make([]string, 0, len(m.selected))
	for name := range m.selected {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m model) installArgs() []string {
	if m.installScope == "project" {
		return []string{"--target", "codex", "--scope", "project", "--project", m.projectDir}
	}
	return nil
}

func (m model) installTargetSummary() string {
	return fmt.Sprintf("codex/%s -> %s", m.installScope, m.installTargetPath())
}

func (m model) installTargetLabel() string {
	if m.installScope == "project" {
		return "Codex project"
	}
	return "Codex global"
}

func (m model) installTargetPath() string {
	if m.installScope == "project" {
		return targetSkillRoot("codex", "project", m.projectDir)
	}
	if dir := os.Getenv("AGENT_SKILLS_DIR"); strings.TrimSpace(dir) != "" {
		return dir
	}
	return targetSkillRoot("codex", "global", m.projectDir)
}

func isNativeScopedTarget(target string) bool {
	switch target {
	case "codex", "claude", "gemini", "opencode":
		return true
	default:
		return false
	}
}

func targetSkillRoot(target, scope, projectDir string) string {
	switch target {
	case "codex":
		if scope == "project" {
			return filepath.Join(projectDir, ".agents", "skills")
		}
		if dir := legacyCodexSkillsDir(); dir != "" {
			return dir
		}
		return homePath(".agents", "skills")
	case "claude":
		if scope == "project" {
			return filepath.Join(projectDir, ".claude", "skills")
		}
		return homePath(".claude", "skills")
	case "gemini":
		if scope == "project" {
			return filepath.Join(projectDir, ".gemini", "skills")
		}
		return homePath(".gemini", "skills")
	case "opencode":
		if scope == "project" {
			return filepath.Join(projectDir, ".opencode", "skills")
		}
		if dir := os.Getenv("OPENCODE_CONFIG_DIR"); strings.TrimSpace(dir) != "" {
			return filepath.Join(dir, "skills")
		}
		return homePath(".config", "opencode", "skills")
	default:
		return ""
	}
}

func legacyCodexSkillsDir() string {
	return strings.TrimSpace(os.Getenv("AGENT_SKILLS_DIR"))
}

func usesLegacyCodexEnv(choice InstallTargetChoice) bool {
	return choice.Target == "codex" && choice.Scope == "global" && legacyCodexSkillsDir() != ""
}

func applyLegacyCodexEnv(choice *InstallTargetChoice) {
	if choice == nil || !usesLegacyCodexEnv(*choice) {
		return
	}
	root := legacyCodexSkillsDir()
	choice.Path = root
	choice.Exists = dirExistsLabel(root)
	choice.Skills = fmt.Sprintf("%d", countSkillDirs(root, false))
	choice.Managed = fmt.Sprintf("%d", countSkillDirs(root, true))
}

func dirExistsLabel(path string) string {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return "yes"
	}
	return "no"
}

func countSkillDirs(root string, managedOnly bool) int {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(root, entry.Name())
		if managedOnly {
			if _, err := os.Stat(filepath.Join(skillDir, ".skillhub.json")); err == nil {
				count++
			}
			continue
		}
		if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err == nil {
			count++
		}
	}
	return count
}

func homePath(parts ...string) string {
	if home := os.Getenv("HOME"); strings.TrimSpace(home) != "" {
		return filepath.Join(append([]string{home}, parts...)...)
	}
	return "~/" + filepath.Join(parts...)
}

func callerCwd() string {
	if cwd := os.Getenv("SKILLHUB_CALLER_CWD"); strings.TrimSpace(cwd) != "" {
		return cwd
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

func loadSkills(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/skills.sh", "list", "--tsv")
		if err != nil {
			return skillsLoadedMsg{err: commandError(err, output)}
		}
		skills, err := parseSkillsTSV(output)
		return skillsLoadedMsg{skills: skills, err: err}
	}
}

func loadDefaultSources(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/sources.sh", "defaults", "list", "--tsv")
		if err != nil {
			return defaultsLoadedMsg{err: commandError(err, output)}
		}
		defaults, err := parseDefaultSourcesTSV(output)
		return defaultsLoadedMsg{defaults: defaults, err: err}
	}
}

func loadTargets(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/targets.sh", "list", "--tsv")
		if err != nil {
			return targetsLoadedMsg{err: commandError(err, output)}
		}
		targets, err := parseTargetsTSV(output)
		if err != nil {
			return targetsLoadedMsg{err: err}
		}
		detectOutput, err := runScript(repoRoot, "scripts/targets.sh", "detect", "--tsv", "--project", callerCwd())
		if err != nil {
			return targetsLoadedMsg{err: commandError(err, detectOutput)}
		}
		detections, err := parseTargetDetectionsTSV(detectOutput)
		return targetsLoadedMsg{targets: targets, detections: detections, err: err}
	}
}

func runSkillCommand(repoRoot, action string, args ...string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/skills.sh", args...)
		return commandDoneMsg{action: action, output: output, err: commandError(err, output)}
	}
}

func runSourceCommand(repoRoot, action string, args ...string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/sources.sh", args...)
		return commandDoneMsg{action: action, output: output, err: commandError(err, output)}
	}
}

func runInstallTargetsCommand(repoRoot string, choices []InstallTargetChoice, projectDir string, names []string) tea.Cmd {
	return func() tea.Msg {
		var combined strings.Builder
		for _, choice := range choices {
			args := installArgsForTargetChoice(choice, projectDir, names)

			output, err := runScript(repoRoot, "scripts/skills.sh", args...)
			if strings.TrimSpace(output) != "" {
				fmt.Fprintf(&combined, "[%s]\n%s", choice.Label, output)
			}
			if err != nil {
				return commandDoneMsg{action: "Install", output: combined.String(), err: commandError(err, output)}
			}
		}
		return commandDoneMsg{action: "Install", output: combined.String()}
	}
}

func installArgsForTargetChoice(choice InstallTargetChoice, projectDir string, names []string) []string {
	args := []string{"install"}
	if !usesLegacyCodexEnv(choice) {
		args = append(args, "--target", choice.Target, "--scope", choice.Scope)
		if choice.Scope == "project" {
			args = append(args, "--project", projectDir)
		}
	}
	return append(args, names...)
}

func runScript(repoRoot, script string, args ...string) (string, error) {
	cmd := exec.Command("sh", append([]string{filepath.Join(repoRoot, script)}, args...)...)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	if os.Getenv("SKILLHUB_CALLER_CWD") == "" {
		cmd.Env = append(cmd.Env, "SKILLHUB_CALLER_CWD="+callerCwd())
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func parseSkillsTSV(input string) ([]Skill, error) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] != "source\tname\tcategory\ttriggers\tdescription" {
		return nil, fmt.Errorf("unexpected skills TSV header")
	}
	skills := make([]Skill, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) != 5 {
			return nil, fmt.Errorf("invalid skills TSV row: %q", line)
		}
		skills = append(skills, Skill{
			Source:      parts[0],
			Name:        parts[1],
			Category:    parts[2],
			Triggers:    parts[3],
			Description: parts[4],
		})
	}
	return skills, nil
}

func parseDefaultSourcesTSV(input string) ([]SourcePreset, error) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] != "name\ttype\tlocation\tref\tcatalog" {
		return nil, fmt.Errorf("unexpected source defaults TSV header")
	}
	defaults := make([]SourcePreset, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) != 5 {
			return nil, fmt.Errorf("invalid source defaults TSV row: %q", line)
		}
		defaults = append(defaults, SourcePreset{
			Name:     parts[0],
			Type:     parts[1],
			Location: parts[2],
			Ref:      parts[3],
			Catalog:  parts[4],
		})
	}
	return defaults, nil
}

func parseTargetsTSV(input string) ([]Target, error) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] != "id\tlabel\tstatus\tadapter\tdescription" {
		return nil, fmt.Errorf("unexpected targets TSV header")
	}
	targets := make([]Target, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) != 5 {
			return nil, fmt.Errorf("invalid targets TSV row: %q", line)
		}
		targets = append(targets, Target{
			ID:          parts[0],
			Label:       parts[1],
			Status:      parts[2],
			Adapter:     parts[3],
			Description: parts[4],
		})
	}
	return targets, nil
}

func parseTargetDetectionsTSV(input string) ([]TargetDetection, error) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] != "target\tscope\tstatus\tpath\texists\tskills\tmanaged" {
		return nil, fmt.Errorf("unexpected target detections TSV header")
	}
	detections := make([]TargetDetection, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 7)
		if len(parts) != 7 {
			return nil, fmt.Errorf("invalid target detections TSV row: %q", line)
		}
		detections = append(detections, TargetDetection{
			Target:  parts[0],
			Scope:   parts[1],
			Status:  parts[2],
			Path:    parts[3],
			Exists:  parts[4],
			Skills:  parts[5],
			Managed: parts[6],
		})
	}
	return detections, nil
}

func commandError(err error, output string) error {
	if err == nil {
		return nil
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, compactOutput(output))
}

func isNoSourcesError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "No sources configured")
}

func compactOutput(output string) string {
	lines := strings.Fields(strings.ReplaceAll(output, "\n", " "))
	if len(lines) == 0 {
		return ""
	}
	return truncate(strings.Join(lines, " "), 140)
}

func truncate(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func emptyLabel(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func labelLine(label, value string) string {
	return fmt.Sprintf("%-9s %s", subtleStyle.Render(label), value)
}

func wrapText(value string, width int) string {
	value = strings.TrimSpace(value)
	if value == "" || width <= 0 {
		return value
	}
	words := strings.Fields(value)
	if len(words) == 0 {
		return ""
	}
	lines := []string{}
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) > width {
			lines = append(lines, current)
			current = word
			continue
		}
		current += " " + word
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n")
}

func indent(value, prefix string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
