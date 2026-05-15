package tui

import tea "github.com/charmbracelet/bubbletea"

const (
	FlagDir     = flagDir
	FlagName    = flagName
	FlagScope   = flagScope
	FlagProject = flagProject
	FlagTarget  = flagTarget

	LabelCodex  = labelCodex
	LabelClaude = labelClaude
	LabelGemini = labelGemini

	ManagedYes = managedYes

	ScopeGlobal  = scopeGlobal
	ScopeProject = scopeProject
	ScopeCustom  = scopeCustom

	TargetCodex           = targetCodex
	TargetClaude          = targetClaude
	TargetDirectory       = targetDirectory
	TargetGemini          = targetGemini
	TargetPurposeInstall  = targetPurposeInstall
	TargetStatusSupported = targetStatusSupported

	ViewSkills           = viewSkills
	ViewInstalled        = viewInstalled
	ViewUsage            = viewUsage
	ViewUsageDetails     = viewUsageDetails
	ViewSources          = viewSources
	ViewUpdate           = viewUpdate
	ViewAddSource        = viewAddSource
	ViewDetails          = viewDetails
	ViewInstalledDetails = viewInstalledDetails
	ViewTargets          = viewTargets
	ViewInstallScope     = viewInstallScope
	ViewInstallConfirm   = viewInstallConfirm
	ViewInstallResult    = viewInstallResult
	ViewConfirmDelete    = viewConfirmDelete
	ViewHelp             = viewHelp
)

var (
	SearchBadgeStyle = searchBadgeStyle
	ActiveRowStyle   = activeRowStyle
	SelectedRowStyle = selectedRowStyle
)

type (
	TestInstallProgress = installProgressState
	TestSourceProgress  = sourceSyncProgressState
)

type TestModel struct {
	inner model

	RepoRoot string

	Skills         []Skill
	Filtered       []int
	Selected       map[string]bool
	Sources        []SourcePreset
	InstalledRows  []InstalledSkill
	UsageRows      []InstalledSkill
	UsageSummaries []UsageSummary
	Defaults       []SourcePreset
	Targets        []Target
	TargetStats    map[string]TargetDetection
	LockStatus     ProjectLockStatus

	Cursor                int
	InstalledCursor       int
	InstalledDetailCursor int
	UsageCursor           int
	UsageDetailCursor     int
	SourceCursor          int
	DefaultCursor         int
	TargetCursor          int
	Offset                int
	InstalledOffset       int
	InstalledDetailOffset int
	UsageOffset           int
	UsageDetailOffset     int
	SourceOffset          int
	TargetOffset          int
	Width                 int
	Height                int
	BodyHeight            int
	Search                string
	UsageFilter           string
	SourceInput           string
	SourceNameInput       string
	SourceField           int
	SearchMode            bool
	UsageFilterMode       bool
	Loading               bool
	LoadingSkills         bool
	LoadingInstalled      bool
	Busy                  bool
	NoSources             bool
	Status                string
	ViewMode              string
	PreviousViewMode      string
	InstalledDetailKey    string
	UsageDetailKey        string
	ReturnToUsageDetails  bool
	ReloadOnFinish        bool
	PostReloadStatus      string
	SpinnerFrame          int

	InstallScope       string
	InstallScopeCursor int
	ProjectDir         string
	TargetPurpose      string
	TargetChoices      []InstallTargetChoice
	SelectedTargets    map[string]bool
	PendingInstall     InstallResult
	InstallResult      InstallResult
	InstallProgress    TestInstallProgress
	SourceProgress     TestSourceProgress
	PendingUninstall   InstalledSkill
}

type SkillsLoadedMsg struct {
	Skills  []Skill
	Warning string
	Err     error
}

type DefaultsLoadedMsg struct {
	Defaults []SourcePreset
	Err      error
}

type SourcesLoadedMsg struct {
	Sources []SourcePreset
	Err     error
}

type InstalledLoadedMsg struct {
	Rows []InstalledSkill
	Err  error
}

type UsageLoadedMsg struct {
	Rows []InstalledSkill
	Err  error
}

type TargetsLoadedMsg struct {
	Targets    []Target
	Detections []TargetDetection
	Err        error
}

type LockStatusLoadedMsg struct {
	Status ProjectLockStatus
	Err    error
}

type CommandDoneMsg struct {
	Action string
	Output string
	Err    error
}

type SourceSyncStepDoneMsg struct {
	Output string
	Err    error
}

type InstallStepDoneMsg struct {
	Output string
	Err    error
}

func InitialModel(repoRoot string) TestModel {
	return testModelFromModel(initialModel(repoRoot))
}

