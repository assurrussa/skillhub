package tui

const (
	flagDir      = "--dir"
	flagScope    = "--scope"
	flagProjects = "--projects"
	flagTSV      = "--tsv"
	flagProject  = "--project"
	flagVerbose  = "--verbose"
	flagName     = "--name"
	flagTarget   = "--target"

	commandAdd       = "add"
	commandUninstall = "uninstall"
	commandUpdate    = "update"
	commandUsage     = "usage"

	actionUpdateUsage = "Update usage"

	labelCodex   = "Codex"
	labelClaude  = "Claude"
	labelGemini  = "Gemini"
	labelTargets = "Targets"

	statusLoadingInstalled = "Loading installed skills..."
	statusEnterSource      = "Enter a source path or git URL."
	statusSyncing          = "Syncing sources..."
	statusReturnedSkills   = "Returned to skills."
	statusUsageFilterClear = "Usage filter cleared."
	statusSelectTarget     = "Select at least one supported install target."

	keyEsc       = "esc"
	keyEnter     = "enter"
	keyBackspace = "backspace"
	keyDown      = "down"

	scopeGlobal           = "global"
	scopeProject          = "project"
	scopeCustom           = "custom"
	managedYes            = "yes"
	targetCodex           = "codex"
	targetClaude          = "claude"
	targetDirectory       = "directory"
	targetGemini          = "gemini"
	targetOpenCode        = "opencode"
	targetPurposeInstall  = "install"
	targetStatusSupported = "supported"
	targetAdapterSkillDir = "skill-dir"
	targetGroupCustomDirs = "Custom directories"
	targetGroupProjects   = "Projects"
	uiSelectedCursor      = "›"
)
