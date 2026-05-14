package tui

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
	lockStatus     ProjectLockStatus

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
	spinnerFrame          int

	projectDir       string
	install          installFlowState
	pendingUninstall InstalledSkill
}

type installFlowState struct {
	scope           string
	scopeCursor     int
	targetPurpose   string
	targetChoices   []InstallTargetChoice
	selectedTargets map[string]bool
	pending         InstallResult
	result          InstallResult
	progress        installProgressState
}
