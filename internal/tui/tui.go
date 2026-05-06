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

type InstalledSkill struct {
	Target         string
	Scope          string
	Skill          string
	Managed        string
	Source         string
	QualifiedSkill string
	ProjectPath    string
	TargetRoot     string
	InstalledPath  string
	ContentHash    string
	InstalledAt    string
	UpdatedAt      string
	Path           string
	RegistryOnly   bool
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

type UsageSummary struct {
	Key           string
	Source        string
	Skill         string
	InstallCount  int
	ProjectCount  int
	TargetCount   int
	LatestUpdated string
}

const (
	viewSkills           = "skills"
	viewInstalled        = "installed"
	viewUsage            = "usage"
	viewUsageDetails     = "usage-details"
	viewSources          = "sources"
	viewUpdate           = "update"
	viewDefaults         = "defaults"
	viewAddSource        = "add-source"
	viewDetails          = "details"
	viewInstalledDetails = "installed-details"
	viewTargets          = "targets"
	viewInstallResult    = "install-result"
	viewConfirmDelete    = "confirm-delete"
	viewHelp             = "help"
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

type installedStats struct {
	Installed int
	Skills    int
	Projects  int
	Managed   int
}

type model struct {
	repoRoot string

	skills         []Skill
	filtered       []int
	selected       map[string]bool
	sources        []SourcePreset
	installedRows  []InstalledSkill
	usageRows      []InstalledSkill
	usageSummaries []UsageSummary
	defaults       []SourcePreset
	targets        []Target
	targetStats    map[string]TargetDetection

	cursor                int
	installedCursor       int
	installedDetailCursor int
	usageCursor           int
	usageDetailCursor     int
	defaultCursor         int
	targetCursor          int
	offset                int
	installedOffset       int
	installedDetailOffset int
	usageOffset           int
	usageDetailOffset     int
	targetOffset          int
	width                 int
	height                int
	bodyHeight            int
	search                string
	usageFilter           string
	sourceInput           string
	sourceNameInput       string
	sourceField           int
	searchMode            bool
	usageFilterMode       bool
	loading               bool
	loadingSkills         bool
	loadingInstalled      bool
	busy                  bool
	noSources             bool
	status                string
	viewMode              string
	previousViewMode      string
	installedDetailKey    string
	usageDetailKey        string
	returnToUsageDetails  bool
	reloadOnFinish        bool
	postReloadStatus      string

	installScope     string
	projectDir       string
	targetPurpose    string
	targetChoices    []InstallTargetChoice
	selectedTargets  map[string]bool
	pendingInstall   InstallResult
	installResult    InstallResult
	installProgress  installProgressState
	pendingUninstall InstalledSkill
}

type skillsLoadedMsg struct {
	skills  []Skill
	warning string
	err     error
}

type defaultsLoadedMsg struct {
	defaults []SourcePreset
	err      error
}

type sourcesLoadedMsg struct {
	sources []SourcePreset
	err     error
}

type installedLoadedMsg struct {
	rows []InstalledSkill
	err  error
}

type usageLoadedMsg struct {
	rows []InstalledSkill
	err  error
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

type installStepDoneMsg struct {
	output string
	err    error
}

type installQueueItem struct {
	Skill  string
	Choice InstallTargetChoice
}

type installProgressState struct {
	Items     []installQueueItem
	Current   int
	Completed int
	Total     int
	LastLine  string
	Failed    bool
	Error     string
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
	searchBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("63")).
				Bold(true).
				Padding(0, 1)
	projectBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Bold(true)
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
		repoRoot:         repoRoot,
		selected:         map[string]bool{},
		loading:          true,
		loadingSkills:    true,
		loadingInstalled: true,
		status:           "Loading installed skills...",
		viewMode:         viewInstalled,

		installScope:    "global",
		projectDir:      callerCwd(),
		selectedTargets: map[string]bool{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadInstalled(m.repoRoot), loadSkills(m.repoRoot))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorVisible()
		m.ensureUsageCursorVisible()
		m.ensureUsageDetailCursorVisible()
		m.ensureTargetCursorVisible()
		return m, nil
	case skillsLoadedMsg:
		m.loadingSkills = false
		m.loading = m.currentViewLoading()
		showStatus := m.dashboardSection() == viewSkills
		if msg.err != nil {
			if isNoSourcesError(msg.err) {
				m.skills = nil
				m.filtered = nil
				m.noSources = true
				if showStatus {
					m.status = "No sources configured. Press d for presets or n for a custom source."
				}
				return m, nil
			}
			if showStatus {
				m.status = "Load failed: " + msg.err.Error()
			}
			return m, nil
		}
		m.noSources = false
		m.skills = msg.skills
		m.applyFilter()
		if showStatus {
			m.status = fmt.Sprintf("Loaded %d skill(s).", len(m.skills))
			if strings.TrimSpace(msg.warning) != "" {
				m.status += " " + compactOutput(msg.warning)
			}
		}
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
	case sourcesLoadedMsg:
		m.loading = false
		m.viewMode = viewSources
		if msg.err != nil {
			m.status = "Load sources failed: " + msg.err.Error()
			return m, nil
		}
		m.sources = msg.sources
		m.status = fmt.Sprintf("Loaded %d active source(s).", len(m.sources))
		return m, nil
	case installedLoadedMsg:
		m.loadingInstalled = false
		m.loading = m.currentViewLoading()
		showStatus := m.dashboardSection() == viewInstalled || m.viewMode == viewInstallResult || strings.TrimSpace(m.postReloadStatus) != ""
		if msg.err != nil {
			m.postReloadStatus = ""
			if showStatus {
				m.status = "Load installed skills failed: " + msg.err.Error()
			}
			return m, nil
		}
		m.installedRows = msg.rows
		if m.installedCursor >= len(m.installedRows) {
			m.installedCursor = len(m.installedRows) - 1
		}
		if m.installedCursor < 0 {
			m.installedCursor = 0
		}
		m.ensureInstalledCursorVisible()
		if strings.TrimSpace(m.postReloadStatus) != "" {
			m.status = m.postReloadStatus
			m.postReloadStatus = ""
		} else if showStatus {
			m.status = fmt.Sprintf("Loaded %d installed skill row(s).", len(m.installedRows))
		}
		return m, nil
	case usageLoadedMsg:
		m.loading = false
		returnToDetails := m.returnToUsageDetails
		detailKey := m.usageDetailKey
		m.returnToUsageDetails = false
		if msg.err != nil {
			m.postReloadStatus = ""
			m.status = "Load usage failed: " + msg.err.Error()
			return m, nil
		}
		m.usageRows = msg.rows
		m.applyUsageFilter()
		if m.usageCursor >= len(m.usageSummaries) {
			m.usageCursor = len(m.usageSummaries) - 1
		}
		if m.usageCursor < 0 {
			m.usageCursor = 0
		}
		m.ensureUsageCursorVisible()
		if returnToDetails && strings.TrimSpace(detailKey) != "" {
			m.usageDetailKey = detailKey
			detailRows := m.usageDetailRows()
			if len(detailRows) > 0 {
				if m.usageDetailCursor >= len(detailRows) {
					m.usageDetailCursor = len(detailRows) - 1
				}
				if m.usageDetailCursor < 0 {
					m.usageDetailCursor = 0
				}
				m.ensureUsageDetailCursorVisible()
				m.viewMode = viewUsageDetails
			} else {
				m.viewMode = viewUsage
				m.usageDetailKey = ""
			}
		} else {
			m.viewMode = viewUsage
		}
		if strings.TrimSpace(m.postReloadStatus) != "" {
			m.status = m.postReloadStatus
			m.postReloadStatus = ""
		} else {
			m.status = fmt.Sprintf("Loaded %d managed usage row(s).", len(m.usageRows))
		}
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
		if m.targetPurpose == "install" {
			m.status = fmt.Sprintf("Choose install targets for %d selected skill(s).", len(m.selected))
		} else {
			m.status = "Loaded target paths."
		}
		return m, nil
	case installStepDoneMsg:
		if strings.TrimSpace(msg.output) != "" {
			m.installProgress.LastLine = lastOutputLine(msg.output)
		}
		if msg.err != nil {
			m.busy = false
			m.installProgress.Failed = true
			m.installProgress.Error = msg.err.Error()
			m.status = fmt.Sprintf("Install failed at %d/%d.", m.installProgress.Current, m.installProgress.Total)
			return m, nil
		}
		m.installProgress.Completed++
		if m.installProgress.Completed >= m.installProgress.Total {
			m.busy = false
			m.installResult = m.pendingInstall
			m.pendingInstall = InstallResult{}
			m.installProgress = installProgressState{}
			m.viewMode = viewInstallResult
			m.postReloadStatus = fmt.Sprintf("Installed %d skill(s) to %d target(s).", len(m.installResult.SkillNames), len(m.installResult.Targets))
			m.status = m.postReloadStatus
			m.loadingInstalled = true
			return m, loadInstalled(m.repoRoot)
		}
		m.installProgress.Current = m.installProgress.Completed + 1
		item, _ := m.installProgress.currentItem()
		m.status = m.installProgressStatus()
		return m, runInstallStepCommand(m.repoRoot, item, m.projectDir)
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
			m.postReloadStatus = fmt.Sprintf("Installed %d skill(s) to %d target(s).", len(m.installResult.SkillNames), len(m.installResult.Targets))
			m.status = m.postReloadStatus
			m.loadingInstalled = true
			return m, loadInstalled(m.repoRoot)
		}
		successStatus := fmt.Sprintf("%s complete.", msg.action)
		if strings.TrimSpace(msg.output) != "" {
			successStatus += " " + compactOutput(msg.output)
		}
		m.status = successStatus
		if msg.action == "Sync" || m.reloadOnFinish {
			m.reloadOnFinish = false
			return m.reloadCurrentView("Reloading...")
		}
		if msg.action == "Update installed" || msg.action == "Uninstall" {
			m.loading = true
			m.loadingInstalled = true
			m.viewMode = viewInstalled
			m.postReloadStatus = successStatus
			m.status = "Reloading installed skills..."
			return m, loadInstalled(m.repoRoot)
		}
		if msg.action == "Update usage" {
			m.loading = true
			if !m.returnToUsageDetails {
				m.viewMode = viewUsage
			}
			m.postReloadStatus = successStatus
			m.status = "Reloading usage..."
			return m, loadUsage(m.repoRoot)
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

	if m.usageFilterMode {
		switch msg.String() {
		case "esc":
			m.usageFilterMode = false
			m.usageFilter = ""
			m.applyUsageFilter()
			m.status = "Usage filter cleared."
			return m, nil
		case "enter":
			m.usageFilterMode = false
			m.status = "Usage filter applied."
			return m, nil
		case "backspace":
			if len(m.usageFilter) > 0 {
				runes := []rune(m.usageFilter)
				m.usageFilter = string(runes[:len(runes)-1])
				m.applyUsageFilter()
			}
			return m, nil
		}
		if msg.Type == tea.KeyRunes {
			m.usageFilter += msg.String()
			m.applyUsageFilter()
		}
		return m, nil
	}

	if m.installProgress.Failed {
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "esc", "enter", "b":
			m.installProgress = installProgressState{}
			m.status = "Returned to install targets."
			return m, nil
		default:
			return m, nil
		}
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

	switch msg.String() {
	case "?":
		m.previousViewMode = m.viewMode
		m.viewMode = viewHelp
		m.status = "Help."
		return m, nil
	case "left":
		if m.canMoveDashboardSection() {
			return m.moveDashboardSection(-1)
		}
		return m, nil
	case "right":
		if m.canMoveDashboardSection() {
			return m.moveDashboardSection(1)
		}
		return m, nil
	case "1":
		return m.openDashboardSection(viewInstalled)
	case "2":
		return m.openDashboardSection(viewSkills)
	case "3":
		return m.openDashboardSection(viewUsage)
	case "4":
		return m.openDashboardSection(viewSources)
	case "5":
		return m.openDashboardSection(viewTargets)
	case "6":
		return m.openDashboardSection(viewUpdate)
	}

	if m.viewMode == viewHelp {
		return m.updateHelpKey(msg)
	}

	if m.viewMode == viewConfirmDelete {
		return m.updateConfirmDeleteKey(msg)
	}

	if m.viewMode == viewDetails {
		return m.updateDetailsKey(msg)
	}

	if m.viewMode == viewInstalledDetails {
		return m.updateInstalledDetailsKey(msg)
	}

	if m.viewMode == viewUsageDetails {
		return m.updateUsageDetailsKey(msg)
	}

	if m.viewMode == viewInstallResult {
		return m.updateInstallResultKey(msg)
	}

	if m.viewMode == viewTargets {
		return m.updateTargetsKey(msg)
	}

	if m.viewMode == viewInstalled {
		return m.updateInstalledKey(msg)
	}

	if m.viewMode == viewUsage {
		return m.updateUsageKey(msg)
	}

	if m.viewMode == viewSources {
		return m.updateSourcesKey(msg)
	}

	if m.viewMode == viewUpdate {
		return m.updateUpdateKey(msg)
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
		m.sourceNameInput = ""
		m.sourceField = 0
		m.viewMode = viewAddSource
		m.status = "Enter a source path or git URL."
		return m, nil
	case "t":
		return m.openTargetSelection()
	case "r":
		return m.reloadCurrentView("Reloading catalog...")
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
	m.targetPurpose = "install"
	m.status = "Loading install targets..."
	return m, loadTargets(m.repoRoot)
}

func (m model) reloadCurrentView(status string) (tea.Model, tea.Cmd) {
	m.loading = true
	if strings.TrimSpace(status) != "" {
		m.status = status
	}
	switch m.dashboardSection() {
	case viewInstalled:
		m.viewMode = viewInstalled
		m.loadingInstalled = true
		return m, loadInstalled(m.repoRoot)
	case viewUsage:
		m.viewMode = viewUsage
		return m, loadUsage(m.repoRoot)
	case viewSources:
		m.viewMode = viewSources
		return m, loadSources(m.repoRoot)
	case viewTargets:
		m.viewMode = viewTargets
		return m, loadTargets(m.repoRoot)
	case viewUpdate:
		m.loading = false
		m.viewMode = viewUpdate
		m.status = "Update commands."
		return m, nil
	default:
		m.viewMode = viewSkills
		m.loadingSkills = true
		return m, loadSkills(m.repoRoot)
	}
}

func (m model) openDashboardSection(section string) (tea.Model, tea.Cmd) {
	if m.dashboardSection() == viewSkills && section != viewSkills {
		m.clearSkillSearch()
	}
	switch section {
	case viewSkills:
		m.loading = true
		m.loadingSkills = true
		m.viewMode = viewSkills
		m.status = "Loading catalog..."
		return m, loadSkills(m.repoRoot)
	case viewInstalled:
		m.loading = true
		m.loadingInstalled = true
		m.viewMode = viewInstalled
		m.status = "Loading installed skills..."
		return m, loadInstalled(m.repoRoot)
	case viewUsage:
		m.loading = true
		m.viewMode = viewUsage
		m.status = "Loading usage..."
		return m, loadUsage(m.repoRoot)
	case viewSources:
		m.loading = true
		m.viewMode = viewSources
		m.status = "Loading sources..."
		return m, loadSources(m.repoRoot)
	case viewTargets:
		m.loading = true
		m.targetPurpose = "browse"
		m.viewMode = viewTargets
		m.status = "Loading targets..."
		return m, loadTargets(m.repoRoot)
	case viewUpdate:
		m.viewMode = viewUpdate
		m.status = "Update commands."
		return m, nil
	default:
		return m, nil
	}
}

func (m model) moveDashboardSection(delta int) (tea.Model, tea.Cmd) {
	sections := dashboardSections()
	current := m.dashboardSection()
	index := 0
	for i, section := range sections {
		if section == current {
			index = i
			break
		}
	}
	next := (index + delta + len(sections)) % len(sections)
	return m.openDashboardSection(sections[next])
}

func (m *model) clearSkillSearch() {
	m.searchMode = false
	if m.search == "" {
		return
	}
	m.search = ""
	m.applyFilter()
}

func (m model) currentViewLoading() bool {
	switch m.viewMode {
	case viewDetails, viewInstalledDetails, viewUsageDetails, viewInstallResult, viewAddSource, viewConfirmDelete, viewHelp:
		return false
	}
	switch m.dashboardSection() {
	case viewSkills:
		return m.loadingSkills
	case viewInstalled:
		return m.loadingInstalled
	default:
		return m.loading
	}
}

func (m model) canMoveDashboardSection() bool {
	switch m.viewMode {
	case viewSkills, viewInstalled, viewUsage, viewSources, viewUpdate:
		return true
	case viewTargets:
		return m.targetPurpose != "install"
	default:
		return false
	}
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
	queue := buildInstallQueue(choices, names)
	if len(queue) == 0 {
		m.status = "Select at least one supported install target."
		return m, nil
	}
	m.busy = true
	m.pendingInstall = buildInstallResult(choices, names)
	m.installProgress = installProgressState{
		Items:   queue,
		Current: 1,
		Total:   len(queue),
	}
	item, _ := m.installProgress.currentItem()
	m.status = m.installProgressStatus()
	return m, runInstallStepCommand(m.repoRoot, item, m.projectDir)
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
		m.viewMode = viewInstalled
		m.status = "Returned to installed skills."
		return m, nil
	default:
		return m, nil
	}
}

func (m model) updateHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc", "enter", "?":
		if m.previousViewMode != "" {
			m.viewMode = m.previousViewMode
		} else {
			m.viewMode = viewInstalled
		}
		m.status = "Returned."
		return m, nil
	default:
		return m, nil
	}
}

