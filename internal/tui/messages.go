package tui

import "time"

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

type targetsLoadedMsg struct {
	targets    []Target
	detections []TargetDetection
	err        error
}

type lockStatusLoadedMsg struct {
	status ProjectLockStatus
	err    error
}

type commandDoneMsg struct {
	action string
	output string
	err    error
}

type sourceSyncStepDoneMsg struct {
	output string
	err    error
}

type installStepDoneMsg struct {
	output string
	err    error
}

type spinnerTickMsg time.Time

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

type sourceSyncQueueItem struct {
	Name string
}

type sourceSyncProgressState struct {
	Items     []sourceSyncQueueItem
	Current   int
	Completed int
	Total     int
	LastLine  string
	Failed    bool
	Error     string
}