func (m TestModel) Init() tea.Cmd {
	return m.innerModel().Init()
}

func (m TestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.innerModel().Update(toInnerMsg(msg))
	return testModelFromTea(updated), cmd
}

func (m TestModel) View() string {
	return m.innerModel().View()
}

func (m *TestModel) ApplyFilter() {
	inner := m.innerModel()
	inner.applyFilter()
	*m = testModelFromModel(inner)
}

func (m *TestModel) ApplyUsageFilter() {
	inner := m.innerModel()
	inner.applyUsageFilter()
	*m = testModelFromModel(inner)
}

func (m *TestModel) ToggleCurrent() {
	inner := m.innerModel()
	inner.toggleCurrent()
	*m = testModelFromModel(inner)
}

func (m *TestModel) ToggleCurrentTarget() {
	inner := m.innerModel()
	inner.toggleCurrentTarget()
	*m = testModelFromModel(inner)
}

func (m *TestModel) EnsureInstalledCursorVisible() {
	inner := m.innerModel()
	inner.ensureInstalledCursorVisible()
	*m = testModelFromModel(inner)
}

func (m TestModel) SelectedNames() []string {
	return m.innerModel().selectedNames()
}

func (m TestModel) CurrentInstalledDetail() (InstalledSkill, bool) {
	return m.innerModel().currentInstalledDetail()
}

func (m TestModel) InstallToSelectedTargets() (tea.Model, tea.Cmd) {
	updated, cmd := m.innerModel().installToSelectedTargets()
	return testModelFromTea(updated), cmd
}

func (m TestModel) BuildTargetChoices(targets []Target) []InstallTargetChoice {
	return m.innerModel().buildTargetChoices(targets)
}

func (m TestModel) VisibleCount() int {
	return m.innerModel().visibleCount()
}

func (m TestModel) InstallTargetPath() string {
	return m.innerModel().installTargetPath()
}

func (m TestModel) RenderHeader(width int) string {
	return m.innerModel().renderHeader(width)
}

func (m TestModel) AddSourceContent(width int) string {
	return m.innerModel().addSourceContent(width)
}

func ParseSkillsTSV(input string) ([]Skill, error) {
	return parseSkillsTSV(input)
}

func ParseInstalledTSV(input string) ([]InstalledSkill, error) {
	return parseInstalledTSV(input)
}

func ParseInstalledUsageTSV(input string) ([]InstalledSkill, error) {
	return parseInstalledUsageTSV(input)
}

func ParseActiveSourcesTSV(input string) ([]SourcePreset, error) {
	return parseSourcesTSV(input)
}

func ParseDefaultSourcesTSV(input string) ([]SourcePreset, error) {
	return parseDefaultSourcesTSV(input)
}

func ParseTargetsTSV(input string) ([]Target, error) {
	return parseTargetsTSV(input)
}

func ParseTargetDetectionsTSV(input string) ([]TargetDetection, error) {
	return parseTargetDetectionsTSV(input)
}

func BuildUsageSummaries(rows []InstalledSkill) []UsageSummary {
	return buildUsageSummaries(rows)
}

func MergeInstalledUsageRows(scanned []InstalledSkill, usageRows []InstalledSkill) []InstalledSkill {
	return mergeInstalledUsageRows(scanned, usageRows)
}

func UsageUpdateArgsForKey(key string) []string {
	return usageUpdateArgsForKey(key)
}

func UsageUpdateArgsForLocation(row InstalledSkill) []string {
	return usageUpdateArgsForLocation(row)
}

func UsageBulkUpdateArgGroups(rows []InstalledSkill) [][]string {
	return usageBulkUpdateArgGroups(rows)
}

func InstallArgsForTargetChoice(choice InstallTargetChoice, projectDir string, names []string) []string {
	return installArgsForTargetChoice(choice, projectDir, names)
}

func SortInstalledRows(rows []InstalledSkill) {
	sortInstalledRows(rows)
}

func ParseSourcesTSV(input string) ([]SourcePreset, error) {
	return parseSourcesTSV(input)
}

func InstalledUpdateArgsForRow(row InstalledSkill, projectDir string) []string {
	return installedUpdateArgsForRow(row, projectDir)
}

func InstalledUninstallArgsForRow(row InstalledSkill, projectDir string) []string {
	return installedUninstallArgsForRow(row, projectDir)
}

func SourceAddArgs(location string, name string) []string {
	return sourceAddArgs(location, name)
}

