package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/assurrussa/skillhub/internal/core"
)

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
		backend, err := newBackend(repoRoot)
		if err != nil {
			return skillsLoadedMsg{err: err}
		}
		skills, warning, err := backend.ListSkills("")
		return skillsLoadedMsg{skills: tuiSkills(skills), warning: warning, err: err}
	}
}

func loadDefaultSources(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		backend, err := newBackend(repoRoot)
		if err != nil {
			return defaultsLoadedMsg{err: err}
		}
		defaults, err := backend.ListDefaultSources()
		return defaultsLoadedMsg{defaults: tuiSources(defaults), err: err}
	}
}

func loadSources(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		backend, err := newBackend(repoRoot)
		if err != nil {
			return sourcesLoadedMsg{err: err}
		}
		statuses, err := backend.ListSourceStatuses()
		return sourcesLoadedMsg{sources: tuiSourceStatuses(statuses), err: err}
	}
}

func loadTargets(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		backend, err := newBackend(repoRoot)
		if err != nil {
			return targetsLoadedMsg{err: err}
		}
		targets, err := backend.ListTargets()
		if err != nil {
			return targetsLoadedMsg{err: err}
		}
		detections, err := backend.DetectTargets(callerCwd())
		if err != nil {
			return targetsLoadedMsg{err: err}
		}
		return targetsLoadedMsg{targets: tuiTargets(targets), detections: tuiTargetDetections(detections)}
	}
}

func loadInstalled(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		backend, err := newBackend(repoRoot)
		if err != nil {
			return installedLoadedMsg{err: err}
		}
		targets, err := backend.ListTargets()
		if err != nil {
			return installedLoadedMsg{err: err}
		}

		rows := []InstalledSkill{}
		for _, target := range sortedTargetsForDisplay(tuiTargets(targets)) {
			if target.Status != targetStatusSupported || target.Adapter != targetAdapterSkillDir || !isNativeScopedTarget(target.ID) {
				continue
			}
			for _, scope := range []string{scopeGlobal, scopeProject} {
				parsed, err := backend.ListInstalled(core.InstalledListOptions{Target: target.ID, Scope: scope, Project: callerCwd()})
				if err != nil {
					return installedLoadedMsg{err: err}
				}
				rows = append(rows, tuiInstalled(parsed)...)
			}
		}
		usageRows, err := backend.ReadUsage("")
		if err != nil {
			return installedLoadedMsg{err: err}
		}
		usageTUIRows := tuiInstalled(usageRows)
		markMissingInstalledPaths(usageTUIRows)
		rows = mergeInstalledUsageRows(rows, usageTUIRows)
		sortInstalledRows(rows)
		return installedLoadedMsg{rows: rows}
	}
}

func loadProjectLockStatus(repoRoot, projectDir string) tea.Cmd {
	return func() tea.Msg {
		projectPath := strings.TrimSpace(projectDir)
		if projectPath == "" {
			projectPath = callerCwd()
		}
		if abs, err := filepath.Abs(projectPath); err == nil {
			projectPath = abs
		}
		lockPath := filepath.Join(projectPath, "skills.lock.toml")
		backend, err := newBackend(repoRoot)
		if err != nil {
			return lockStatusLoadedMsg{err: err}
		}
		summary, err := backend.Restore(core.RestoreOptions{Project: projectPath, Check: true})
		status := projectLockStatusFromRestoreRows(summary.Rows)
		status.Path = lockPath
		if _, err := os.Stat(lockPath); err == nil {
			status.Present = true
		}
		return lockStatusLoadedMsg{status: status, err: err}
	}
}

func runSkillCommand(repoRoot, action string, args ...string) tea.Cmd {
	return func() tea.Msg {
		output, err := runCoreSkillCommand(repoRoot, args...)
		return commandDoneMsg{action: action, output: output, err: err}
	}
}

func runInstalledCommand(repoRoot, action string, args ...string) tea.Cmd {
	return func() tea.Msg {
		output, err := runCoreInstalledCommand(repoRoot, args...)
		return commandDoneMsg{action: action, output: output, err: err}
	}
}