func (m model) updateInstalledKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if m.installedCursor > 0 {
			m.installedCursor--
			m.ensureInstalledCursorVisible()
		}
		return m, nil
	case "down", "j":
		if m.installedCursor < len(m.installedRows)-1 {
			m.installedCursor++
			m.ensureInstalledCursorVisible()
		}
		return m, nil
	case "u":
		row, ok := m.currentInstalled()
		if !ok {
			m.status = "No installed target selected."
			return m, nil
		}
		m.busy = true
		m.status = fmt.Sprintf("Updating %s/%s managed skills...", row.Target, row.Scope)
		return m, runInstalledCommand(m.repoRoot, "Update installed", installedUpdateArgsForRow(row, m.projectDir)...)
	case "x":
		row, ok := m.currentInstalled()
		if !ok {
			m.status = "No installed skill selected."
			return m, nil
		}
		if row.Managed != "yes" {
			m.status = "Unmanaged skills are read-only in TUI. Use CLI --force if needed."
			return m, nil
		}
		m.pendingUninstall = row
		m.viewMode = viewConfirmDelete
		m.status = "Confirm uninstall."
		return m, nil
	case "enter":
		row, ok := m.currentInstalled()
		if !ok {
			m.status = "No installed skill selected."
			return m, nil
		}
		m.installedDetailKey = installedSkillKey(row)
		m.installedDetailCursor = m.indexInstalledDetailRow(row)
		m.installedDetailOffset = 0
		m.ensureInstalledDetailCursorVisible()
		m.viewMode = viewInstalledDetails
		m.status = "Viewing installed " + installedSkillLabel(row) + "."
		return m, nil
	case "r":
		return m.reloadCurrentView("Reloading installed skills...")
	default:
		return m, nil
	}
}

func (m model) updateInstalledDetailsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc", "b":
		m.viewMode = viewInstalled
		m.status = "Returned to installed skills."
		return m, nil
	case "up", "k":
		if m.installedDetailCursor > 0 {
			m.installedDetailCursor--
			m.ensureInstalledDetailCursorVisible()
		}
		return m, nil
	case "down", "j":
		rows := m.installedDetailRows()
		if m.installedDetailCursor < len(rows)-1 {
			m.installedDetailCursor++
			m.ensureInstalledDetailCursorVisible()
		}
		return m, nil
	case "u":
		row, ok := m.currentInstalledDetail()
		if !ok {
			m.status = "No installed location selected."
			return m, nil
		}
		m.busy = true
		m.status = fmt.Sprintf("Updating %s/%s managed skills...", row.Target, row.Scope)
		return m, runInstalledCommand(m.repoRoot, "Update installed", installedUpdateArgsForRow(row, m.projectDir)...)
	case "x":
		row, ok := m.currentInstalledDetail()
		if !ok {
			m.status = "No installed location selected."
			return m, nil
		}
		if row.Managed != "yes" {
			m.status = "Unmanaged skills are read-only in TUI. Use CLI --force if needed."
			return m, nil
		}
		m.pendingUninstall = row
		m.viewMode = viewConfirmDelete
		m.status = "Confirm uninstall."
		return m, nil
	default:
		return m, nil
	}
}