func LoadSkills(repoRoot string) tea.Cmd {
	cmd := loadSkills(repoRoot)
	return func() tea.Msg {
		return toExportMsg(cmd())
	}
}

func LoadTargets(repoRoot string) tea.Cmd {
	cmd := loadTargets(repoRoot)
	return func() tea.Msg {
		return toExportMsg(cmd())
	}
}

func testModelFromTea(tm tea.Model) TestModel {
	inner, ok := tm.(model)
	if !ok {
		panic("expected tui model from test facade")
	}
	return testModelFromModel(inner)
}

func testModelFromModel(m model) TestModel {
	return TestModel{
		inner: m,

		RepoRoot: m.repoRoot,

		Skills:         m.skills,
		Filtered:       m.filtered,
		Selected:       m.selected,
		Sources:        m.sources,
		InstalledRows:  m.installedRows,
		UsageRows:      m.usageRows,
		UsageSummaries: m.usageSummaries,
		Defaults:       m.defaults,
		Targets:        m.targets,
		TargetStats:    m.targetStats,
		LockStatus:     m.lockStatus,

		Cursor:                m.cursor,
		InstalledCursor:       m.installedCursor,
		InstalledDetailCursor: m.installedDetailCursor,
		UsageCursor:           m.usageCursor,
		UsageDetailCursor:     m.usageDetailCursor,
		SourceCursor:          m.sourceCursor,
		DefaultCursor:         m.defaultCursor,
		TargetCursor:          m.targetCursor,
		Offset:                m.offset,
		InstalledOffset:       m.installedOffset,
		InstalledDetailOffset: m.installedDetailOffset,
		UsageOffset:           m.usageOffset,
		UsageDetailOffset:     m.usageDetailOffset,
		SourceOffset:          m.sourceOffset,
		TargetOffset:          m.targetOffset,
		Width:                 m.width,
		Height:                m.height,
		BodyHeight:            m.bodyHeight,
		Search:                m.search,
		UsageFilter:           m.usageFilter,
		SourceInput:           m.sourceInput,
		SourceNameInput:       m.sourceNameInput,
		SourceField:           m.sourceField,
		SearchMode:            m.searchMode,
		UsageFilterMode:       m.usageFilterMode,
		Loading:               m.loading,
		LoadingSkills:         m.loadingSkills,
		LoadingInstalled:      m.loadingInstalled,
		Busy:                  m.busy,
		NoSources:             m.noSources,
		Status:                m.status,
		ViewMode:              m.viewMode,
		PreviousViewMode:      m.previousViewMode,
		InstalledDetailKey:    m.installedDetailKey,
		UsageDetailKey:        m.usageDetailKey,
		ReturnToUsageDetails:  m.returnToUsageDetails,
		ReloadOnFinish:        m.reloadOnFinish,
		PostReloadStatus:      m.postReloadStatus,
		SpinnerFrame:          m.spinnerFrame,

		InstallScope:       m.install.scope,
		InstallScopeCursor: m.install.scopeCursor,
		ProjectDir:         m.projectDir,
		TargetPurpose:      m.install.targetPurpose,
		TargetChoices:      m.install.targetChoices,
		SelectedTargets:    m.install.selectedTargets,
		PendingInstall:     m.install.pending,
		InstallResult:      m.install.result,
		InstallProgress:    m.install.progress,
		SourceProgress:     m.sourceProgress,
		PendingUninstall:   m.pendingUninstall,
	}
}

