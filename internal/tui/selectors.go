package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

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
		if left.Source != right.Source {
			return left.Source < right.Source
		}
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

func (m model) lastVisibleIndexBySourceCategory(start, end int) map[string]int {
	result := map[string]int{}
	for _, idx := range m.filtered[start:end] {
		skill := m.skills[idx]
		result[skill.Source+"\x00"+skill.Category] = idx
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

func (m model) panelBodyBudget() int {
	if m.bodyHeight > 0 {
		return m.bodyHeight - panelChromeHeight()
	}
	if m.height > 0 {
		return m.height - 14
	}
	return 0
}

func (m model) installedVisibleCount() int {
	if m.height <= 0 {
		return 6
	}
	budget := m.panelBodyBudget()
	if budget <= 0 {
		budget = m.height - 14
	}
	count := (budget - 8) / 4
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
	maxOffset := len(m.install.targetChoices) - visible
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
	if len(m.install.targetChoices) == 0 || m.targetCursor < 0 || m.targetCursor >= len(m.install.targetChoices) {
		return
	}
	choice := m.install.targetChoices[m.targetCursor]
	if !choice.Supported {
		m.status = choice.Label + " is not supported by the TUI installer yet."
		return
	}
	if m.install.selectedTargets == nil {
		m.install.selectedTargets = map[string]bool{}
	}
	if m.install.selectedTargets[choice.Key] {
		delete(m.install.selectedTargets, choice.Key)
		m.status = "Unselected " + choice.Label + "."
	} else {
		m.install.selectedTargets[choice.Key] = true
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
	return row.Scope == scopeProject && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-"
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
	case targetCodex:
		label = labelCodex
	case targetClaude:
		label = labelClaude
	case targetGemini:
		label = labelGemini
	case targetOpenCode:
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
		if row.Managed == managedYes {
			stats.Managed++
		}
		if row.Scope == scopeProject && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			projects[row.ProjectPath] = true
		}
	}
	stats.Skills = len(skills)
	stats.Projects = len(projects)
	return stats
}

func (m model) installedRangeLabel() string {
	total := len(m.installedRows)
	if total == 0 {
		return "Showing 0/0"
	}
	visible := m.installedVisibleCount()
	start := m.installedOffset + 1
	end := m.installedOffset + visible
	if end > total {
		end = total
	}
	return fmt.Sprintf("Showing %d-%d/%d", start, end, total)
}

func (m model) shouldFocusInstallResultRows() bool {
	return m.viewMode == viewInstallResult && len(m.install.result.SkillNames) > 0 && len(m.installedRows) > 0
}

func (m *model) focusInstallResultRows() {
	wanted := map[string]bool{}
	for _, name := range m.install.result.SkillNames {
		wanted[name] = true
	}
	for i, row := range m.installedRows {
		if installedRowMatchesAnyName(row, wanted) {
			m.installedCursor = i
			m.installedOffset = i
			return
		}
	}
}

func installedRowMatchesAnyName(row InstalledSkill, wanted map[string]bool) bool {
	if wanted[row.QualifiedSkill] {
		return true
	}
	if strings.TrimSpace(row.Source) != "" && row.Source != "-" && wanted[row.Source+"/"+row.Skill] {
		return true
	}
	return wanted[row.Skill]
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
		if row.Scope == scopeProject && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
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
	if row.Managed == managedYes && strings.TrimSpace(row.Source) != "" && row.Source != "-" {
		return row.Source + "/" + row.Skill
	}
	return row.Skill
}

func installedSkillLabel(row InstalledSkill) string {
	return installedSkillKey(row)
}

func installedSection(row InstalledSkill) string {
	if row.Target == targetDirectory || row.Scope == scopeCustom {
		return targetGroupCustomDirs
	}
	if row.Scope == scopeProject {
		return targetGroupProjects
	}
	return "Global"
}

func installedSubgroup(row InstalledSkill) string {
	switch installedSection(row) {
	case targetGroupProjects:
		if strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			return "Project " + row.ProjectPath
		}
		return "project path not recorded"
	case targetGroupCustomDirs:
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
		return labelTargets
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