func (m model) updateUsageKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "/":
		m.usageFilterMode = true
		m.status = "Type usage filter."
		return m, nil
	case "esc":
		if strings.TrimSpace(m.usageFilter) != "" {
			m.usageFilter = ""
			m.applyUsageFilter()
			m.status = "Usage filter cleared."
		}
		return m, nil
	case "c":
		if strings.TrimSpace(m.usageFilter) != "" {
			m.usageFilter = ""
			m.applyUsageFilter()
			m.status = "Usage filter cleared."
		}
		return m, nil
	case "up", "k":
		if m.usageCursor > 0 {
			m.usageCursor--
			m.ensureUsageCursorVisible()
		}
		return m, nil
	case "down", "j":
		if m.usageCursor < len(m.usageSummaries)-1 {
			m.usageCursor++
			m.ensureUsageCursorVisible()
		}
		return m, nil
	case "enter":
		summary, ok := m.currentUsageSummary()
		if !ok {
			m.status = "No managed usage selected."
			return m, nil
		}
		m.usageDetailKey = summary.Key
		m.usageDetailCursor = 0
		m.usageDetailOffset = 0
		m.ensureUsageDetailCursorVisible()
		m.viewMode = viewUsageDetails
		m.status = "Viewing usage for " + summary.Key + "."
		return m, nil
	case "u":
		m.returnToUsageDetails = false
		summary, ok := m.currentUsageSummary()
		if !ok {
			m.status = "No managed usage selected."
			return m, nil
		}
		if summary.ProjectCount == 0 {
			m.status = summary.Key + " has no recorded project installs to update."
			return m, nil
		}
		rows := m.projectUsageRowsForKey(summary.Key)
		if len(rows) == 0 {
			m.status = summary.Key + " has no visible recorded project installs to update."
			return m, nil
		}
		m.busy = true
		m.status = "Updating visible project usage for " + summary.Key + "..."
		return m, runUsageBulkUpdateCommand(m.repoRoot, rows)
	case "U":
		m.returnToUsageDetails = false
		rows := m.visibleProjectUsageRows()
		if len(rows) == 0 {
			m.status = "No visible recorded project installs to update."
			return m, nil
		}
		m.busy = true
		m.status = fmt.Sprintf("Updating %d visible project usage row(s)...", len(rows))
		return m, runUsageBulkUpdateCommand(m.repoRoot, rows)
	case "r":
		m.returnToUsageDetails = false
		return m.reloadCurrentView("Reloading usage...")
	default:
		return m, nil
	}
}

func (m model) updateUsageDetailsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "/":
		m.usageFilterMode = true
		m.status = "Type usage filter."
		return m, nil
	case "esc", "b":
		m.viewMode = viewUsage
		m.returnToUsageDetails = false
		m.status = "Returned to usage."
		return m, nil
	case "up", "k":
		if m.usageDetailCursor > 0 {
			m.usageDetailCursor--
			m.ensureUsageDetailCursorVisible()
		}
		return m, nil
	case "down", "j":
		rows := m.usageDetailRows()
		if m.usageDetailCursor < len(rows)-1 {
			m.usageDetailCursor++
			m.ensureUsageDetailCursorVisible()
		}
		return m, nil
	case "u":
		row, ok := m.currentUsageDetail()
		if !ok {
			m.status = "No usage location selected."
			return m, nil
		}
		if !isProjectUsageRow(row) {
			m.status = installedSkillKey(row) + " location is not a recorded project install."
			return m, nil
		}
		m.busy = true
		m.returnToUsageDetails = true
		m.status = "Updating recorded project usage for " + installedSkillKey(row) + "..."
		return m, runInstalledCommand(m.repoRoot, "Update usage", usageUpdateArgsForLocation(row)...)
	case "U":
		rows := projectUsageRows(m.usageDetailRows())
		if len(rows) == 0 {
			m.status = m.usageDetailKey + " has no visible recorded project installs to update."
			return m, nil
		}
		m.busy = true
		m.returnToUsageDetails = true
		m.status = fmt.Sprintf("Updating %d visible project usage row(s)...", len(rows))
		return m, runUsageBulkUpdateCommand(m.repoRoot, rows)
	case "r":
		m.loading = true
		m.returnToUsageDetails = true
		m.status = "Reloading usage..."
		return m, loadUsage(m.repoRoot)
	default:
		return m, nil
	}
}

func (m model) updateConfirmDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc", "n":
		m.viewMode = viewInstalled
		m.status = "Uninstall cancelled."
		return m, nil
	case "enter", "y":
		if strings.TrimSpace(m.pendingUninstall.Skill) == "" {
			m.viewMode = viewInstalled
			m.status = "No uninstall target selected."
			return m, nil
		}
		row := m.pendingUninstall
		m.pendingUninstall = InstalledSkill{}
		m.busy = true
		m.viewMode = viewInstalled
		m.status = "Uninstalling " + row.Skill + "..."
		return m, runInstalledCommand(m.repoRoot, "Uninstall", installedUninstallArgsForRow(row, m.projectDir)...)
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
		if m.targetPurpose != "install" {
			m.status = "Select skills in 2 Skills, then press i to install."
			return m, nil
		}
		return m.installToSelectedTargets()
	case "r":
		m.loading = true
		m.status = "Reloading targets..."
		return m, loadTargets(m.repoRoot)
	default:
		return m, nil
	}
}

func (m model) updateSourcesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "d":
		m.loading = true
		m.status = "Loading source defaults..."
		return m, loadDefaultSources(m.repoRoot)
	case "n":
		m.sourceInput = ""
		m.sourceNameInput = ""
		m.sourceField = 0
		m.viewMode = viewAddSource
		m.status = "Enter a source path or git URL."
		return m, nil
	case "s":
		m.busy = true
		m.reloadOnFinish = true
		m.status = "Syncing sources..."
		return m, runSourceCommand(m.repoRoot, "Sync", "sync")
	case "r":
		return m.reloadCurrentView("Reloading sources...")
	default:
		return m, nil
	}
}

func (m model) updateUpdateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "u":
		m.loading = true
		m.viewMode = viewInstalled
		m.status = "Loading installed skills..."
		return m, loadInstalled(m.repoRoot)
	default:
		return m, nil
	}
}

