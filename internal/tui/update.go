package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorVisible()
		m.ensureInstalledCursorVisible()
		m.ensureUsageCursorVisible()
		m.ensureUsageDetailCursorVisible()
		m.ensureTargetCursorVisible()
		return m, nil
	case spinnerTickMsg:
		if !m.busy {
			return m, nil
		}
		m.spinnerFrame++
		return m, busyTick()
	case skillsLoadedMsg:
		return m.updateSkillsLoaded(msg)
	case defaultsLoadedMsg:
		return m.updateDefaultsLoaded(msg)
	case sourcesLoadedMsg:
		return m.updateSourcesLoaded(msg)
	case installedLoadedMsg:
		return m.updateInstalledLoaded(msg)
	case usageLoadedMsg:
		return m.updateUsageLoaded(msg)
	case targetsLoadedMsg:
		return m.updateTargetsLoaded(msg)
	case lockStatusLoadedMsg:
		return m.updateLockStatusLoaded(msg)
	case installStepDoneMsg:
		return m.updateInstallStepDone(msg)
	case commandDoneMsg:
		return m.updateCommandDone(msg)
	case tea.KeyMsg:
		return m.updateKey(msg)
	default:
		return m, nil
	}
}

func (m model) updateSkillsLoaded(msg skillsLoadedMsg) (tea.Model, tea.Cmd) {
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
}

func (m model) updateDefaultsLoaded(msg defaultsLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.viewMode = viewDefaults
	if msg.err != nil {
		m.status = "Load source defaults failed: " + msg.err.Error()
		return m, nil
	}
	m.defaults = msg.defaults
	m.defaultCursor = clampCursor(m.defaultCursor, len(m.defaults))
	m.status = fmt.Sprintf("Loaded %d source default(s).", len(m.defaults))
	return m, nil
}

func (m model) updateSourcesLoaded(msg sourcesLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.viewMode = viewSources
	if msg.err != nil {
		m.status = "Load sources failed: " + msg.err.Error()
		return m, nil
	}
	m.sources = msg.sources
	m.status = fmt.Sprintf("Loaded %d active source(s).", len(m.sources))
	return m, nil
}

func (m model) updateInstalledLoaded(msg installedLoadedMsg) (tea.Model, tea.Cmd) {
	m.loadingInstalled = false
	m.loading = m.currentViewLoading()
	showStatus := m.dashboardSection() == viewInstalled ||
		m.viewMode == viewInstallResult ||
		strings.TrimSpace(m.postReloadStatus) != ""
	if msg.err != nil {
		m.postReloadStatus = ""
		if showStatus {
			m.status = "Load installed skills failed: " + msg.err.Error()
		}
		return m, nil
	}
	m.installedRows = msg.rows
	if m.shouldFocusInstallResultRows() {
		m.focusInstallResultRows()
	}
	m.installedCursor = clampCursor(m.installedCursor, len(m.installedRows))
	m.ensureInstalledCursorVisible()
	if strings.TrimSpace(m.postReloadStatus) != "" {
		m.status = m.postReloadStatus
		m.postReloadStatus = ""
	} else if showStatus {
		m.status = fmt.Sprintf("Loaded %d installed skill row(s).", len(m.installedRows))
	}
	return m, nil
}

func (m model) updateUsageLoaded(msg usageLoadedMsg) (tea.Model, tea.Cmd) {
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
	m.usageCursor = clampCursor(m.usageCursor, len(m.usageSummaries))
	m.ensureUsageCursorVisible()
	m = m.restoreUsageDetails(returnToDetails, detailKey)
	m.applyPostReloadStatus(fmt.Sprintf("Loaded %d managed usage row(s).", len(m.usageRows)))
	return m, nil
}