func runSourceCommand(repoRoot, action string, args ...string) tea.Cmd {
	return func() tea.Msg {
		output, err := runCoreSourceCommand(repoRoot, args...)
		return commandDoneMsg{action: action, output: output, err: err}
	}
}

func installedUpdateArgsForRow(row InstalledSkill, projectDir string) []string {
	if row.Target == targetDirectory || row.Scope == scopeCustom {
		dir := row.TargetRoot
		if strings.TrimSpace(dir) == "" && strings.TrimSpace(row.Path) != "" {
			dir = filepath.Dir(row.Path)
		}
		return []string{commandUpdate, flagTarget, targetDirectory, flagDir, dir}
	}
	args := []string{commandUpdate, flagTarget, row.Target, flagScope, row.Scope}
	if row.Scope == scopeProject {
		args = append(args, flagProject, installedRowProjectDir(row, projectDir))
	}
	return args
}

func installedUninstallArgsForRow(row InstalledSkill, projectDir string) []string {
	if row.Target == targetDirectory || row.Scope == scopeCustom {
		dir := row.TargetRoot
		if strings.TrimSpace(dir) == "" && strings.TrimSpace(row.Path) != "" {
			dir = filepath.Dir(row.Path)
		}
		return []string{commandUninstall, row.Skill, flagTarget, targetDirectory, flagDir, dir}
	}
	args := []string{commandUninstall, row.Skill, flagTarget, row.Target, flagScope, row.Scope}
	if row.Scope == scopeProject {
		args = append(args, flagProject, installedRowProjectDir(row, projectDir))
	}
	return args
}

func installedRowProjectDir(row InstalledSkill, fallback string) string {
	if strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
		return row.ProjectPath
	}
	return fallback
}

func mergeInstalledUsageRows(rows []InstalledSkill, usageRows []InstalledSkill) []InstalledSkill {
	merged := append([]InstalledSkill(nil), rows...)
	usageByKey := map[string]InstalledSkill{}
	for _, row := range usageRows {
		usageByKey[installedRowKey(row)] = row
	}
	seen := map[string]bool{}
	for i := range merged {
		key := installedRowKey(merged[i])
		seen[key] = true
		usageRow, ok := usageByKey[key]
		if !ok {
			continue
		}
		merged[i].ProjectPath = usageRow.ProjectPath
		merged[i].TargetRoot = usageRow.TargetRoot
		merged[i].UpdatedAt = usageRow.UpdatedAt
		merged[i].RegistryOnly = true
		merged[i].PathMissing = false
		if strings.TrimSpace(merged[i].InstalledPath) == "" || merged[i].InstalledPath == "-" {
			merged[i].InstalledPath = usageRow.InstalledPath
		}
		if strings.TrimSpace(merged[i].ContentHash) == "" || merged[i].ContentHash == "-" {
			merged[i].ContentHash = usageRow.ContentHash
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

func markMissingInstalledPaths(rows []InstalledSkill) {
	for i := range rows {
		rows[i].PathMissing = !installedSkillPathExists(rows[i])
	}
}

func installedSkillPathExists(row InstalledSkill) bool {
	path := row.InstalledPath
	if strings.TrimSpace(path) == "" || path == "-" {
		path = row.Path
	}
	if strings.TrimSpace(path) == "" || path == "-" {
		return false
	}
	if _, err := os.Stat(filepath.Join(path, "SKILL.md")); err != nil {
		return false
	}
	return true
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
	case targetCodex:
		return 0
	case targetDirectory:
		return 1
	case targetClaude:
		return 2
	case targetGemini:
		return 3
	case targetAntigravity:
		return 4
	case targetOpenCode:
		return 5
	default:
		return 20
	}
}

func scopeDisplayRank(scope string) int {
	switch scope {
	case scopeGlobal:
		return 0
	case scopeProject:
		return 1
	case scopeCustom:
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

func (p sourceSyncProgressState) currentItem() (sourceSyncQueueItem, bool) {
	index := p.Current - 1
	if index < 0 || index >= len(p.Items) {
		return sourceSyncQueueItem{}, false
	}
	return p.Items[index], true
}

func (m model) installProgressStatus() string {
	item, ok := m.install.progress.currentItem()
	if !ok {
		return "Installing..."
	}
	return fmt.Sprintf(
		"Installing %d/%d: %s -> %s.",
		m.install.progress.Current,
		m.install.progress.Total,
		item.Skill,
		item.Choice.Label,
	)
}

func (m model) sourceSyncProgressStatus() string {
	item, ok := m.sourceProgress.currentItem()
	if !ok {
		return "Updating sources..."
	}
	return fmt.Sprintf(
		"Updating %d/%d: %s.",
		m.sourceProgress.Current,
		m.sourceProgress.Total,
		item.Name,
	)
}

func (m model) spinnerView() string {
	frames := []string{"|", "/", "-", "\\"}
	return frames[m.spinnerFrame%len(frames)]
}

func busyTick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTickMsg(t)
	})
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
		backend, err := newBackend(repoRoot)
		if err != nil {
			return installStepDoneMsg{err: err}
		}
		opts := core.InstallOptions{
			Names:  []string{item.Skill},
			Target: item.Choice.Target,
			Scope:  item.Choice.Scope,
		}
		if item.Choice.Scope == scopeProject {
			opts.Project = projectDir
		}
		output, warning, err := backend.Install(opts)
		output = warning + output
		if err != nil {
			return installStepDoneMsg{output: output, err: err}
		}
		return installStepDoneMsg{output: output}
	}
}