func (m model) updateAddSourceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes {
		if m.sourceField == 1 {
			m.sourceNameInput += msg.String()
		} else {
			m.sourceInput += msg.String()
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.viewMode = viewSkills
		m.status = "Returned to skills."
		return m, nil
	case "tab", "down":
		m.sourceField = (m.sourceField + 1) % 2
		return m, nil
	case "shift+tab", "up":
		m.sourceField = (m.sourceField + 1) % 2
		return m, nil
	case "enter":
		location := normalizeSourceLocationInput(m.sourceInput)
		if location == "" {
			m.status = "Enter a source path or git URL before adding."
			return m, nil
		}
		args := sourceAddArgs(location, m.sourceNameInput)
		m.busy = true
		m.reloadOnFinish = true
		m.status = "Adding source " + location + "..."
		return m, runSourceCommand(m.repoRoot, "Add source", args...)
	case "backspace":
		if m.sourceField == 1 {
			if m.sourceNameInput != "" {
				runes := []rune(m.sourceNameInput)
				m.sourceNameInput = string(runes[:len(runes)-1])
			}
		} else if m.sourceInput != "" {
			runes := []rune(m.sourceInput)
			m.sourceInput = string(runes[:len(runes)-1])
		}
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
		m.sourceNameInput = ""
		m.sourceField = 0
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
	header := m.renderHeader(contentWidth)
	navigation := m.renderNavigation(contentWidth)
	status := statusStyle.Width(contentWidth).Render(m.status)
	help := helpStyle.Width(contentWidth).Render(m.helpText())

	bodyModel := m
	if m.height > 0 {
		innerHeight := m.height - appStyle.GetPaddingTop() - appStyle.GetPaddingBottom()
		if innerHeight < 1 {
			innerHeight = 1
		}
		reservedHeight := lipgloss.Height(header) + lipgloss.Height(navigation) + lipgloss.Height(status) + lipgloss.Height(help)
		bodyModel.bodyHeight = innerHeight - reservedHeight
		if bodyModel.bodyHeight < 0 {
			bodyModel.bodyHeight = 0
		}
	}
	body := bodyModel.renderBody(contentWidth)

	parts := []string{
		header,
		navigation,
	}
	if bodyModel.bodyHeight != 0 || m.height <= 0 {
		parts = append(parts, body)
	}
	parts = append(parts, status, help)

	rendered := appStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
	if m.height > 0 && lipgloss.Height(rendered) > m.height {
		rendered = lipgloss.NewStyle().MaxHeight(m.height).Render(rendered)
	}
	return rendered
}

func (m model) renderHeader(width int) string {
	search := subtleStyle.Render("none")
	if m.searchMode {
		search = searchBadgeStyle.Render(m.search + "_")
	} else if strings.TrimSpace(m.search) != "" {
		search = searchBadgeStyle.Render(m.search)
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
		if m.targetPurpose == "install" {
			target = fmt.Sprintf("Install targets: %d selected   Project: %s", len(m.selectedTargetChoices()), m.projectDir)
		} else {
			target = fmt.Sprintf("Targets   Project: %s", m.projectDir)
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, subtleStyle.Render(summary), subtleStyle.Render(target))
}

func (m model) renderNavigation(width int) string {
	items := []struct {
		key     string
		label   string
		section string
	}{
		{"1", "Installed", viewInstalled},
		{"2", "Skills", viewSkills},
		{"3", "Usage", viewUsage},
		{"4", "Sources", viewSources},
		{"5", "Targets", viewTargets},
		{"6", "Update", viewUpdate},
	}
	rendered := make([]string, 0, len(items))
	current := m.dashboardSection()
	for _, item := range items {
		label := item.key + " " + item.label
		if item.section == current {
			rendered = append(rendered, badgeStyle.Render(label))
		} else {
			rendered = append(rendered, subtleStyle.Render(label))
		}
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(rendered, "  "))
}

func (m model) renderBody(width int) string {
	if m.installProgress.Total > 0 && (m.busy || m.installProgress.Failed) {
		title := "Install progress"
		if m.installProgress.Failed {
			title = "Install failed"
		}
		return m.renderPanel(title, m.installProgressContent(width-6), width)
	}
	if m.loading {
		return m.renderPanel(sectionTitle(m.dashboardSection()), statusStyle.Render(m.status), width)
	}
	if m.viewMode == viewHelp {
		return m.renderPanel("Help", m.helpContent(width-6), width)
	}
	if m.viewMode == viewAddSource {
		return m.renderPanel("Add source", m.addSourceContent(width-6), width)
	}
	if m.viewMode == viewConfirmDelete {
		return m.renderPanel("Confirm uninstall", m.confirmDeleteContent(width-6), width)
	}
	if m.viewMode == viewDetails {
		return m.renderPanel("Skill details", m.detailsContent(width-6), width)
	}
	if m.viewMode == viewInstalledDetails {
		return m.renderPanel("Installed skill details", m.installedDetailsContent(width-6), width)
	}
	if m.viewMode == viewUsageDetails {
		return m.renderPanel("Usage details", m.usageDetailsContent(width-6), width)
	}
	if m.viewMode == viewInstallResult {
		return m.renderPanel("Install complete", m.installResultContent(width-6), width)
	}
	if m.viewMode == viewTargets {
		return m.renderPanel(m.targetsPanelTitle(), m.targetsContent(width-6), width)
	}
	if m.viewMode == viewInstalled {
		return m.renderPanel("Installed skills", m.installedContent(width-6), width)
	}
	if m.viewMode == viewUsage {
		return m.renderPanel("Usage", m.usageContent(width-6), width)
	}
	if m.viewMode == viewSources {
		return m.renderPanel("Sources", m.sourcesContent(width-6), width)
	}
	if m.viewMode == viewUpdate {
		return m.renderPanel("Update", m.updateContent(width-6), width)
	}
	if m.viewMode == viewDefaults {
		return m.renderPanel("Source presets", m.defaultsContent(width-6), width)
	}
	if m.noSources {
		return m.renderPanel("Sources", "No sources configured.\n\nPress d for presets or n for a custom source.", width)
	}
	if len(m.filtered) == 0 {
		return m.renderPanel("Skills", "No skills matched.", width)
	}

	return m.renderPanel("Skills", m.skillsContent(width-6), width)
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
		locations := m.installedLocationsForSkill(skill)
		if len(locations) > 0 {
			title += "   " + projectBadgeStyle.Render("installed: "+installedLocationsBadge(locations))
		}
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
		labelLine("Installed", installedLocationsBadge(m.installedLocationsForSkill(skill))),
		"",
		wrapText(skill.Description, width),
	}
	if locations := m.installedLocationsForSkill(skill); len(locations) > 0 {
		lines = append(lines, "", titleStyle.Render("Installed locations"), m.installedLocationsContent(locations, width))
	}
	return strings.Join(lines, "\n")
}

func (m model) installedLocationsForSkill(skill Skill) []InstalledSkill {
	key := skill.Key()
	rows := []InstalledSkill{}
	for _, row := range m.installedRows {
		if row.Managed != "yes" {
			continue
		}
		if strings.TrimSpace(row.Source) == "" || row.Source == "-" {
			continue
		}
		if row.Source+"/"+row.Skill != key && row.QualifiedSkill != key {
			continue
		}
		rows = append(rows, row)
	}
	sortInstalledRows(rows)
	return rows
}

func installedLocationsBadge(rows []InstalledSkill) string {
	if len(rows) == 0 {
		return "none"
	}
	if len(rows) == 1 {
		return targetScopeLabel(rows[0].Target, rows[0].Scope)
	}
	return fmt.Sprintf("%d locations", len(rows))
}

func (m model) installedLocationsContent(rows []InstalledSkill, width int) string {
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			fmt.Fprintln(&b)
		}
		metaParts := []string{"managed"}
		if row.RegistryOnly {
			metaParts = append(metaParts, "registry")
		}
		fmt.Fprintln(&b, "- "+targetScopeLabel(row.Target, row.Scope))
		if row.Scope == "project" && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			fmt.Fprintln(&b, "  project: "+truncate(row.ProjectPath, max(12, width-11)))
		}
		path := row.InstalledPath
		if strings.TrimSpace(path) == "" || path == "-" {
			path = row.Path
		}
		if strings.TrimSpace(path) != "" && path != "-" {
			fmt.Fprintln(&b, "  path: "+truncate(path, max(12, width-8)))
		}
		fmt.Fprintln(&b, "  source: "+emptyLabel(row.Source, "-")+"   "+strings.Join(metaParts, " "))
	}
	return strings.TrimRight(b.String(), "\n")
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

