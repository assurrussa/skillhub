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

func (m model) lastVisibleIndexBySourceCategory(start, end int) map[string]int {
	result := map[string]int{}
	for _, idx := range m.filtered[start:end] {
		skill := m.skills[idx]
		result[skill.Source+"\x00"+skill.Category] = idx
	}
	return result
}

type cursorWindow struct {
	Cursor int
	Offset int
	Start  int
	End    int
}

func clampCursorWindow(cursor, offset, total, visible int) cursorWindow {
	if visible < 1 {
		visible = 1
	}
	cursor = clampCursor(cursor, total)
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+visible {
		offset = cursor - visible + 1
	}
	if offset < 0 {
		offset = 0
	}
	maxOffset := total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	end := offset + visible
	if end > total {
		end = total
	}
	return cursorWindow{Cursor: cursor, Offset: offset, Start: offset, End: end}
}

func ensureCursorWindow(cursor *int, offset *int, total int, visible int) {
	window := clampCursorWindow(*cursor, *offset, total, visible)
	*cursor = window.Cursor
	*offset = window.Offset
}

func (m *model) ensureCursorVisible() {
	ensureCursorWindow(&m.cursor, &m.offset, len(m.filtered), m.visibleCount())
}

func (m *model) ensureInstalledCursorVisible() {
	ensureCursorWindow(&m.installedCursor, &m.installedOffset, len(m.installedSummaries()), m.installedVisibleCount())
}

func (m *model) ensureInstalledDetailCursorVisible() {
	ensureCursorWindow(
		&m.installedDetailCursor,
		&m.installedDetailOffset,
		len(m.installedDetailRows()),
		m.installedDetailVisibleCount(),
	)
}

func (m *model) ensureSourceCursorVisible() {
	ensureCursorWindow(&m.sourceCursor, &m.sourceOffset, len(m.sources), m.sourceVisibleCount())
}

func (m *model) ensureDefaultCursorVisible() {
	ensureCursorWindow(&m.defaultCursor, &m.defaultOffset, len(m.defaults), m.defaultVisibleCount())
}

func (m model) sourceVisibleCount() int {
	return m.listVisibleCount(5, 4, 5)
}

func (m model) visibleCount() int {
	return m.listVisibleCount(5, 0, 5)
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

func (m model) listVisibleCount(rowHeight int, fixedLines int, fallback int) int {
	if m.height <= 0 {
		return fallback
	}
	if rowHeight < 1 {
		rowHeight = 1
	}
	budget := m.panelBodyBudget()
	if budget <= 0 {
		budget = m.height - 14
	}
	count := (budget - fixedLines) / rowHeight
	if count < 1 {
		return 1
	}
	return count
}

func (m model) installedVisibleCount() int {
	return m.listVisibleCount(4, 8, 6)
}

func (m model) installedDetailVisibleCount() int {
	return m.listVisibleCount(5, 4, 4)
}

func (m model) defaultVisibleCount() int {
	return m.listVisibleCount(1, 0, 10)
}

func (m *model) ensureTargetCursorVisible() {
	ensureCursorWindow(&m.targetCursor, &m.targetOffset, len(m.install.targetChoices), m.targetVisibleCount())
}

func (m model) targetVisibleCount() int {
	return m.listVisibleCount(5, 0, 5)
}

func (m *model) ensureAllCursorsVisible() {
	m.ensureCursorVisible()
	m.ensureInstalledCursorVisible()
	m.ensureInstalledDetailCursorVisible()
	m.ensureSourceCursorVisible()
	m.ensureDefaultCursorVisible()
	m.ensureTargetCursorVisible()
}

func (m *model) ensureActiveCursorVisible() {
	switch m.viewMode {
	case viewInstalled:
		m.ensureInstalledCursorVisible()
	case viewInstalledDetails:
		m.ensureInstalledDetailCursorVisible()
	case viewSources:
		m.ensureSourceCursorVisible()
	case viewDefaults:
		m.ensureDefaultCursorVisible()
	case viewTargets:
		m.ensureTargetCursorVisible()
	default:
		m.ensureCursorVisible()
	}
}

func (m model) currentSource() (SourcePreset, bool) {
	if len(m.sources) == 0 || m.sourceCursor < 0 || m.sourceCursor >= len(m.sources) {
		return SourcePreset{}, false
	}
	return m.sources[m.sourceCursor], true
}

func (m *model) toggleCurrent() {
	if len(m.filtered) == 0 {
		return
	}
	skill := m.skills[m.filtered[m.cursor]]
	key := skill.Key()
	if m.selected[key] {
		delete(m.selected, key)
		m.status = "Removed " + key + " from install queue."
	} else {
		m.selected[key] = true
		m.status = "Queued " + key + " for install."
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
	summaries := m.installedSummaries()
	if len(summaries) == 0 || m.installedCursor < 0 || m.installedCursor >= len(summaries) {
		return InstalledSkill{}, false
	}
	return summaries[m.installedCursor].Row, true
}

func (m model) currentInstalledSummary() (installedSummary, bool) {
	summaries := m.installedSummaries()
	if len(summaries) == 0 || m.installedCursor < 0 || m.installedCursor >= len(summaries) {
		return installedSummary{}, false
	}
	return summaries[m.installedCursor], true
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
	return []string{viewInstalled, viewSkills, viewSources, viewTargets, viewUpdate}
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

type installedSummary struct {
	Key           string
	Row           InstalledSkill
	Rows          []InstalledSkill
	ManagedCount  int
	RegistryCount int
	MissingCount  int
	ProjectCount  int
	TargetLabels  []string
}

func (m model) installedSummaries() []installedSummary {
	return buildInstalledSummaries(m.installedRows)
}

func buildInstalledSummaries(rows []InstalledSkill) []installedSummary {
	grouped := map[string][]InstalledSkill{}
	order := []string{}
	for _, row := range rows {
		key := installedSkillKey(row)
		if _, ok := grouped[key]; !ok {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], row)
	}
	summaries := make([]installedSummary, 0, len(grouped))
	for _, key := range order {
		group := grouped[key]
		sortInstalledRows(group)
		summaries = append(summaries, summarizeInstalledRows(key, group))
	}
	return summaries
}

func summarizeInstalledRows(key string, rows []InstalledSkill) installedSummary {
	summary := installedSummary{Key: key, Row: rows[0], Rows: append([]InstalledSkill(nil), rows...)}
	projects := map[string]bool{}
	labels := map[string]bool{}
	for _, row := range rows {
		if row.Managed == managedYes {
			summary.ManagedCount++
		}
		if row.RegistryOnly {
			summary.RegistryCount++
		}
		if row.PathMissing {
			summary.MissingCount++
		}
		if row.Scope == scopeProject && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			projects[row.ProjectPath] = true
		}
		label := targetScopeLabel(row.Target, row.Scope)
		if !labels[label] {
			labels[label] = true
			summary.TargetLabels = append(summary.TargetLabels, label)
		}
	}
	summary.ProjectCount = len(projects)
	return summary
}

func (m model) installedRangeLabel() string {
	total := len(m.installedSummaries())
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
	for i, summary := range m.installedSummaries() {
		if installedSummaryMatchesAnyName(summary, wanted) {
			m.installedCursor = i
			m.installedOffset = i
			return
		}
	}
}

func installedSummaryMatchesAnyName(summary installedSummary, wanted map[string]bool) bool {
	for _, row := range summary.Rows {
		if installedRowMatchesAnyName(row, wanted) {
			return true
		}
	}
	return false
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