func (m model) restoreUsageDetails(returnToDetails bool, detailKey string) model {
	if !returnToDetails || strings.TrimSpace(detailKey) == "" {
		m.viewMode = viewUsage
		return m
	}
	m.usageDetailKey = detailKey
	detailRows := m.usageDetailRows()
	if len(detailRows) == 0 {
		m.viewMode = viewUsage
		m.usageDetailKey = ""
		return m
	}
	m.usageDetailCursor = clampCursor(m.usageDetailCursor, len(detailRows))
	m.ensureUsageDetailCursorVisible()
	m.viewMode = viewUsageDetails
	return m
}

func (m *model) applyPostReloadStatus(defaultStatus string) {
	if strings.TrimSpace(m.postReloadStatus) != "" {
		m.status = m.postReloadStatus
		m.postReloadStatus = ""
		return
	}
	m.status = defaultStatus
}

func (m model) updateTargetsLoaded(msg targetsLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.viewMode = viewTargets
	if msg.err != nil {
		m.status = "Load install targets failed: " + msg.err.Error()
		return m, nil
	}
	m.targets = msg.targets
	m.targetStats = indexTargetDetections(msg.detections)
	if m.install.targetPurpose == targetPurposeInstall {
		m.install.targetChoices = m.buildInstallAgentChoices(msg.targets)
	} else {
		m.install.targetChoices = m.buildTargetChoices(msg.targets)
	}
	m.seedSelectedTargets()
	m.targetCursor = clampCursor(m.targetCursor, len(m.install.targetChoices))
	m.ensureTargetCursorVisible()
	if m.install.targetPurpose == targetPurposeInstall {
		m.status = fmt.Sprintf("Choose agents for %d selected skill(s).", len(m.selected))
	} else {
		m.status = "Loaded target paths."
	}
	return m, nil
}

func (m model) updateLockStatusLoaded(msg lockStatusLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.viewMode = viewUpdate
	if msg.err != nil {
		m.postReloadStatus = ""
		m.status = "Load project lockfile status failed: " + msg.err.Error()
		return m, nil
	}
	m.lockStatus = msg.status
	m.applyPostReloadStatus("Loaded project lockfile status.")
	return m, nil
}

func (m model) updateInstallStepDone(msg installStepDoneMsg) (tea.Model, tea.Cmd) {
	if strings.TrimSpace(msg.output) != "" {
		m.install.progress.LastLine = lastOutputLine(msg.output)
	}
	if msg.err != nil {
		m.busy = false
		m.install.progress.Failed = true
		m.install.progress.Error = msg.err.Error()
		m.status = fmt.Sprintf("Install failed at %d/%d.", m.install.progress.Current, m.install.progress.Total)
		return m, nil
	}
	m.install.progress.Completed++
	if m.install.progress.Completed >= m.install.progress.Total {
		return m.finishInstallProgress()
	}
	m.install.progress.Current = m.install.progress.Completed + 1
	item, _ := m.install.progress.currentItem()
	m.status = m.installProgressStatus()
	return m, tea.Batch(runInstallStepCommand(m.repoRoot, item, m.projectDir), busyTick())
}

func (m model) finishInstallProgress() (tea.Model, tea.Cmd) {
	m.busy = false
	m.install.result = m.install.pending
	m.install.pending = InstallResult{}
	m.install.progress = installProgressState{}
	m.viewMode = viewInstallResult
	m.postReloadStatus = installCompleteStatus(m.install.result)
	m.status = m.postReloadStatus
	m.loadingInstalled = true
	return m, loadInstalled(m.repoRoot)
}

func (m model) updateCommandDone(msg commandDoneMsg) (tea.Model, tea.Cmd) {
	m.busy = false
	if msg.err != nil {
		m.install.pending = InstallResult{}
		m.reloadOnFinish = false
		m.status = fmt.Sprintf("%s failed: %s", msg.action, msg.err)
		if strings.TrimSpace(msg.output) != "" {
			m.status += " | " + compactOutput(msg.output)
		}
		return m, nil
	}
	if msg.action == "Install" && len(m.install.pending.Targets) > 0 {
		return m.finishPendingInstall()
	}
	successStatus := msg.action + " complete."
	if strings.TrimSpace(msg.output) != "" {
		successStatus += " " + compactOutput(msg.output)
	}
	m.status = successStatus
	return m.afterCommandSuccess(msg.action, successStatus)
}