func (m model) installedContent(width int) string {
	legend := helpStyle.Render("[M] managed by Skillhub: update/uninstall   [ ] unmanaged: read-only in TUI   registry: recorded usage")
	if len(m.installedRows) == 0 {
		return legend + "\n\nNo installed skills found.\n\nInstall skills from 2 Skills, or run skillhub installed list in CLI."
	}

	var b strings.Builder
	stats := installedOverviewStats(m.installedRows)
	fmt.Fprintf(&b, "%s   %s   %s   %s\n",
		badgeStyle.Render(fmt.Sprintf("Installed: %d", stats.Installed)),
		badgeStyle.Render(fmt.Sprintf("Skills: %d", stats.Skills)),
		badgeStyle.Render(fmt.Sprintf("Projects: %d", stats.Projects)),
		badgeStyle.Render(fmt.Sprintf("Managed: %d", stats.Managed)),
	)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, legend)
	fmt.Fprintln(&b)
	visible := m.installedVisibleCount()
	end := m.installedOffset + visible
	if end > len(m.installedRows) {
		end = len(m.installedRows)
	}
	previousSection := ""
	previousSubgroup := ""
	for row, installed := range m.installedRows[m.installedOffset:end] {
		section := installedSection(installed)
		subgroup := installedSubgroup(installed)
		if section != previousSection {
			if row != 0 {
				fmt.Fprintln(&b)
			}
			fmt.Fprintln(&b, categoryStyle.Render(section))
			previousSection = section
			previousSubgroup = ""
		}
		if subgroup != "" && subgroup != previousSubgroup {
			fmt.Fprintln(&b, treeStyle.Render("  "+subgroup))
			previousSubgroup = subgroup
		}

		i := m.installedOffset + row
		cursor := " "
		if i == m.installedCursor {
			cursor = "›"
		}
		marker := checkboxStyle.Render("[ ]")
		managedLabel := "unmanaged"
		if installed.Managed == "yes" {
			marker = checkedStyle.Render("[M]")
			managedLabel = "managed"
		}
		indent := "  "
		if subgroup != "" {
			indent = "    "
		}
		title := fmt.Sprintf("%s%s %s %s    %s", indent, cursor, marker, installed.Skill, targetScopeLabel(installed.Target, installed.Scope))
		if i == m.installedCursor {
			title = activeRowStyle.Render(title)
		}
		if installed.Managed == "yes" {
			title = selectedRowStyle.Render(title)
		}
		metaParts := []string{managedLabel}
		if installed.RegistryOnly {
			metaParts = append(metaParts, "registry")
		}
		if strings.TrimSpace(installed.ProjectPath) != "" && installed.ProjectPath != "-" {
			metaParts = append(metaParts, "project: "+truncate(installed.ProjectPath, max(12, width-30)))
		}
		meta := fmt.Sprintf("%ssource: %s   %s", indent, emptyLabel(installed.Source, "-"), strings.Join(metaParts, "   "))
		path := indent + truncate(installed.Path, max(12, width-len(indent)))
		fmt.Fprintln(&b, title)
		fmt.Fprintln(&b, subtleStyle.Render(meta))
		fmt.Fprintln(&b, subtleStyle.Render(path))
		if row != end-m.installedOffset-1 {
			fmt.Fprintln(&b)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) installedDetailsContent(width int) string {
	rows := m.installedDetailRows()
	if len(rows) == 0 {
		return "No installed locations found."
	}

	if m.installedDetailCursor >= len(rows) {
		m.installedDetailCursor = len(rows) - 1
	}
	if m.installedDetailCursor < 0 {
		m.installedDetailCursor = 0
	}

	var b strings.Builder
	fmt.Fprintln(&b, titleStyle.Render(installedSkillLabel(rows[0])))
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "%s\n\n", badgeStyle.Render(fmt.Sprintf("Installed in: %d", len(rows))))

	visible := m.installedDetailVisibleCount()
	end := m.installedDetailOffset + visible
	if end > len(rows) {
		end = len(rows)
	}
	for rowIndex, row := range rows[m.installedDetailOffset:end] {
		i := m.installedDetailOffset + rowIndex
		cursor := " "
		if i == m.installedDetailCursor {
			cursor = "›"
		}
		marker := checkboxStyle.Render("[ ]")
		if row.Managed == "yes" {
			marker = checkedStyle.Render("[M]")
		}
		title := fmt.Sprintf("%s %s %s", cursor, marker, targetScopeLabel(row.Target, row.Scope))
		if row.Scope == "project" && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			title = fmt.Sprintf("%s %s %s %s", cursor, marker, projectBadgeStyle.Render("LOCAL PROJECT"), targetScopeLabel(row.Target, row.Scope))
		}
		if i == m.installedDetailCursor {
			title = activeRowStyle.Render(title)
		}
		if row.Managed == "yes" {
			title = selectedRowStyle.Render(title)
		}
		fmt.Fprintln(&b, title)
		if strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			fmt.Fprintln(&b, "    "+projectBadgeStyle.Render("Project root: "+truncate(row.ProjectPath, max(12, width-18))))
		}
		fmt.Fprintln(&b, subtleStyle.Render("    path: "+truncate(row.Path, max(12, width-10))))
		metaParts := []string{fmt.Sprintf("source: %s", emptyLabel(row.Source, "-"))}
		if strings.TrimSpace(row.ContentHash) != "" && row.ContentHash != "-" {
			metaParts = append(metaParts, "hash: "+row.ContentHash)
		}
		if strings.TrimSpace(row.InstalledAt) != "" && row.InstalledAt != "-" {
			metaParts = append(metaParts, "installed: "+row.InstalledAt)
		}
		if strings.TrimSpace(row.UpdatedAt) != "" && row.UpdatedAt != "-" {
			metaParts = append(metaParts, "updated: "+row.UpdatedAt)
		}
		meta := indent(wrapText(strings.Join(metaParts, "   "), max(12, width-4)), "    ")
		fmt.Fprintln(&b, subtleStyle.Render(meta))
		if rowIndex != end-m.installedDetailOffset-1 {
			fmt.Fprintln(&b)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) usageContent(width int) string {
	legend := helpStyle.Render("Registry usage only: managed installs created or updated by Skillhub")
	visibleRows := m.visibleUsageRows()
	if len(m.usageSummaries) == 0 {
		if strings.TrimSpace(m.usageFilter) != "" {
			return legend + "\n\n" + labelLine("Filter", m.usageFilter) + "\n\nNo managed usage matched the filter."
		}
		return legend + "\n\nNo managed usage recorded.\n\nInstall a skill through Skillhub to create usage records."
	}

	var b strings.Builder
	stats := installedOverviewStats(visibleRows)
	fmt.Fprintf(&b, "%s   %s   %s   %s\n",
		badgeStyle.Render(fmt.Sprintf("Installs: %d", stats.Installed)),
		badgeStyle.Render(fmt.Sprintf("Skills: %d", stats.Skills)),
		badgeStyle.Render(fmt.Sprintf("Projects: %d", stats.Projects)),
		badgeStyle.Render(fmt.Sprintf("Managed: %d", stats.Managed)),
	)
	fmt.Fprintln(&b)
	filter := emptyLabel(m.usageFilter, "none")
	if m.usageFilterMode {
		filter = m.usageFilter + "_"
	}
	fmt.Fprintf(&b, "%s   %s\n", labelLine("Filter", filter), labelLine("Visible", fmt.Sprintf("%d/%d", len(visibleRows), len(m.usageRows))))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, legend)
	fmt.Fprintln(&b)

	visible := m.usageVisibleCount()
	end := m.usageOffset + visible
	if end > len(m.usageSummaries) {
		end = len(m.usageSummaries)
	}
	for rowIndex, summary := range m.usageSummaries[m.usageOffset:end] {
		i := m.usageOffset + rowIndex
		cursor := " "
		if i == m.usageCursor {
			cursor = "›"
		}
		title := fmt.Sprintf("%s %s", cursor, summary.Key)
		if i == m.usageCursor {
			title = activeRowStyle.Render(title)
		}
		if summary.ProjectCount > 0 {
			title = selectedRowStyle.Render(title)
		}
		meta := fmt.Sprintf("    installs: %d   projects: %d   targets: %d",
			summary.InstallCount,
			summary.ProjectCount,
			summary.TargetCount,
		)
		if strings.TrimSpace(summary.LatestUpdated) != "" && summary.LatestUpdated != "-" {
			meta += "   latest: " + summary.LatestUpdated
		}
		fmt.Fprintln(&b, title)
		fmt.Fprintln(&b, subtleStyle.Render(meta))
		if rowIndex != end-m.usageOffset-1 {
			fmt.Fprintln(&b)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) usageDetailsContent(width int) string {
	rows := m.usageDetailRows()
	if len(rows) == 0 {
		return "No usage locations found."
	}
	summary := summarizeUsageRows(installedSkillKey(rows[0]), rows)

	var b strings.Builder
	fmt.Fprintln(&b, titleStyle.Render(summary.Key))
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "%s   %s   %s\n\n",
		badgeStyle.Render(fmt.Sprintf("Installed in: %d", summary.InstallCount)),
		badgeStyle.Render(fmt.Sprintf("Projects: %d", summary.ProjectCount)),
		badgeStyle.Render(fmt.Sprintf("Targets: %d", summary.TargetCount)),
	)
	if strings.TrimSpace(m.usageFilter) != "" || m.usageFilterMode {
		filter := m.usageFilter
		if m.usageFilterMode {
			filter += "_"
		}
		fmt.Fprintf(&b, "%s\n\n", labelLine("Filter", emptyLabel(filter, "none")))
	}

	visible := m.usageDetailVisibleCount()
	end := m.usageDetailOffset + visible
	if end > len(rows) {
		end = len(rows)
	}
	for rowIndex, row := range rows[m.usageDetailOffset:end] {
		i := m.usageDetailOffset + rowIndex
		cursor := " "
		if i == m.usageDetailCursor {
			cursor = "›"
		}
		title := fmt.Sprintf("%s %s", cursor, targetScopeLabel(row.Target, row.Scope))
		if row.Scope == "project" && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			title = fmt.Sprintf("%s %s %s", cursor, projectBadgeStyle.Render("LOCAL PROJECT"), targetScopeLabel(row.Target, row.Scope))
		}
		if i == m.usageDetailCursor {
			title = activeRowStyle.Render(title)
		}
		fmt.Fprintln(&b, title)
		if strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			fmt.Fprintln(&b, "    "+projectBadgeStyle.Render("Project root: "+truncate(row.ProjectPath, max(12, width-18))))
		}
		fmt.Fprintln(&b, subtleStyle.Render("    path: "+truncate(row.Path, max(12, width-10))))
		metaParts := []string{fmt.Sprintf("source: %s", emptyLabel(row.Source, "-"))}
		if strings.TrimSpace(row.ContentHash) != "" && row.ContentHash != "-" {
			metaParts = append(metaParts, "hash: "+row.ContentHash)
		}
		if strings.TrimSpace(row.InstalledAt) != "" && row.InstalledAt != "-" {
			metaParts = append(metaParts, "installed: "+row.InstalledAt)
		}
		if strings.TrimSpace(row.UpdatedAt) != "" && row.UpdatedAt != "-" {
			metaParts = append(metaParts, "updated: "+row.UpdatedAt)
		}
		meta := indent(wrapText(strings.Join(metaParts, "   "), max(12, width-4)), "    ")
		fmt.Fprintln(&b, subtleStyle.Render(meta))
		if rowIndex != end-m.usageDetailOffset-1 {
			fmt.Fprintln(&b)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) installProgressContent(width int) string {
	item, ok := m.installProgress.currentItem()
	if !ok {
		return "No install step is running."
	}

	title := fmt.Sprintf("Installing %d/%d", m.installProgress.Current, m.installProgress.Total)

	var b strings.Builder
	fmt.Fprintln(&b, titleStyle.Render(title))
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "Skill %s\n", item.Skill)
	fmt.Fprintf(&b, "Target %s\n", item.Choice.Label)
	if strings.TrimSpace(item.Choice.Path) != "" {
		fmt.Fprintf(&b, "Path %s\n", truncate(item.Choice.Path, max(12, width-5)))
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, installProgressBar(m.installProgress.Completed, m.installProgress.Total, max(12, width-8)))
	if strings.TrimSpace(m.installProgress.LastLine) != "" {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "Last result")
		fmt.Fprintln(&b, indent(wrapText(m.installProgress.LastLine, max(12, width-2)), "  "))
	}
	if strings.TrimSpace(m.installProgress.Error) != "" {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "Error")
		fmt.Fprintln(&b, indent(wrapText(m.installProgress.Error, max(12, width-2)), "  "))
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

func (m model) sourcesContent(width int) string {
	if len(m.sources) == 0 {
		return "No sources configured.\n\nPress d for recommended presets or n for a custom source."
	}
	var b strings.Builder
	for i, source := range m.sources {
		if i > 0 {
			fmt.Fprintln(&b)
		}
		fmt.Fprintln(&b, titleStyle.Render(source.Name))
		fmt.Fprintln(&b, subtleStyle.Render(fmt.Sprintf("    %s %s   catalog: %s", source.Type, source.Ref, source.Catalog)))
		fmt.Fprintln(&b, "    "+truncate(source.Location, max(16, width-4)))
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, helpStyle.Render("d presets  n custom source  s sync  r reload"))
	return strings.TrimRight(b.String(), "\n")
}

func (m model) updateContent(width int) string {
	lines := []string{
		titleStyle.Render("Update commands"),
		"",
		"Self-update is CLI-only so the TUI does not rewrite the binary while it is running.",
		"",
		labelLine("Command", "skillhub update"),
		labelLine("Cascade", "skillhub update --cascade"),
		labelLine("Verbose", "skillhub update --cascade -v"),
		labelLine("Skills", "skillhub installed update"),
		labelLine("Projects", "skillhub installed usage update --projects"),
		"",
		wrapText("Use 1 Installed to update target folders, or 3 Usage to update recorded project installs for a highlighted skill.", width),
	}
	return strings.Join(lines, "\n")
}

func (m model) confirmDeleteContent(width int) string {
	row := m.pendingUninstall
	if strings.TrimSpace(row.Skill) == "" {
		return "No managed skill selected."
	}
	lines := []string{
		titleStyle.Render("Remove managed skill?"),
		"",
		labelLine("Skill", row.Skill),
		labelLine("Target", targetScopeLabel(row.Target, row.Scope)),
		labelLine("Path", row.Path),
		"",
		wrapText("This removes only the selected managed skill directory. Unmanaged skills remain CLI-only.", width),
		"",
		checkedStyle.Render("enter/y confirm") + "  " + helpStyle.Render("esc/n cancel"),
	}
	return strings.Join(lines, "\n")
}

func (m model) helpContent(width int) string {
	lines := []string{
		"1/2/3/4/5/6 switch sections",
		"left/right  switch sections",
		"j/k         move",
		"space       select or toggle where applicable",
		"enter       open or confirm",
		"/           search skills or filter usage rows",
		"u           update highlighted install or usage entry",
		"U           update all visible recorded project installs in Usage",
		"x           uninstall highlighted managed skill",
		"s           sync sources",
		"r           reload current section",
		"esc         back",
		"q           quit",
		"",
		"[M]         managed by Skillhub; update/uninstall available",
		"[ ]         unmanaged local skill; read-only in TUI",
	}
	return strings.Join(lines, "\n")
}