func runSourceSyncStepCommand(repoRoot string, item sourceSyncQueueItem) tea.Cmd {
	return func() tea.Msg {
		backend, err := newBackend(repoRoot)
		if err != nil {
			return sourceSyncStepDoneMsg{err: err}
		}
		output, err := backend.SyncSources(item.Name)
		if err != nil {
			return sourceSyncStepDoneMsg{output: output, err: err}
		}
		return sourceSyncStepDoneMsg{output: output}
	}
}

func installArgsForTargetChoice(choice InstallTargetChoice, projectDir string, names []string) []string {
	args := []string{"install"}
	args = append(args, flagTarget, choice.Target, flagScope, choice.Scope)
	if choice.Scope == scopeProject {
		args = append(args, flagProject, projectDir)
	}
	return append(args, names...)
}

func normalizeBracketedInput(input string) string {
	value := strings.TrimSpace(input)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, ")") {
		if closeIndex := strings.Index(value, "]("); closeIndex > 0 {
			value = value[closeIndex+2 : len(value)-1]
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
	return strings.ToLower(normalizeBracketedInput(input))
}

func sourceAddArgs(locationInput, nameInput string) []string {
	args := []string{commandAdd, normalizeSourceLocationInput(locationInput)}
	name := normalizeSourceNameInput(nameInput)
	if name != "" {
		args = append(args, flagName, name)
	}
	return args
}

func newBackend(repoRoot string) (*core.Backend, error) {
	return core.New(core.Context{RepoRoot: repoRoot, CallerCWD: callerCwd()})
}

func tuiSkills(rows []core.Skill) []Skill {
	out := make([]Skill, 0, len(rows))
	for _, row := range rows {
		out = append(out, Skill{
			Source: row.Source, Name: row.Name, Category: row.Category,
			Triggers: row.Triggers, Description: row.Description,
		})
	}
	return out
}

func tuiSources(rows []core.Source) []SourcePreset {
	out := make([]SourcePreset, 0, len(rows))
	for _, row := range rows {
		out = append(out, SourcePreset{
			Name: row.Name, Type: row.Type, Location: row.Location,
			Ref: row.Ref, Catalog: row.Catalog,
		})
	}
	return out
}

func tuiSourceStatuses(rows []core.SourceStatus) []SourcePreset {
	out := make([]SourcePreset, 0, len(rows))
	for _, row := range rows {
		out = append(out, SourcePreset{
			Name: row.Name, Type: row.Type, Status: row.Status,
			LastSyncedAt: formatTUITime(row.LastSyncedAt), CachePath: row.CachePath,
			Location: row.Location, Ref: row.Ref, Catalog: row.Catalog, Message: row.Message,
		})
	}
	return out
}

func formatTUITime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}