func (m model) finishPendingInstall() (tea.Model, tea.Cmd) {
	m.install.result = m.install.pending
	m.install.pending = InstallResult{}
	m.viewMode = viewInstallResult
	m.postReloadStatus = installCompleteStatus(m.install.result)
	m.status = m.postReloadStatus
	m.loadingInstalled = true
	return m, loadInstalled(m.repoRoot)
}

func installCompleteStatus(result InstallResult) string {
	return fmt.Sprintf("Installed %d skill(s) to %d target(s).", len(result.SkillNames), len(result.Targets))
}

func (m model) afterCommandSuccess(action, successStatus string) (tea.Model, tea.Cmd) {
	if action == "Sync" || m.reloadOnFinish {
		m.reloadOnFinish = false
		return m.reloadCurrentView("Reloading...")
	}
	if action == "Update installed" || action == "Uninstall" {
		m.loading = true
		m.loadingInstalled = true
		m.viewMode = viewInstalled
		m.postReloadStatus = successStatus
		m.status = "Reloading installed skills..."
		return m, loadInstalled(m.repoRoot)
	}
	if action == actionUpdateUsage {
		m.loading = true
		if !m.returnToUsageDetails {
			m.viewMode = viewUsage
		}
		m.postReloadStatus = successStatus
		m.status = "Reloading usage..."
		return m, loadUsage(m.repoRoot)
	}
	if action == "Restore project" {
		m.loading = true
		m.viewMode = viewUpdate
		m.postReloadStatus = successStatus
		m.status = "Reloading project lockfile..."
		return m, loadProjectLockStatus(m.repoRoot, m.projectDir)
	}
	return m, nil
}

func clampCursor(cursor, total int) int {
	if cursor >= total {
		cursor = total - 1
	}
	if cursor < 0 {
		return 0
	}
	return cursor
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if m.searchMode {
		return m.updateSearchKey(msg)
	}

	if m.usageFilterMode {
		return m.updateUsageFilterKey(msg)
	}

	if m.install.progress.Failed {
		return m.installFlow().updateFailedProgressKey(msg)
	}

	if m.loading || m.busy {
		return m.updateBusyKey(msg)
	}

	if m.viewMode == viewAddSource {
		return m.sourcesSection().updateAddKey(msg)
	}

	if m.viewMode == viewInstallScope {
		return m.installFlow().updateScopeKey(msg)
	}

	if m.viewMode == viewInstallConfirm {
		return m.installFlow().updateConfirmKey(msg)
	}

	if handled, next, cmd := m.updateGlobalKey(msg); handled {
		return next, cmd
	}

	if m.viewMode == viewHelp {
		return m.updateHelpKey(msg)
	}

	if m.viewMode == viewConfirmDelete {
		return m.updateConfirmDeleteKey(msg)
	}

	if m.viewMode == viewDetails {
		return m.skillsSection().updateDetailsKey(msg)
	}

	if m.viewMode == viewInstalledDetails {
		return m.installedSectionModel().updateDetailsKey(msg)
	}

	if m.viewMode == viewUsageDetails {
		return m.usageSection().updateDetailsKey(msg)
	}

	if m.viewMode == viewInstallResult {
		return m.installFlow().updateResultKey(msg)
	}

	if m.viewMode == viewTargets {
		if m.install.targetPurpose == targetPurposeInstall {
			return m.installFlow().updateTargetsKey(msg)
		}
		return m.targetsSection().updateKey(msg)
	}

	if m.viewMode == viewInstalled {
		return m.installedSectionModel().updateKey(msg)
	}

	if m.viewMode == viewUsage {
		return m.usageSection().updateKey(msg)
	}

	if m.viewMode == viewSources {
		return m.sourcesSection().updateKey(msg)
	}

	if m.viewMode == viewUpdate {
		return m.updateUpdateKey(msg)
	}

	if m.viewMode == viewDefaults {
		return m.sourcesSection().updateDefaultsKey(msg)
	}

	return m.skillsSection().updateKey(msg)
}