func (m model) addSourceContent(width int) string {
	input := m.sourceInput
	nameInput := m.sourceNameInput
	locationCursor := ""
	nameCursor := ""
	if m.sourceField == 1 {
		nameCursor = "_"
	} else {
		locationCursor = "_"
	}
	return strings.Join([]string{
		labelLine("Location", input+locationCursor),
		labelLine("Name", nameInput+nameCursor),
		"",
		wrapText("Location expects a local path or git URL, for example https://github.com/mattpocock/skills. Name is optional; use a short id such as mattpocock for generic repository names. Tab switches fields.", width),
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
	if m.installProgress.Failed {
		return "enter/esc back to targets  q quit"
	}
	if m.usageFilterMode {
		return "type filter  enter apply  backspace delete  esc clear  ctrl+c quit"
	}
	if m.viewMode == viewHelp {
		return "enter/esc back  q quit"
	}
	if m.viewMode == viewDefaults {
		return "j/k move  enter add preset  n add custom  esc back  q quit"
	}
	if m.viewMode == viewAddSource {
		return "tab field  enter add  esc back  ctrl+c quit"
	}
	if m.viewMode == viewDetails {
		return "space select  i targets  enter/esc back  q quit"
	}
	if m.viewMode == viewInstalledDetails {
		return "j/k move location  u update  x uninstall  esc back  q quit"
	}
	if m.viewMode == viewUsageDetails {
		return "j/k move location  / filter  u update location  U update visible  r reload  esc back  q quit"
	}
	if m.viewMode == viewInstallResult {
		return "enter/b back  t targets  q quit"
	}
	if m.viewMode == viewConfirmDelete {
		return "enter/y confirm  esc/n cancel  q quit"
	}
	if m.viewMode == viewInstalled {
		return "1-6/left-right sections  j/k move  enter details  u update target  x uninstall managed  r reload  ? help  q quit"
	}
	if m.viewMode == viewUsage {
		return "1-6/left-right sections  j/k move  / filter  enter details  u update skill  U update visible  r reload  ? help  q quit"
	}
	if m.viewMode == viewSources {
		return "1-6/left-right sections  d presets  n custom  s sync  r reload  ? help  q quit"
	}
	if m.viewMode == viewUpdate {
		return "1-6/left-right sections  u installed screen  ? help  q quit"
	}
	if m.viewMode == viewTargets {
		if m.targetPurpose == "install" {
			return "1-6 sections  j/k move  space toggle  enter/i install  a all  c clear  r reload  ? help  q quit"
		}
		return "1-6/left-right sections  j/k move  r reload  ? help  q quit"
	}
	return "1-6/left-right sections  j/k move  space select  enter details  / search  a all  c clear  d presets  n source  t targets  i install  ? help  q quit"
}

func (m model) targetsPanelTitle() string {
	if m.targetPurpose == "install" {
		return "Install targets"
	}
	return "Targets"
}

func (m model) renderPanel(title, body string, width int) string {
	return panelWithMaxHeight(title, body, width, m.bodyHeight)
}

func panel(title, body string, width int) string {
	return panelWithMaxHeight(title, body, width, 0)
}

func panelWithMaxHeight(title, body string, width int, maxHeight int) string {
	if width < 24 {
		width = 24
	}
	contentWidth := width - 6
	if contentWidth < 10 {
		contentWidth = 10
	}
	if maxHeight > 0 {
		bodyBudget := maxHeight - panelChromeHeight()
		if bodyBudget < 0 {
			bodyBudget = 0
		}
		body = limitLines(body, bodyBudget)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, panelTitleStyle.Render(title), "", body)
	rendered := panelStyle.Width(contentWidth).Render(content)
	if maxHeight > 0 && lipgloss.Height(rendered) > maxHeight {
		rendered = lipgloss.NewStyle().MaxHeight(maxHeight).Render(rendered)
	}
	return rendered
}

func panelChromeHeight() int {
	return 6
}

func limitLines(value string, maxLines int) string {
	if maxLines < 0 {
		maxLines = 0
	}
	if maxLines == 0 || strings.TrimSpace(value) == "" {
		return ""
	}
	lines := strings.Split(value, "\n")
	if len(lines) <= maxLines {
		return value
	}
	return strings.Join(lines[:maxLines], "\n")
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

func (m *model) applyUsageFilter() {
	rows := m.visibleUsageRows()
	m.usageSummaries = buildUsageSummaries(rows)
	if m.usageCursor >= len(m.usageSummaries) {
		m.usageCursor = len(m.usageSummaries) - 1
	}
	if m.usageCursor < 0 {
		m.usageCursor = 0
	}
	if m.usageDetailCursor >= len(m.usageDetailRows()) {
		m.usageDetailCursor = len(m.usageDetailRows()) - 1
	}
	if m.usageDetailCursor < 0 {
		m.usageDetailCursor = 0
	}
	m.usageOffset = 0
	m.usageDetailOffset = 0
	m.ensureUsageCursorVisible()
	m.ensureUsageDetailCursorVisible()
}

func (m model) visibleUsageRows() []InstalledSkill {
	rows := []InstalledSkill{}
	for _, row := range m.usageRows {
		if usageRowMatchesFilter(row, m.usageFilter) {
			rows = append(rows, row)
		}
	}
	return rows
}

func usageRowMatchesFilter(row InstalledSkill, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		row.Source,
		row.Skill,
		installedSkillKey(row),
		row.Target,
		row.Scope,
		row.ProjectPath,
		row.TargetRoot,
		row.InstalledPath,
		row.Path,
		row.ContentHash,
	}, " "))
	return strings.Contains(haystack, q)
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

func (m *model) ensureInstalledCursorVisible() {
	visible := m.installedVisibleCount()
	if m.installedCursor < m.installedOffset {
		m.installedOffset = m.installedCursor
	}
	if m.installedCursor >= m.installedOffset+visible {
		m.installedOffset = m.installedCursor - visible + 1
	}
	if m.installedOffset < 0 {
		m.installedOffset = 0
	}
	maxOffset := len(m.installedRows) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.installedOffset > maxOffset {
		m.installedOffset = maxOffset
	}
}

func (m *model) ensureInstalledDetailCursorVisible() {
	visible := m.installedDetailVisibleCount()
	if m.installedDetailCursor < m.installedDetailOffset {
		m.installedDetailOffset = m.installedDetailCursor
	}
	if m.installedDetailCursor >= m.installedDetailOffset+visible {
		m.installedDetailOffset = m.installedDetailCursor - visible + 1
	}
	if m.installedDetailOffset < 0 {
		m.installedDetailOffset = 0
	}
	maxOffset := len(m.installedDetailRows()) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.installedDetailOffset > maxOffset {
		m.installedDetailOffset = maxOffset
	}
}

func (m *model) ensureUsageCursorVisible() {
	visible := m.usageVisibleCount()
	if m.usageCursor < m.usageOffset {
		m.usageOffset = m.usageCursor
	}
	if m.usageCursor >= m.usageOffset+visible {
		m.usageOffset = m.usageCursor - visible + 1
	}
	if m.usageOffset < 0 {
		m.usageOffset = 0
	}
	maxOffset := len(m.usageSummaries) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.usageOffset > maxOffset {
		m.usageOffset = maxOffset
	}
}

func (m *model) ensureUsageDetailCursorVisible() {
	visible := m.usageDetailVisibleCount()
	if m.usageDetailCursor < m.usageDetailOffset {
		m.usageDetailOffset = m.usageDetailCursor
	}
	if m.usageDetailCursor >= m.usageDetailOffset+visible {
		m.usageDetailOffset = m.usageDetailCursor - visible + 1
	}
	if m.usageDetailOffset < 0 {
		m.usageDetailOffset = 0
	}
	maxOffset := len(m.usageDetailRows()) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.usageDetailOffset > maxOffset {
		m.usageDetailOffset = maxOffset
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

func (m model) installedVisibleCount() int {
	if m.height <= 0 {
		return 6
	}
	count := (m.height - 14) / 4
	if count < 1 {
		return 1
	}
	return count
}

func (m model) installedDetailVisibleCount() int {
	if m.height <= 0 {
		return 4
	}
	count := (m.height - 16) / 5
	if count < 1 {
		return 1
	}
	return count
}

func (m model) usageVisibleCount() int {
	if m.height <= 0 {
		return 6
	}
	count := (m.height - 14) / 3
	if count < 1 {
		return 1
	}
	return count
}

func (m model) usageDetailVisibleCount() int {
	if m.height <= 0 {
		return 4
	}
	count := (m.height - 16) / 5
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

func (m model) currentInstalled() (InstalledSkill, bool) {
	if len(m.installedRows) == 0 || m.installedCursor < 0 || m.installedCursor >= len(m.installedRows) {
		return InstalledSkill{}, false
	}
	return m.installedRows[m.installedCursor], true
}

func (m model) currentInstalledDetail() (InstalledSkill, bool) {
	rows := m.installedDetailRows()
	if len(rows) == 0 || m.installedDetailCursor < 0 || m.installedDetailCursor >= len(rows) {
		return InstalledSkill{}, false
	}
	return rows[m.installedDetailCursor], true
}

func (m model) installedDetailRows() []InstalledSkill {
	key := m.installedDetailKey
	if strings.TrimSpace(key) == "" {
		if row, ok := m.currentInstalled(); ok {
			key = installedSkillKey(row)
		}
	}
	rows := []InstalledSkill{}
	for _, row := range m.installedRows {
		if installedSkillKey(row) == key {
			rows = append(rows, row)
		}
	}
	sortInstalledRows(rows)
	return rows
}

func (m model) indexInstalledDetailRow(wanted InstalledSkill) int {
	key := installedRowKey(wanted)
	for i, row := range m.installedDetailRows() {
		if installedRowKey(row) == key {
			return i
		}
	}
	return 0
}

func (m model) currentUsageSummary() (UsageSummary, bool) {
	if len(m.usageSummaries) == 0 || m.usageCursor < 0 || m.usageCursor >= len(m.usageSummaries) {
		return UsageSummary{}, false
	}
	return m.usageSummaries[m.usageCursor], true
}

func (m model) currentUsageDetail() (InstalledSkill, bool) {
	rows := m.usageDetailRows()
	if len(rows) == 0 || m.usageDetailCursor < 0 || m.usageDetailCursor >= len(rows) {
		return InstalledSkill{}, false
	}
	return rows[m.usageDetailCursor], true
}

func (m model) usageDetailRows() []InstalledSkill {
	key := m.usageDetailKey
	if strings.TrimSpace(key) == "" {
		if summary, ok := m.currentUsageSummary(); ok {
			key = summary.Key
		}
	}
	rows := []InstalledSkill{}
	for _, row := range m.visibleUsageRows() {
		if installedSkillKey(row) == key {
			rows = append(rows, row)
		}
	}
	sortInstalledRows(rows)
	return rows
}

func (m model) projectUsageRowsForKey(key string) []InstalledSkill {
	rows := []InstalledSkill{}
	for _, row := range m.visibleUsageRows() {
		if installedSkillKey(row) == key && isProjectUsageRow(row) {
			rows = append(rows, row)
		}
	}
	return rows
}

func (m model) visibleProjectUsageRows() []InstalledSkill {
	return projectUsageRows(m.visibleUsageRows())
}

func projectUsageRows(rows []InstalledSkill) []InstalledSkill {
	result := []InstalledSkill{}
	for _, row := range rows {
		if isProjectUsageRow(row) {
			result = append(result, row)
		}
	}
	return result
}

func isProjectUsageRow(row InstalledSkill) bool {
	return row.Scope == "project" && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-"
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

func (m model) dashboardSection() string {
	switch m.viewMode {
	case viewDetails, viewInstallResult:
		return viewSkills
	case viewInstalledDetails:
		return viewInstalled
	case viewUsageDetails:
		return viewUsage
	case viewDefaults, viewAddSource:
		return viewSources
	case viewConfirmDelete:
		return viewInstalled
	case viewHelp:
		if m.previousViewMode != "" {
			previous := m
			previous.viewMode = m.previousViewMode
			return previous.dashboardSection()
		}
		return viewInstalled
	default:
		return m.viewMode
	}
}

func dashboardSections() []string {
	return []string{viewInstalled, viewSkills, viewUsage, viewSources, viewTargets, viewUpdate}
}

func targetScopeLabel(target, scope string) string {
	label := target
	switch target {
	case "codex":
		label = "Codex"
	case "claude":
		label = "Claude"
	case "gemini":
		label = "Gemini"
	case "opencode":
		label = "OpenCode"
	}
	return strings.TrimSpace(label + " " + scope)
}

func installedOverviewStats(rows []InstalledSkill) installedStats {
	stats := installedStats{Installed: len(rows)}
	skills := map[string]bool{}
	projects := map[string]bool{}
	for _, row := range rows {
		skills[installedSkillKey(row)] = true
		if row.Managed == "yes" {
			stats.Managed++
		}
		if row.Scope == "project" && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			projects[row.ProjectPath] = true
		}
	}
	stats.Skills = len(skills)
	stats.Projects = len(projects)
	return stats
}

func buildUsageSummaries(rows []InstalledSkill) []UsageSummary {
	grouped := map[string][]InstalledSkill{}
	for _, row := range rows {
		key := installedSkillKey(row)
		grouped[key] = append(grouped[key], row)
	}
	summaries := make([]UsageSummary, 0, len(grouped))
	for key, group := range grouped {
		summaries = append(summaries, summarizeUsageRows(key, group))
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		if summaries[i].Source != summaries[j].Source {
			return summaries[i].Source < summaries[j].Source
		}
		return summaries[i].Skill < summaries[j].Skill
	})
	return summaries
}

func summarizeUsageRows(key string, rows []InstalledSkill) UsageSummary {
	summary := UsageSummary{Key: key, InstallCount: len(rows)}
	projects := map[string]bool{}
	targets := map[string]bool{}
	for _, row := range rows {
		if summary.Source == "" {
			summary.Source = row.Source
		}
		if summary.Skill == "" {
			summary.Skill = row.Skill
		}
		if row.Scope == "project" && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			projects[row.ProjectPath] = true
		}
		targets[row.Target+"/"+row.Scope] = true
		if strings.TrimSpace(row.UpdatedAt) != "" && row.UpdatedAt != "-" && row.UpdatedAt > summary.LatestUpdated {
			summary.LatestUpdated = row.UpdatedAt
		}
	}
	summary.ProjectCount = len(projects)
	summary.TargetCount = len(targets)
	if summary.LatestUpdated == "" {
		summary.LatestUpdated = "-"
	}
	return summary
}

func installedSkillKey(row InstalledSkill) string {
	if row.Managed == "yes" && strings.TrimSpace(row.Source) != "" && row.Source != "-" {
		return row.Source + "/" + row.Skill
	}
	return row.Skill
}

func installedSkillLabel(row InstalledSkill) string {
	return installedSkillKey(row)
}

func installedSection(row InstalledSkill) string {
	if row.Target == "directory" || row.Scope == "custom" {
		return "Custom directories"
	}
	if row.Scope == "project" {
		return "Projects"
	}
	return "Global"
}

func installedSubgroup(row InstalledSkill) string {
	switch installedSection(row) {
	case "Projects":
		if strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			return "Project " + row.ProjectPath
		}
		return "project path not recorded"
	case "Custom directories":
		if strings.TrimSpace(row.TargetRoot) != "" && row.TargetRoot != "-" {
			return row.TargetRoot
		}
		if strings.TrimSpace(row.Path) != "" && row.Path != "-" {
			return filepath.Dir(row.Path)
		}
	}
	return ""
}

func sectionTitle(section string) string {
	switch section {
	case viewUsage:
		return "Usage"
	case viewInstalled:
		return "Installed"
	case viewSources:
		return "Sources"
	case viewTargets:
		return "Targets"
	case viewUpdate:
		return "Update"
	default:
		return "Skills"
	}
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
		output, stderr, err := runScriptOutput(repoRoot, "scripts/skills.sh", "list", "--tsv")
		if err != nil {
			return skillsLoadedMsg{err: commandError(err, output+stderr)}
		}
		skills, err := parseSkillsTSV(output)
		return skillsLoadedMsg{skills: skills, warning: stderr, err: err}
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

func loadSources(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/sources.sh", "list", "--tsv")
		if err != nil {
			return sourcesLoadedMsg{err: commandError(err, output)}
		}
		sources, err := parseSourcesTSV(output)
		return sourcesLoadedMsg{sources: sources, err: err}
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

func loadInstalled(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		targetOutput, err := runScript(repoRoot, "scripts/targets.sh", "list", "--tsv")
		if err != nil {
			return installedLoadedMsg{err: commandError(err, targetOutput)}
		}
		targets, err := parseTargetsTSV(targetOutput)
		if err != nil {
			return installedLoadedMsg{err: err}
		}

		rows := []InstalledSkill{}
		for _, target := range sortedTargetsForDisplay(targets) {
			if target.Status != "supported" || target.Adapter != "skill-dir" || !isNativeScopedTarget(target.ID) {
				continue
			}
			for _, scope := range []string{"global", "project"} {
				args := installedListArgs(target.ID, scope, callerCwd())
				output, err := runScript(repoRoot, "scripts/installed.sh", args...)
				if err != nil {
					return installedLoadedMsg{err: commandError(err, output)}
				}
				parsed, err := parseInstalledTSV(output)
				if err != nil {
					return installedLoadedMsg{err: err}
				}
				rows = append(rows, parsed...)
			}
		}
		usageOutput, err := runScript(repoRoot, "scripts/installed.sh", "usage", "--tsv")
		if err != nil {
			return installedLoadedMsg{err: commandError(err, usageOutput)}
		}
		usageRows, err := parseInstalledUsageTSV(usageOutput)
		if err != nil {
			return installedLoadedMsg{err: err}
		}
		rows = mergeInstalledUsageRows(rows, usageRows)
		sortInstalledRows(rows)
		return installedLoadedMsg{rows: rows}
	}
}

func loadUsage(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/installed.sh", "usage", "--tsv")
		if err != nil {
			return usageLoadedMsg{err: commandError(err, output)}
		}
		rows, err := parseInstalledUsageTSV(output)
		if err != nil {
			return usageLoadedMsg{err: err}
		}
		sortInstalledRows(rows)
		return usageLoadedMsg{rows: rows}
	}
}

func runSkillCommand(repoRoot, action string, args ...string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/skills.sh", args...)
		return commandDoneMsg{action: action, output: output, err: commandError(err, output)}
	}
}