func tuiTargets(rows []core.Target) []Target {
	out := make([]Target, 0, len(rows))
	for _, row := range rows {
		out = append(out, Target{
			ID: row.ID, Label: row.Label, Status: row.Status,
			Adapter: row.Adapter, Description: row.Description,
		})
	}
	return out
}

func tuiTargetDetections(rows []core.TargetDetection) []TargetDetection {
	out := make([]TargetDetection, 0, len(rows))
	for _, row := range rows {
		out = append(out, TargetDetection{
			Target: row.Target, Scope: row.Scope, Status: row.Status,
			Path: row.Path, Exists: row.Exists, Skills: row.Skills, Managed: row.Managed,
		})
	}
	return out
}

func tuiInstalled(rows []core.InstalledSkill) []InstalledSkill {
	out := make([]InstalledSkill, 0, len(rows))
	for _, row := range rows {
		out = append(out, InstalledSkill{
			Target: row.Target, Scope: row.Scope, Skill: row.Skill, Managed: row.Managed,
			Source: row.Source, QualifiedSkill: row.QualifiedSkill, ProjectPath: row.ProjectPath,
			TargetRoot: row.TargetRoot, InstalledPath: row.InstalledPath, ContentHash: row.ContentHash,
			InstalledAt: row.InstalledAt, UpdatedAt: row.UpdatedAt, Path: row.Path,
			RegistryOnly: row.RegistryOnly,
		})
	}
	return out
}

func projectLockStatusFromRestoreRows(rows []core.RestoreRow) ProjectLockStatus {
	status := ProjectLockStatus{Total: len(rows)}
	for _, row := range rows {
		switch row.Status {
		case core.ResultMissing:
			status.Missing++
		case "changed":
			status.Changed++
		case "skipped":
			status.Skipped++
		case core.ResultFailed:
			status.Failed++
		default:
			status.Unchanged++
		}
	}
	return status
}

func runCoreSkillCommand(repoRoot string, args ...string) (string, error) {
	backend, err := newBackend(repoRoot)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		return "", errors.New("missing skill command")
	}
	switch args[0] {
	case "restore":
		opts := core.RestoreOptions{}
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case flagProject:
				i++
				if i >= len(args) {
					return "", errors.New("--project requires a value")
				}
				opts.Project = args[i]
			case "--check":
				opts.Check = true
			case "-v", flagVerbose:
				opts.Verbose = true
			case flagTSV:
				opts.Check = true
			default:
				return "", fmt.Errorf("unsupported restore arg: %s", args[i])
			}
		}
		summary, err := backend.Restore(opts)
		var out strings.Builder
		if opts.Verbose {
			for _, row := range summary.Rows {
				_, _ = fmt.Fprintf(&out, "%s %s/%s %s: %s\n", row.Status, row.Source, row.Skill, row.Target, row.Reason)
			}
		}
		if opts.Check {
			_, _ = fmt.Fprintf(&out,
				"Project lockfile check: missing=%d changed=%d unchanged=%d skipped=%d failed=%d\n",
				summary.Installed, summary.Updated, summary.Unchanged, summary.Skipped, summary.Failed)
		} else {
			_, _ = fmt.Fprintf(&out,
				"Restored project skills: installed=%d updated=%d unchanged=%d skipped=%d failed=%d\n",
				summary.Installed, summary.Updated, summary.Unchanged, summary.Skipped, summary.Failed)
		}
		return out.String(), err
	default:
		return "", fmt.Errorf("unsupported skill command: %s", args[0])
	}
}

func runCoreSourceCommand(repoRoot string, args ...string) (string, error) {
	backend, err := newBackend(repoRoot)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		return "", errors.New("missing source command")
	}
	switch args[0] {
	case "sync":
		return runCoreSourceSync(backend, args)
	case commandAdd:
		return runCoreSourceAdd(backend, args)
	case "defaults":
		return runCoreSourceDefaults(backend, args)
	default:
		return "", fmt.Errorf("unsupported source command: %s", args[0])
	}
}

