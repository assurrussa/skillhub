package tui

type model struct {
	repoRoot string

	skills        []Skill
	filtered      []int
	selected      map[string]bool
	sources       []SourcePreset
	installedRows []InstalledSkill
	defaults      []SourcePreset
	targets       []Target
	targetStats   map[string]TargetDetection
	lockStatus    ProjectLockStatus

	cursor                int
	installedCursor       int
	installedDetailCursor int
	sourceCursor          int
	defaultCursor         int
	targetCursor          int
	offset                int
	installedOffset       int
	installedDetailOffset int
	sourceOffset          int
	defaultOffset         int
	targetOffset          int
	width                 int
	height                int
	bodyHeight            int
	search                string
	sourceInput           string
	sourceNameInput       string
	sourceField           int
	searchMode            bool
	loading               bool
	loadingSkills         bool
	loadingInstalled      bool
	busy                  bool
	noSources             bool
	status                string
	viewMode              string
	previousViewMode      string
	installedDetailKey    string
	reloadOnFinish        bool
	postReloadStatus      string
	spinnerFrame          int

	projectDir       string
	install          installFlowState
	sourceProgress   sourceSyncProgressState
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