func runInstalledCommand(repoRoot, action string, args ...string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/installed.sh", args...)
		return commandDoneMsg{action: action, output: output, err: commandError(err, output)}
	}
}

func runSourceCommand(repoRoot, action string, args ...string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/sources.sh", args...)
		return commandDoneMsg{action: action, output: output, err: commandError(err, output)}
	}
}

func installedUpdateArgsForRow(row InstalledSkill, projectDir string) []string {
	if usesLegacyInstalledRow(row) {
		return []string{"update"}
	}
	if row.Target == "directory" || row.Scope == "custom" {
		dir := row.TargetRoot
		if strings.TrimSpace(dir) == "" && strings.TrimSpace(row.Path) != "" {
			dir = filepath.Dir(row.Path)
		}
		return []string{"update", "--target", "directory", "--dir", dir}
	}
	args := []string{"update", "--target", row.Target, "--scope", row.Scope}
	if row.Scope == "project" {
		args = append(args, "--project", installedRowProjectDir(row, projectDir))
	}
	return args
}

func installedUninstallArgsForRow(row InstalledSkill, projectDir string) []string {
	if usesLegacyInstalledRow(row) {
		return []string{"uninstall", row.Skill}
	}
	if row.Target == "directory" || row.Scope == "custom" {
		dir := row.TargetRoot
		if strings.TrimSpace(dir) == "" && strings.TrimSpace(row.Path) != "" {
			dir = filepath.Dir(row.Path)
		}
		return []string{"uninstall", row.Skill, "--target", "directory", "--dir", dir}
	}
	args := []string{"uninstall", row.Skill, "--target", row.Target, "--scope", row.Scope}
	if row.Scope == "project" {
		args = append(args, "--project", installedRowProjectDir(row, projectDir))
	}
	return args
}

func usageUpdateArgsForKey(key string) []string {
	return []string{"usage", "update", "--projects", key}
}

func usageUpdateArgsForLocation(row InstalledSkill) []string {
	args := []string{"usage", "update", "--projects"}
	if strings.TrimSpace(row.Target) != "" && row.Target != "-" {
		args = append(args, "--target", row.Target)
	}
	if isProjectUsageRow(row) {
		args = append(args, "--project", row.ProjectPath)
	}
	return append(args, installedSkillKey(row))
}

func usageBulkUpdateArgGroups(rows []InstalledSkill) [][]string {
	type groupKey struct {
		target  string
		project string
	}
	order := []groupKey{}
	grouped := map[groupKey]map[string]bool{}
	for _, row := range rows {
		if !isProjectUsageRow(row) {
			continue
		}
		key := groupKey{target: row.Target, project: row.ProjectPath}
		if _, ok := grouped[key]; !ok {
			grouped[key] = map[string]bool{}
			order = append(order, key)
		}
		grouped[key][installedSkillKey(row)] = true
	}

	groups := make([][]string, 0, len(order))
	for _, key := range order {
		skills := make([]string, 0, len(grouped[key]))
		for skill := range grouped[key] {
			skills = append(skills, skill)
		}
		sort.Strings(skills)
		args := []string{"usage", "update", "--projects", "--target", key.target, "--project", key.project}
		args = append(args, skills...)
		groups = append(groups, args)
	}
	return groups
}