func (m model) updateSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, keyEnter:
		m.searchMode = false
		return m, nil
	case keyBackspace:
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

func (m model) updateUsageFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.usageFilterMode = false
		m.usageFilter = ""
		m.applyUsageFilter()
		m.status = statusUsageFilterClear
		return m, nil
	case keyEnter:
		m.usageFilterMode = false
		m.status = "Usage filter applied."
		return m, nil
	case keyBackspace:
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

func (m model) updateBusyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", keyEsc:
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m model) updateGlobalKey(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.previousViewMode = m.viewMode
		m.viewMode = viewHelp
		m.status = "Help."
		return true, m, nil
	case "left":
		if m.canMoveDashboardSection() {
			next, cmd := m.moveDashboardSection(-1)
			return true, next, cmd
		}
		return true, m, nil
	case "right":
		if m.canMoveDashboardSection() {
			next, cmd := m.moveDashboardSection(1)
			return true, next, cmd
		}
		return true, m, nil
	case "1":
		next, cmd := m.openDashboardSection(viewInstalled)
		return true, next, cmd
	case "2":
		next, cmd := m.openDashboardSection(viewSkills)
		return true, next, cmd
	case "3":
		next, cmd := m.openDashboardSection(viewUsage)
		return true, next, cmd
	case "4":
		next, cmd := m.openDashboardSection(viewSources)
		return true, next, cmd
	case "5":
		next, cmd := m.openDashboardSection(viewTargets)
		return true, next, cmd
	case "6":
		next, cmd := m.openDashboardSection(viewUpdate)
		return true, next, cmd
	default:
		return false, m, nil
	}
}

func (m model) updateSkillsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "/":
		m.searchMode = true
		return m, nil
	case keyEsc:
		m.search = ""
		m.applyFilter()
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
		return m, nil
	case keyDown, "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}
		return m, nil
	case " ":
		m.toggleCurrent()
		return m, nil
	case keyEnter:
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
		m.status = statusEnterSource
		return m, nil
	case "t":
		return m.openDashboardSection(viewTargets)
	case "r":
		return m.reloadCurrentView("Reloading catalog...")
	case "s":
		m.busy = true
		m.status = statusSyncing
		return m, tea.Batch(runSourceCommand(m.repoRoot, "Sync", "sync"), busyTick())
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
	return m.openInstallScope()
}

func (m model) openInstallScope() (tea.Model, tea.Cmd) {
	m.loading = false
	m.install.targetPurpose = targetPurposeInstall
	m.install.scope = scopeProject
	m.install.scopeCursor = 0
	m.targetCursor = 0
	m.targetOffset = 0
	m.install.targetChoices = nil
	m.install.selectedTargets = map[string]bool{}
	m.viewMode = viewInstallScope
	m.status = fmt.Sprintf("Choose install scope for %d selected skill(s).", len(m.selectedNames()))
	return m, nil
}

func (m model) openTargetSelection() (tea.Model, tea.Cmd) {
	m.loading = true
	m.install.targetPurpose = targetPurposeInstall
	m.status = "Loading install agents..."
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
		m.loading = true
		m.viewMode = viewUpdate
		m.status = "Loading project lockfile status..."
		return m, loadProjectLockStatus(m.repoRoot, m.projectDir)
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
		m.status = statusLoadingInstalled
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
		m.install.targetPurpose = "browse"
		m.viewMode = viewTargets
		m.status = "Loading targets..."
		return m, loadTargets(m.repoRoot)
	case viewUpdate:
		m.loading = true
		m.viewMode = viewUpdate
		m.status = "Loading project lockfile status..."
		return m, loadProjectLockStatus(m.repoRoot, m.projectDir)
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
		return m.install.targetPurpose != targetPurposeInstall
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
		m.status = statusSelectTarget
		return m, nil
	}
	queue := buildInstallQueue(choices, names)
	if len(queue) == 0 {
		m.status = statusSelectTarget
		return m, nil
	}
	m.busy = true
	m.install.pending = buildInstallResult(choices, names)
	m.install.progress = installProgressState{
		Items:   queue,
		Current: 1,
		Total:   len(queue),
	}
	item, _ := m.install.progress.currentItem()
	m.status = m.installProgressStatus()
	return m, tea.Batch(runInstallStepCommand(m.repoRoot, item, m.projectDir), busyTick())
}

