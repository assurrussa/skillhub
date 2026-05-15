package tui

type Skill struct {
	Source      string
	Name        string
	Category    string
	Triggers    string
	Description string
}

type SourcePreset struct {
	Name         string
	Type         string
	Status       string
	LastSyncedAt string
	CachePath    string
	Location     string
	Ref          string
	Catalog      string
	Message      string
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
	PathMissing    bool
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

type ProjectLockStatus struct {
	Path      string
	Present   bool
	Total     int
	Missing   int
	Changed   int
	Unchanged int
	Skipped   int
	Failed    int
}

const (
	viewSkills           = "skills"
	viewInstalled        = "installed"
	viewSources          = "sources"
	viewUpdate           = "update"
	viewDefaults         = "defaults"
	viewAddSource        = "add-source"
	viewDetails          = "details"
	viewInstalledDetails = "installed-details"
	viewTargets          = "targets"
	viewInstallScope     = "install-scope"
	viewInstallConfirm   = "install-confirm"
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