func runUsageBulkUpdateCommand(repoRoot string, rows []InstalledSkill) tea.Cmd {
	return func() tea.Msg {
		groups := usageBulkUpdateArgGroups(rows)
		var combined strings.Builder
		for _, args := range groups {
			output, err := runScript(repoRoot, "scripts/installed.sh", args...)
			if strings.TrimSpace(output) != "" {
				combined.WriteString(output)
				if !strings.HasSuffix(output, "\n") {
					combined.WriteString("\n")
				}
			}
			if err != nil {
				return commandDoneMsg{action: "Update usage", output: combined.String(), err: commandError(err, output)}
			}
		}
		return commandDoneMsg{action: "Update usage", output: combined.String()}
	}
}

func installedRowProjectDir(row InstalledSkill, fallback string) string {
	if strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
		return row.ProjectPath
	}
	return fallback
}

func installedListArgs(target, scope, projectDir string) []string {
	if target == "codex" && scope == "global" && legacyCodexSkillsDir() != "" {
		return []string{"list", "--tsv"}
	}
	args := []string{"list", "--target", target, "--scope", scope, "--tsv"}
	if scope == "project" {
		args = append(args, "--project", projectDir)
	}
	return args
}

func usesLegacyInstalledRow(row InstalledSkill) bool {
	root := legacyCodexSkillsDir()
	if row.Target != "codex" || row.Scope != "global" || root == "" {
		return false
	}
	return row.Path == root || strings.HasPrefix(row.Path, root+string(os.PathSeparator))
}

func mergeInstalledUsageRows(rows []InstalledSkill, usageRows []InstalledSkill) []InstalledSkill {
	seen := map[string]bool{}
	merged := append([]InstalledSkill(nil), rows...)
	for i, row := range merged {
		seen[installedRowKey(row)] = true
		for _, usageRow := range usageRows {
			if installedRowKey(row) != installedRowKey(usageRow) {
				continue
			}
			merged[i].ProjectPath = usageRow.ProjectPath
			merged[i].TargetRoot = usageRow.TargetRoot
			merged[i].UpdatedAt = usageRow.UpdatedAt
			merged[i].RegistryOnly = true
			if strings.TrimSpace(merged[i].InstalledPath) == "" || merged[i].InstalledPath == "-" {
				merged[i].InstalledPath = usageRow.InstalledPath
			}
			if strings.TrimSpace(merged[i].ContentHash) == "" || merged[i].ContentHash == "-" {
				merged[i].ContentHash = usageRow.ContentHash
			}
			break
		}
	}
	for _, row := range usageRows {
		key := installedRowKey(row)
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, row)
	}
	return merged
}

func installedRowKey(row InstalledSkill) string {
	path := row.InstalledPath
	if strings.TrimSpace(path) == "" || path == "-" {
		path = row.Path
	}
	return row.Target + "\x00" + row.Scope + "\x00" + path
}

func sortInstalledRows(rows []InstalledSkill) {
	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]
		if installedSectionRank(left) != installedSectionRank(right) {
			return installedSectionRank(left) < installedSectionRank(right)
		}
		if installedSubgroup(left) != installedSubgroup(right) {
			return installedSubgroup(left) < installedSubgroup(right)
		}
		if targetDisplayRank(left.Target) != targetDisplayRank(right.Target) {
			return targetDisplayRank(left.Target) < targetDisplayRank(right.Target)
		}
		if scopeDisplayRank(left.Scope) != scopeDisplayRank(right.Scope) {
			return scopeDisplayRank(left.Scope) < scopeDisplayRank(right.Scope)
		}
		if left.ProjectPath != right.ProjectPath {
			return left.ProjectPath < right.ProjectPath
		}
		return left.Skill < right.Skill
	})
}

func installedSectionRank(row InstalledSkill) int {
	switch installedSection(row) {
	case "Global":
		return 0
	case "Projects":
		return 1
	case "Custom directories":
		return 2
	default:
		return 10
	}
}

func targetDisplayRank(target string) int {
	switch target {
	case "codex":
		return 0
	case "directory":
		return 1
	case "claude":
		return 2
	case "gemini":
		return 3
	case "opencode":
		return 4
	default:
		return 20
	}
}

func scopeDisplayRank(scope string) int {
	switch scope {
	case "global":
		return 0
	case "project":
		return 1
	case "custom":
		return 2
	default:
		return 10
	}
}

func buildInstallQueue(choices []InstallTargetChoice, names []string) []installQueueItem {
	items := make([]installQueueItem, 0, len(choices)*len(names))
	for _, name := range names {
		for _, choice := range choices {
			items = append(items, installQueueItem{Skill: name, Choice: choice})
		}
	}
	return items
}

func (p installProgressState) currentItem() (installQueueItem, bool) {
	index := p.Current - 1
	if index < 0 || index >= len(p.Items) {
		return installQueueItem{}, false
	}
	return p.Items[index], true
}

func (m model) installProgressStatus() string {
	item, ok := m.installProgress.currentItem()
	if !ok {
		return "Installing..."
	}
	return fmt.Sprintf("Installing %d/%d: %s -> %s.", m.installProgress.Current, m.installProgress.Total, item.Skill, item.Choice.Label)
}

func installProgressBar(completed, total, width int) string {
	if total <= 0 {
		total = 1
	}
	if width > 36 {
		width = 36
	}
	if width < 12 {
		width = 12
	}
	barWidth := width - 7
	if barWidth < 4 {
		barWidth = 4
	}
	if completed < 0 {
		completed = 0
	}
	if completed > total {
		completed = total
	}
	filled := completed * barWidth / total
	percent := completed * 100 / total
	return fmt.Sprintf("[%s%s] %3d%%", strings.Repeat("#", filled), strings.Repeat("-", barWidth-filled), percent)
}

func lastOutputLine(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return truncate(line, 180)
		}
	}
	return ""
}

func runInstallStepCommand(repoRoot string, item installQueueItem, projectDir string) tea.Cmd {
	return func() tea.Msg {
		args := installArgsForTargetChoice(item.Choice, projectDir, []string{item.Skill})
		output, err := runScript(repoRoot, "scripts/skills.sh", args...)
		if err != nil {
			return installStepDoneMsg{output: output, err: commandError(err, output)}
		}
		return installStepDoneMsg{output: output}
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

func normalizeBracketedInput(input string) string {
	value := strings.TrimSpace(input)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, ")") {
		if close := strings.Index(value, "]("); close > 0 {
			value = value[close+2 : len(value)-1]
			value = strings.TrimSpace(value)
		}
	}
	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]
		if (first == '[' && last == ']') || (first == '<' && last == '>') {
			value = strings.TrimSpace(value[1 : len(value)-1])
		}
	}
	return value
}

func normalizeSourceLocationInput(input string) string {
	return normalizeBracketedInput(input)
}

func normalizeSourceNameInput(input string) string {
	return normalizeBracketedInput(input)
}

func sourceAddArgs(locationInput, nameInput string) []string {
	args := []string{"add", normalizeSourceLocationInput(locationInput)}
	name := normalizeSourceNameInput(nameInput)
	if name != "" {
		args = append(args, "--name", name)
	}
	return args
}

func runScript(repoRoot, script string, args ...string) (string, error) {
	stdout, stderr, err := runScriptOutput(repoRoot, script, args...)
	return stdout + stderr, err
}

func runScriptOutput(repoRoot, script string, args ...string) (string, string, error) {
	cmd := exec.Command("sh", append([]string{filepath.Join(repoRoot, script)}, args...)...)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	if os.Getenv("SKILLHUB_CALLER_CWD") == "" {
		cmd.Env = append(cmd.Env, "SKILLHUB_CALLER_CWD="+callerCwd())
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
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

func parseSourcesTSV(input string) ([]SourcePreset, error) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] != "name\ttype\tlocation\tref\tcatalog" {
		return nil, fmt.Errorf("unexpected sources TSV header")
	}
	sources := make([]SourcePreset, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) != 5 {
			return nil, fmt.Errorf("invalid sources TSV row: %q", line)
		}
		sources = append(sources, SourcePreset{
			Name:     parts[0],
			Type:     parts[1],
			Location: parts[2],
			Ref:      parts[3],
			Catalog:  parts[4],
		})
	}
	return sources, nil
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

func parseInstalledTSV(input string) ([]InstalledSkill, error) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] != "target\tscope\tskill\tmanaged\tsource\tqualified_skill\tinstalled_path\tcontent_hash\tinstalled_at\tpath" {
		return nil, fmt.Errorf("unexpected installed TSV header")
	}
	rows := make([]InstalledSkill, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 10)
		if len(parts) != 10 {
			return nil, fmt.Errorf("invalid installed TSV row: %q", line)
		}
		rows = append(rows, InstalledSkill{
			Target:         parts[0],
			Scope:          parts[1],
			Skill:          parts[2],
			Managed:        parts[3],
			Source:         parts[4],
			QualifiedSkill: parts[5],
			InstalledPath:  parts[6],
			ContentHash:    parts[7],
			InstalledAt:    parts[8],
			Path:           parts[9],
		})
	}
	return rows, nil
}

func parseInstalledUsageTSV(input string) ([]InstalledSkill, error) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] != "source\tskill\ttarget\tscope\tproject_path\ttarget_root\tinstalled_path\tsource_ref\tsource_location\tcatalog\tcontent_hash\tinstalled_at\tupdated_at" {
		return nil, fmt.Errorf("unexpected installed usage TSV header")
	}
	rows := make([]InstalledSkill, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 13)
		if len(parts) != 13 {
			return nil, fmt.Errorf("invalid installed usage TSV row: %q", line)
		}
		rows = append(rows, InstalledSkill{
			Target:         parts[2],
			Scope:          parts[3],
			Skill:          parts[1],
			Managed:        "yes",
			Source:         parts[0],
			QualifiedSkill: parts[0] + "/" + parts[1],
			ProjectPath:    parts[4],
			TargetRoot:     parts[5],
			InstalledPath:  parts[6],
			ContentHash:    parts[10],
			InstalledAt:    parts[11],
			UpdatedAt:      parts[12],
			Path:           parts[6],
			RegistryOnly:   true,
		})
	}
	return rows, nil
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