func runCoreSourceSync(backend *core.Backend, args []string) (string, error) {
	name := ""
	if len(args) > 1 {
		name = args[1]
	}
	return backend.SyncSources(name)
}

func runCoreSourceAdd(backend *core.Backend, args []string) (string, error) {
	if len(args) < 2 {
		return "", errors.New("source add requires a location")
	}
	opts := core.SourceAddOptions{Location: args[1]}
	for i := 2; i < len(args); i++ {
		flag := args[i]
		value, ok := nextFlagValue(args, &i)
		switch flag {
		case flagName:
			if !ok {
				return "", errors.New("--name requires a value")
			}
			opts.Name = value
		case "--type":
			if !ok {
				return "", errors.New("--type requires a value")
			}
			opts.Type = value
		case "--ref":
			if !ok {
				return "", errors.New("--ref requires a value")
			}
			opts.Ref = value
		case "--catalog":
			if !ok {
				return "", errors.New("--catalog requires a value")
			}
			opts.Catalog = value
		default:
			return "", fmt.Errorf("unsupported source add arg: %s", args[i])
		}
	}
	return backend.AddSource(opts)
}

func nextFlagValue(args []string, index *int) (string, bool) {
	(*index)++
	if *index >= len(args) {
		return "", false
	}
	return args[*index], true
}

func runCoreSourceDefaults(backend *core.Backend, args []string) (string, error) {
	if len(args) == 3 && args[1] == commandAdd {
		return backend.AddDefaultSource(args[2])
	}
	return "", errors.New("unsupported source defaults command")
}

func runCoreInstalledCommand(repoRoot string, args ...string) (string, error) {
	backend, err := newBackend(repoRoot)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		return "", errors.New("missing installed command")
	}
	switch args[0] {
	case commandUpdate:
		opts := core.InstalledUpdateOptions{}
		parseInstalledTargetArgs(args[1:], &opts.Target, &opts.Scope, &opts.Project, &opts.Dir, &opts.Verbose)
		summary, err := backend.UpdateInstalled(opts)
		return summary.Output, err
	case commandUninstall:
		if len(args) < 2 {
			return "", errors.New("uninstall requires a skill")
		}
		opts := core.UninstallOptions{Skill: args[1]}
		parseUninstallArgs(args[2:], &opts)
		return backend.Uninstall(opts)
	default:
		return "", fmt.Errorf("unsupported installed command: %s", args[0])
	}
}

func parseInstalledTargetArgs(args []string, target, scope, project, dir *string, verbose *bool) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case flagTarget:
			i++
			if i < len(args) {
				*target = args[i]
			}
		case flagScope:
			i++
			if i < len(args) {
				*scope = args[i]
			}
		case flagProject:
			i++
			if i < len(args) {
				*project = args[i]
			}
		case flagDir:
			i++
			if i < len(args) {
				*dir = args[i]
			}
		case "-v", flagVerbose:
			*verbose = true
		}
	}
}

func parseUninstallArgs(args []string, opts *core.UninstallOptions) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case flagTarget:
			i++
			if i < len(args) {
				opts.Target = args[i]
			}
		case flagScope:
			i++
			if i < len(args) {
				opts.Scope = args[i]
			}
		case flagProject:
			i++
			if i < len(args) {
				opts.Project = args[i]
			}
		case flagDir:
			i++
			if i < len(args) {
				opts.Dir = args[i]
			}
		case "--force":
			opts.Force = true
		}
	}
}

func parseSkillsTSV(input string) ([]Skill, error) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] != "source\tname\tcategory\ttriggers\tdescription" {
		return nil, errors.New("unexpected skills TSV header")
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
		return nil, errors.New("unexpected source defaults TSV header")
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
		return nil, errors.New("unexpected sources TSV header")
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
		return nil, errors.New("unexpected targets TSV header")
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
	if len(lines) == 0 || lines[0] != core.InstalledHeader {
		return nil, errors.New("unexpected installed TSV header")
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

func parseTargetDetectionsTSV(input string) ([]TargetDetection, error) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] != "target\tscope\tstatus\tpath\texists\tskills\tmanaged" {
		return nil, errors.New("unexpected target detections TSV header")
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