func (m model) updateDetailsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case keyEsc, keyEnter:
		m.viewMode = viewSkills
		m.status = statusReturnedSkills
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

func (m model) updateHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case keyEsc, keyEnter, "?":
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
	case keyDown, "j":
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
		cmd := runInstalledCommand(m.repoRoot, "Update installed", installedUpdateArgsForRow(row, m.projectDir)...)
		return m, tea.Batch(cmd, busyTick())
	case "x":
		row, ok := m.currentInstalled()
		if !ok {
			m.status = "No installed skill selected."
			return m, nil
		}
		if row.Managed != managedYes {
			m.status = "Unmanaged skills are read-only in TUI. Use CLI --force if needed."
			return m, nil
		}
		m.pendingUninstall = row
		m.viewMode = viewConfirmDelete
		m.status = "Confirm uninstall."
		return m, nil
	case keyEnter:
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
	case keyEsc, "b":
		m.viewMode = viewInstalled
		m.status = "Returned to installed skills."
		return m, nil
	case "up", "k":
		if m.installedDetailCursor > 0 {
			m.installedDetailCursor--
			m.ensureInstalledDetailCursorVisible()
		}
		return m, nil
	case keyDown, "j":
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
		cmd := runInstalledCommand(m.repoRoot, "Update installed", installedUpdateArgsForRow(row, m.projectDir)...)
		return m, tea.Batch(cmd, busyTick())
	case "x":
		row, ok := m.currentInstalledDetail()
		if !ok {
			m.status = "No installed location selected."
			return m, nil
		}
		if row.Managed != managedYes {
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
	case keyEsc:
		if strings.TrimSpace(m.usageFilter) != "" {
			m.usageFilter = ""
			m.applyUsageFilter()
			m.status = statusUsageFilterClear
		}
		return m, nil
	case "c":
		if strings.TrimSpace(m.usageFilter) != "" {
			m.usageFilter = ""
			m.applyUsageFilter()
			m.status = statusUsageFilterClear
		}
		return m, nil
	case "up", "k":
		if m.usageCursor > 0 {
			m.usageCursor--
			m.ensureUsageCursorVisible()
		}
		return m, nil
	case keyDown, "j":
		if m.usageCursor < len(m.usageSummaries)-1 {
			m.usageCursor++
			m.ensureUsageCursorVisible()
		}
		return m, nil
	case keyEnter:
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
		return m, tea.Batch(runUsageBulkUpdateCommand(m.repoRoot, rows), busyTick())
	case "U":
		m.returnToUsageDetails = false
		rows := m.visibleProjectUsageRows()
		if len(rows) == 0 {
			m.status = "No visible recorded project installs to update."
			return m, nil
		}
		m.busy = true
		m.status = fmt.Sprintf("Updating %d visible project usage row(s)...", len(rows))
		return m, tea.Batch(runUsageBulkUpdateCommand(m.repoRoot, rows), busyTick())
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
	case keyEsc, "b":
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
	case keyDown, "j":
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
		return m, tea.Batch(runInstalledCommand(m.repoRoot, actionUpdateUsage, usageUpdateArgsForLocation(row)...), busyTick())
	case "U":
		rows := projectUsageRows(m.usageDetailRows())
		if len(rows) == 0 {
			m.status = m.usageDetailKey + " has no visible recorded project installs to update."
			return m, nil
		}
		m.busy = true
		m.returnToUsageDetails = true
		m.status = fmt.Sprintf("Updating %d visible project usage row(s)...", len(rows))
		return m, tea.Batch(runUsageBulkUpdateCommand(m.repoRoot, rows), busyTick())
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
	case keyEsc, "n":
		m.viewMode = viewInstalled
		m.status = "Uninstall cancelled."
		return m, nil
	case keyEnter, "y":
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
		cmd := runInstalledCommand(m.repoRoot, "Uninstall", installedUninstallArgsForRow(row, m.projectDir)...)
		return m, tea.Batch(cmd, busyTick())
	default:
		return m, nil
	}
}