func (m TestModel) innerModel() model {
	inner := m.inner

	inner.repoRoot = m.RepoRoot

	inner.skills = m.Skills
	inner.filtered = m.Filtered
	inner.selected = m.Selected
	inner.sources = m.Sources
	inner.installedRows = m.InstalledRows
	inner.usageRows = m.UsageRows
	inner.usageSummaries = m.UsageSummaries
	inner.defaults = m.Defaults
	inner.targets = m.Targets
	inner.targetStats = m.TargetStats
	inner.lockStatus = m.LockStatus

	inner.cursor = m.Cursor
	inner.installedCursor = m.InstalledCursor
	inner.installedDetailCursor = m.InstalledDetailCursor
	inner.usageCursor = m.UsageCursor
	inner.usageDetailCursor = m.UsageDetailCursor
	inner.sourceCursor = m.SourceCursor
	inner.defaultCursor = m.DefaultCursor
	inner.targetCursor = m.TargetCursor
	inner.offset = m.Offset
	inner.installedOffset = m.InstalledOffset
	inner.installedDetailOffset = m.InstalledDetailOffset
	inner.usageOffset = m.UsageOffset
	inner.usageDetailOffset = m.UsageDetailOffset
	inner.sourceOffset = m.SourceOffset
	inner.targetOffset = m.TargetOffset
	inner.width = m.Width
	inner.height = m.Height
	inner.bodyHeight = m.BodyHeight
	inner.search = m.Search
	inner.usageFilter = m.UsageFilter
	inner.sourceInput = m.SourceInput
	inner.sourceNameInput = m.SourceNameInput
	inner.sourceField = m.SourceField
	inner.searchMode = m.SearchMode
	inner.usageFilterMode = m.UsageFilterMode
	inner.loading = m.Loading
	inner.loadingSkills = m.LoadingSkills
	inner.loadingInstalled = m.LoadingInstalled
	inner.busy = m.Busy
	inner.noSources = m.NoSources
	inner.status = m.Status
	inner.viewMode = m.ViewMode
	inner.previousViewMode = m.PreviousViewMode
	inner.installedDetailKey = m.InstalledDetailKey
	inner.usageDetailKey = m.UsageDetailKey
	inner.returnToUsageDetails = m.ReturnToUsageDetails
	inner.reloadOnFinish = m.ReloadOnFinish
	inner.postReloadStatus = m.PostReloadStatus
	inner.spinnerFrame = m.SpinnerFrame

	inner.install.scope = m.InstallScope
	inner.install.scopeCursor = m.InstallScopeCursor
	inner.projectDir = m.ProjectDir
	inner.install.targetPurpose = m.TargetPurpose
	inner.install.targetChoices = m.TargetChoices
	inner.install.selectedTargets = m.SelectedTargets
	inner.install.pending = m.PendingInstall
	inner.install.result = m.InstallResult
	inner.install.progress = m.InstallProgress
	inner.sourceProgress = m.SourceProgress
	inner.pendingUninstall = m.PendingUninstall

	return inner
}

func toInnerMsg(msg tea.Msg) tea.Msg {
	switch msg := msg.(type) {
	case SkillsLoadedMsg:
		return skillsLoadedMsg{skills: msg.Skills, warning: msg.Warning, err: msg.Err}
	case DefaultsLoadedMsg:
		return defaultsLoadedMsg{defaults: msg.Defaults, err: msg.Err}
	case SourcesLoadedMsg:
		return sourcesLoadedMsg{sources: msg.Sources, err: msg.Err}
	case InstalledLoadedMsg:
		return installedLoadedMsg{rows: msg.Rows, err: msg.Err}
	case UsageLoadedMsg:
		return usageLoadedMsg{rows: msg.Rows, err: msg.Err}
	case TargetsLoadedMsg:
		return targetsLoadedMsg{targets: msg.Targets, detections: msg.Detections, err: msg.Err}
	case LockStatusLoadedMsg:
		return lockStatusLoadedMsg{status: msg.Status, err: msg.Err}
	case CommandDoneMsg:
		return commandDoneMsg{action: msg.Action, output: msg.Output, err: msg.Err}
	case SourceSyncStepDoneMsg:
		return sourceSyncStepDoneMsg{output: msg.Output, err: msg.Err}
	case InstallStepDoneMsg:
		return installStepDoneMsg{output: msg.Output, err: msg.Err}
	default:
		return msg
	}
}

func toExportMsg(msg tea.Msg) tea.Msg {
	switch msg := msg.(type) {
	case skillsLoadedMsg:
		return SkillsLoadedMsg{Skills: msg.skills, Warning: msg.warning, Err: msg.err}
	case defaultsLoadedMsg:
		return DefaultsLoadedMsg{Defaults: msg.defaults, Err: msg.err}
	case sourcesLoadedMsg:
		return SourcesLoadedMsg{Sources: msg.sources, Err: msg.err}
	case installedLoadedMsg:
		return InstalledLoadedMsg{Rows: msg.rows, Err: msg.err}
	case usageLoadedMsg:
		return UsageLoadedMsg{Rows: msg.rows, Err: msg.err}
	case targetsLoadedMsg:
		return TargetsLoadedMsg{Targets: msg.targets, Detections: msg.detections, Err: msg.err}
	case lockStatusLoadedMsg:
		return LockStatusLoadedMsg{Status: msg.status, Err: msg.err}
	case commandDoneMsg:
		return CommandDoneMsg{Action: msg.action, Output: msg.output, Err: msg.err}
	case sourceSyncStepDoneMsg:
		return SourceSyncStepDoneMsg{Output: msg.output, Err: msg.err}
	case installStepDoneMsg:
		return InstallStepDoneMsg{Output: msg.output, Err: msg.err}
	default:
		return msg
	}
}
