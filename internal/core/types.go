package core

import "time"

const (
	SourcesHeader        = "name\ttype\tlocation\tref\tcatalog"
	SourceStatusesHeader = "name\ttype\tstatus\tlast_synced_at\tcache_path\t" +
		"ref\tcatalog\tlocation\tmessage"
	TargetsHeader   = "id\tlabel\tstatus\tadapter\tdescription"
	SkillsHeader    = "source\tname\tcategory\ttriggers\tdescription"
	InstalledHeader = "target\tscope\tskill\tmanaged\tsource\tqualified_skill\tinstalled_path\tcontent_hash\t" +
		"installed_at\tpath"
	InstalledUsageHeader = "source\tskill\ttarget\tscope\tproject_path\ttarget_root\tinstalled_path\t" +
		"source_ref\tsource_location\tcatalog\tcontent_hash\tinstalled_at\tupdated_at"
	TargetDetectionsHeader = "target\tscope\tstatus\tpath\texists\tskills\tmanaged"
	RestoreCheckHeader     = "source\tskill\ttarget\tstatus\tinstalled_path\tcontent_hash\treason"
	RecommendHeader        = "source\tskill\treason\tinstall_arg"
)

const (
	TargetCodex     = "codex"
	TargetClaude    = "claude"
	TargetGemini    = "gemini"
	TargetDirectory = "directory"

	ScopeGlobal  = "global"
	ScopeProject = "project"
	ScopeCustom  = "custom"

	StatusSupported = "supported"
	StatusPlanned   = "planned"

	AdapterSkillDir = "skill-dir"

	SourceTypeGit  = "git"
	SourceTypePath = "path"

	SourceStatusLocal   = "local"
	SourceStatusFresh   = "fresh"
	SourceStatusStale   = "stale"
	SourceStatusMissing = "missing"
	SourceStatusError   = "error"

	ManagedYes = "yes"
	ManagedNo  = "no"

	ResultSkipped = "skipped"
	ResultFailed  = "failed"
	ResultMissing = "missing"
	ResultChanged = "changed"

	SourceNameAgentRules = "agent-rules"
)

type Context struct {
	RepoRoot  string
	CallerCWD string
	Env       map[string]string
	Now       func() time.Time
}

type Source struct {
	Name     string
	Type     string
	Location string
	Ref      string
	Catalog  string
}

type SourceStatus struct {
	Name         string
	Type         string
	Status       string
	LastSyncedAt time.Time
	CachePath    string
	Ref          string
	Catalog      string
	Location     string
	Message      string
}

type Skill struct {
	Source      string
	Name        string
	Category    string
	Triggers    string
	Description string
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
	SourceRef      string
	SourceLocation string
	Catalog        string
	ContentHash    string
	InstalledAt    string
	UpdatedAt      string
	Path           string
	RegistryOnly   bool
}

type RestoreRow struct {
	Source        string
	Skill         string
	Target        string
	Status        string
	InstalledPath string
	ContentHash   string
	Reason        string
}

type RestoreSummary struct {
	Rows      []RestoreRow
	Installed int
	Updated   int
	Unchanged int
	Skipped   int
	Failed    int
}

type UpdateSummary struct {
	Output    string
	Updated   int
	Unchanged int
	Skipped   int
	Failed    int
}

type RecommendRow struct {
	Source     string
	Skill      string
	Reason     string
	InstallArg string
	Score      int
}

type InstallResult struct {
	Messages []string
}

type SourceAddOptions struct {
	Location string
	Name     string
	Type     string
	Ref      string
	Catalog  string
}

type InstallOptions struct {
	All     bool
	Names   []string
	Target  string
	Scope   string
	Project string
	Dir     string
}

type InstalledListOptions struct {
	Target  string
	Scope   string
	Project string
	Dir     string
}

type InstalledUpdateOptions struct {
	Target       string
	Scope        string
	Project      string
	Dir          string
	Verbose      bool
	AllSupported bool
}

type UsageUpdateOptions struct {
	Projects bool
	Target   string
	Project  string
	Filters  []string
	Verbose  bool
}

type UninstallOptions struct {
	Skill   string
	Target  string
	Scope   string
	Project string
	Dir     string
	Force   bool
}

type RestoreOptions struct {
	Project string
	Check   bool
	Verbose bool
}

type RecommendOptions struct {
	Project string
}