func (m model) updateTargetsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case keyEsc, "b":
		m.viewMode = viewSkills
		m.status = statusReturnedSkills
		return m, nil
	case "up", "k":
		if m.targetCursor > 0 {
			m.targetCursor--
			m.ensureTargetCursorVisible()
		}
		return m, nil
	case keyDown, "j":
		if m.targetCursor < len(m.install.targetChoices)-1 {
			m.targetCursor++
			m.ensureTargetCursorVisible()
		}
		return m, nil
	case " ":
		m.toggleCurrentTarget()
		return m, nil
	case "a":
		if m.install.selectedTargets == nil {
			m.install.selectedTargets = map[string]bool{}
		}
		for _, choice := range m.install.targetChoices {
			if choice.Supported {
				m.install.selectedTargets[choice.Key] = true
			}
		}
		m.status = fmt.Sprintf("Selected %d supported target option(s).", len(m.selectedTargetChoices()))
		return m, nil
	case "c":
		m.install.selectedTargets = map[string]bool{}
		m.status = "Target selection cleared."
		return m, nil
	case keyEnter, "i":
		m.status = "Select skills in 2 Skills, then press i to install."
		return m, nil
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
		m.status = statusEnterSource
		return m, nil
	case "s":
		m.busy = true
		m.reloadOnFinish = true
		m.status = statusSyncing
		return m, tea.Batch(runSourceCommand(m.repoRoot, "Sync", "sync"), busyTick())
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
		m.status = statusLoadingInstalled
		return m, loadInstalled(m.repoRoot)
	case "r":
		m.busy = true
		m.status = "Restoring project skills from skills.lock.toml..."
		return m, tea.Batch(runSkillCommand(m.repoRoot, "Restore project", "restore", flagProject, m.projectDir, "-v"), busyTick())
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
	case keyEsc:
		m.viewMode = viewSkills
		m.status = statusReturnedSkills
		return m, nil
	case "tab", keyDown:
		m.sourceField = (m.sourceField + 1) % 2
		return m, nil
	case "shift+tab", "up":
		m.sourceField = (m.sourceField + 1) % 2
		return m, nil
	case keyEnter:
		location := normalizeSourceLocationInput(m.sourceInput)
		if location == "" {
			m.status = "Enter a source path or git URL before adding."
			return m, nil
		}
		args := sourceAddArgs(location, m.sourceNameInput)
		m.busy = true
		m.reloadOnFinish = true
		m.status = "Adding source " + location + "..."
		return m, tea.Batch(runSourceCommand(m.repoRoot, "Add source", args...), busyTick())
	case keyBackspace:
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
	case keyEsc:
		m.viewMode = viewSkills
		m.status = statusReturnedSkills
		return m, nil
	case "up", "k":
		if m.defaultCursor > 0 {
			m.defaultCursor--
		}
		return m, nil
	case keyDown, "j":
		if m.defaultCursor < len(m.defaults)-1 {
			m.defaultCursor++
		}
		return m, nil
	case keyEnter, "a":
		if len(m.defaults) == 0 {
			m.status = "No source defaults available."
			return m, nil
		}
		source := m.defaults[m.defaultCursor]
		m.busy = true
		m.reloadOnFinish = true
		m.status = "Adding source " + source.Name + "..."
		return m, tea.Batch(runSourceCommand(m.repoRoot, "Add source", "defaults", "add", source.Name), busyTick())
	case "n":
		m.sourceInput = ""
		m.sourceNameInput = ""
		m.sourceField = 0
		m.viewMode = viewAddSource
		m.status = statusEnterSource
		return m, nil
	default:
		return m, nil
	}
}
