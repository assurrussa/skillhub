package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/assurrussa/skillhub/internal/core"
)

func (m model) View() string {
	if m.width == 0 {
		m.width = 100
	}

	contentWidth := m.contentWidth()
	header := m.renderHeader(contentWidth)
	navigation := m.renderNavigation(contentWidth)
	status := statusStyle.Width(contentWidth).Render(m.status)
	help := helpStyle.Width(contentWidth).Render(m.helpText())

	bodyModel := m
	if m.height > 0 {
		innerHeight := m.height - appStyle.GetPaddingTop() - appStyle.GetPaddingBottom()
		if innerHeight < 1 {
			innerHeight = 1
		}
		reservedHeight := lipgloss.Height(header) + lipgloss.Height(navigation) + lipgloss.Height(status) + lipgloss.Height(help)
		bodyModel.bodyHeight = innerHeight - reservedHeight
		if bodyModel.bodyHeight < 0 {
			bodyModel.bodyHeight = 0
		}
	}
	bodyModel.ensureActiveCursorVisible()
	body := bodyModel.renderBody(contentWidth)

	parts := []string{
		header,
		navigation,
	}
	if bodyModel.bodyHeight != 0 || m.height <= 0 {
		parts = append(parts, body)
	}
	parts = append(parts, status, help)

	rendered := appStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
	if m.height > 0 && lipgloss.Height(rendered) > m.height {
		rendered = lipgloss.NewStyle().MaxHeight(m.height).Render(rendered)
	}
	return rendered
}

func (m model) renderHeader(_ int) string {
	search := subtleStyle.Render("none")
	if m.searchMode {
		search = searchBadgeStyle.Render(m.search + "_")
	} else if strings.TrimSpace(m.search) != "" {
		search = searchBadgeStyle.Render(m.search)
	}

	title := titleStyle.Render("Skillhub")
	summary := fmt.Sprintf(
		"Sources: %d   Skills: %d   Visible: %d   Queued: %d   Search: %s",
		m.sourceCount(),
		len(m.skills),
		len(m.filtered),
		len(m.selected),
		search,
	)
	target := fmt.Sprintf("Target: %s   Change: t", m.installTargetLabel())
	switch m.viewMode {
	case viewTargets:
		if m.install.targetPurpose == targetPurposeInstall {
			target = fmt.Sprintf("Install agents: %d selected   Project: %s", len(m.selectedTargetChoices()), m.projectDir)
		} else {
			target = "Targets   Project: " + m.projectDir
		}
	case viewInstallScope, viewInstallConfirm:
		target = fmt.Sprintf("Install wizard: %s scope   Project: %s", installScopeDisplay(m.install.scope), m.projectDir)
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, subtleStyle.Render(summary), subtleStyle.Render(target))
}

func (m model) renderNavigation(width int) string {
	items := []struct {
		key     string
		label   string
		section string
	}{
		{"1", "Installed", viewInstalled},
		{"2", "Skills", viewSkills},
		{"3", "Sources", viewSources},
		{"4", labelTargets, viewTargets},
		{"5", "Update", viewUpdate},
	}
	rendered := make([]string, 0, len(items))
	current := m.dashboardSection()
	for _, item := range items {
		label := item.key + " " + item.label
		if item.section == current {
			rendered = append(rendered, badgeStyle.Render(label))
		} else {
			rendered = append(rendered, subtleStyle.Render(label))
		}
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(rendered, "  "))
}

func (m model) renderBody(width int) string {
	if m.sourceProgress.Total > 0 && (m.busy || m.sourceProgress.Failed) {
		title := "Source update progress"
		if m.sourceProgress.Failed {
			title = "Source update failed"
		}
		return m.renderPanel(title, m.sourceSyncProgressContent(width-6), width)
	}
	if m.install.progress.Total > 0 && (m.busy || m.install.progress.Failed) {
		title := "Install progress"
		if m.install.progress.Failed {
			title = "Install failed"
		}
		return m.renderPanel(title, m.installFlow().progressContent(width-6), width)
	}
	if m.busy {
		return m.renderPanel("Working", m.busyContent(width-6), width)
	}
	if m.loading {
		return m.renderPanel(sectionTitle(m.dashboardSection()), statusStyle.Render(m.status), width)
	}
	if m.viewMode == viewHelp {
		return m.renderPanel("Help", m.helpContent(width-6), width)
	}
	if m.viewMode == viewAddSource {
		return m.renderPanel("Add source", m.sourcesSection().addContent(width-6), width)
	}
	if m.viewMode == viewConfirmDelete {
		return m.renderPanel("Confirm uninstall", m.confirmDeleteContent(width-6), width)
	}
	if m.viewMode == viewInstallScope {
		return m.renderPanel("Install scope", m.installFlow().scopeContent(width-6), width)
	}
	if m.viewMode == viewInstallConfirm {
		return m.renderPanel("Confirm install", m.installFlow().confirmContent(width-6), width)
	}
	if m.viewMode == viewDetails {
		return m.renderPanel("Skill details", m.skillsSection().detailsContent(width-6), width)
	}
	if m.viewMode == viewInstalledDetails {
		return m.renderPanel("Installed skill details", m.installedSectionModel().detailsContent(width-6), width)
	}
	if m.viewMode == viewInstallResult {
		return m.renderPanel("Install complete", m.installFlow().resultContent(width-6), width)
	}
	if m.viewMode == viewTargets {
		targets := m.targetsSection()
		return m.renderPanel(targets.panelTitle(), targets.content(width-6), width)
	}
	if m.viewMode == viewInstalled {
		return m.renderPanel("Installed skills", m.installedSectionModel().content(width-6), width)
	}
	if m.viewMode == viewSources {
		return m.renderPanel("Sources", m.sourcesSection().content(width-6), width)
	}
	if m.viewMode == viewUpdate {
		return m.renderPanel("Update", m.updateContent(width-6), width)
	}
	if m.viewMode == viewDefaults {
		return m.renderPanel("Source presets", m.sourcesSection().defaultsContent(width-6), width)
	}
	if m.noSources {
		return m.renderPanel("Sources", "No sources configured.\n\nPress d for presets or n for a custom source.", width)
	}
	if len(m.filtered) == 0 {
		return m.renderPanel("Skills", "No skills matched.", width)
	}

	return m.renderPanel("Skills", m.skillsSection().content(width-6), width)
}

func (m model) skillsContent(width int) string {
	var b strings.Builder

	visible := m.visibleCount()
	end := m.offset + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	lastInGroup := m.lastVisibleIndexBySourceCategory(m.offset, end)
	previousSource := ""
	previousCategory := ""
	for row, idx := range m.filtered[m.offset:end] {
		skill := m.skills[idx]
		previousSource, previousCategory = writeSkillGroupHeaders(
			&b,
			skill,
			row,
			previousSource,
			previousCategory,
		)
		card := m.skillListCard(skillListCardOptions{
			Skill:       skill,
			Index:       idx,
			Row:         row,
			Width:       width,
			LastInGroup: lastInGroup,
		})
		_, _ = fmt.Fprintln(&b, card)
		if row != end-m.offset-1 {
			_, _ = fmt.Fprintln(&b)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func writeSkillGroupHeaders(
	b *strings.Builder,
	skill Skill,
	row int,
	previousSource string,
	previousCategory string,
) (source string, category string) {
	source = previousSource
	category = previousCategory
	if skill.Source != source {
		if row != 0 {
			_, _ = fmt.Fprintln(b)
		}
		_, _ = fmt.Fprintln(b, categoryStyle.Render("Source "+emptyLabel(skill.Source, "-")))
		source = skill.Source
		category = ""
	}
	if skill.Category != category {
		if row != 0 && category != "" {
			_, _ = fmt.Fprintln(b)
		}
		_, _ = fmt.Fprintln(b, treeStyle.Render("  • "+skill.Category))
		category = skill.Category
	}
	return source, category
}

type skillListCardOptions struct {
	Skill       Skill
	Index       int
	Row         int
	Width       int
	LastInGroup map[string]int
}

func (m model) skillListCard(opts skillListCardOptions) string {
	title := m.skillListCardTitle(opts)
	description := indent(wrapText(opts.Skill.Description, opts.Width-8), "        ")
	cardLines := []string{title, description}
	if strings.TrimSpace(opts.Skill.Triggers) != "" {
		meta := "        triggers: " + truncate(opts.Skill.Triggers, max(16, opts.Width-18))
		cardLines = append(cardLines, subtleStyle.Render(meta))
	}
	card := strings.Join(cardLines, "\n")
	if m.selected[opts.Skill.Key()] {
		return selectedRowStyle.Render(card)
	}
	return card
}

func (m model) skillListCardTitle(opts skillListCardOptions) string {
	skill := opts.Skill
	cursor := " "
	if m.offset+opts.Row == m.cursor {
		cursor = uiSelectedCursor
	}
	branch := "├─"
	if opts.LastInGroup[skill.Source+"\x00"+skill.Category] == opts.Index {
		branch = "└─"
	}
	checkbox := checkboxStyle.Render("[ ]")
	if m.selected[skill.Key()] {
		checkbox = checkedStyle.Render("[+]")
	}
	title := fmt.Sprintf("%s %s %s %s", cursor, treeStyle.Render("  "+branch), checkbox, skill.Name)
	locations := m.installedLocationsForSkill(skill)
	if len(locations) > 0 {
		title += "   " + projectBadgeStyle.Render("installed: "+installedLocationsBadge(locations))
	}
	if m.offset+opts.Row == m.cursor {
		return activeRowStyle.Render(title)
	}
	return title
}

func (m model) detailsContent(width int) string {
	skill, ok := m.currentSkill()
	if !ok {
		return "No skill selected."
	}
	selected := "no"
	if m.selected[skill.Key()] {
		selected = managedYes
	}
	lines := []string{
		titleStyle.Render(skill.Key()),
		"",
		labelLine("Source", skill.Source),
		labelLine("Category", badgeStyle.Render(skill.Category)),
		labelLine("Triggers", skill.Triggers),
		labelLine("Install queue", selected),
		labelLine("Default path", filepath.Join(m.installTargetPath(), skill.Name)),
		labelLine("Installed", installedLocationsBadge(m.installedLocationsForSkill(skill))),
		"",
		wrapText(skill.Description, width),
	}
	if locations := m.installedLocationsForSkill(skill); len(locations) > 0 {
		lines = append(lines, "", titleStyle.Render("Installed locations"), m.installedLocationsContent(locations, width))
	}
	return strings.Join(lines, "\n")
}

func (m model) installedLocationsForSkill(skill Skill) []InstalledSkill {
	key := skill.Key()
	rows := []InstalledSkill{}
	for _, row := range m.installedRows {
		if row.Managed != managedYes {
			continue
		}
		if strings.TrimSpace(row.Source) == "" || row.Source == "-" {
			continue
		}
		if row.Source+"/"+row.Skill != key && row.QualifiedSkill != key {
			continue
		}
		if row.PathMissing {
			continue
		}
		rows = append(rows, row)
	}
	sortInstalledRows(rows)
	return rows
}

func installedLocationsBadge(rows []InstalledSkill) string {
	if len(rows) == 0 {
		return "none"
	}
	if len(rows) == 1 {
		return targetScopeLabel(rows[0].Target, rows[0].Scope)
	}
	return fmt.Sprintf("%d locations", len(rows))
}

func (m model) installedLocationsContent(rows []InstalledSkill, width int) string {
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			_, _ = fmt.Fprintln(&b)
		}
		metaParts := []string{"managed"}
		if row.RegistryOnly {
			metaParts = append(metaParts, "registry")
		}
		_, _ = fmt.Fprintln(&b, "- "+targetScopeLabel(row.Target, row.Scope))
		if row.Scope == scopeProject && strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
			_, _ = fmt.Fprintln(&b, "  project: "+truncate(row.ProjectPath, max(12, width-11)))
		}
		path := row.InstalledPath
		if strings.TrimSpace(path) == "" || path == "-" {
			path = row.Path
		}
		if strings.TrimSpace(path) != "" && path != "-" {
			_, _ = fmt.Fprintln(&b, "  path: "+truncate(path, max(12, width-8)))
		}
		_, _ = fmt.Fprintln(&b, "  source: "+emptyLabel(row.Source, "-")+"   "+strings.Join(metaParts, " "))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) targetsContent(width int) string {
	if len(m.install.targetChoices) == 0 {
		return "No install targets available."
	}

	var b strings.Builder
	visible := m.targetVisibleCount()
	end := m.targetOffset + visible
	if end > len(m.install.targetChoices) {
		end = len(m.install.targetChoices)
	}
	for row, choice := range m.install.targetChoices[m.targetOffset:end] {
		i := m.targetOffset + row
		if row > 0 {
			_, _ = fmt.Fprintln(&b)
		}

		cursor := " "
		if i == m.targetCursor {
			cursor = uiSelectedCursor
		}
		checkbox := checkboxStyle.Render("[ ]")
		if !choice.Supported {
			checkbox = disabledStyle.Render("[-]")
		} else if m.install.selectedTargets[choice.Key] {
			checkbox = checkedStyle.Render("[✓]")
		}

		title := fmt.Sprintf("%s %s %s", cursor, checkbox, choice.Label)
		if i == m.targetCursor {
			title = activeRowStyle.Render(title)
		}
		if choice.Supported && m.install.selectedTargets[choice.Key] {
			title = selectedRowStyle.Render(title)
		}
		if !choice.Supported {
			title = disabledStyle.Render(title)
		}

		meta := "    " + choice.Status
		if m.install.targetPurpose == targetPurposeInstall && strings.TrimSpace(choice.Scope) != "" {
			meta += "   scope: " + choice.Scope
		}
		if strings.TrimSpace(choice.Path) != "" {
			meta += " -> " + choice.Path
		}
		if choice.Supported {
			meta += fmt.Sprintf("   exists: %s   skills: %s   managed: %s",
				emptyLabel(choice.Exists, "-"),
				emptyLabel(choice.Skills, "0"),
				emptyLabel(choice.Managed, "0"),
			)
		}
		description := indent(wrapText(choice.Description, width-4), "    ")
		_, _ = fmt.Fprintln(&b, title)
		_, _ = fmt.Fprintln(&b, subtleStyle.Render(meta))
		if strings.TrimSpace(description) != "" {
			_, _ = fmt.Fprintln(&b, description)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) installedContent(width int) string {
	legend := helpStyle.Render(
		"[M] managed by Skillhub   [ ] unmanaged: read-only in TUI   enter: locations",
	)
	if len(m.installedRows) == 0 {
		return legend + "\n\nNo installed skills found.\n\nInstall skills from 2 Skills, or run skillhub installed list in CLI."
	}

	var b strings.Builder
	stats := installedOverviewStats(m.installedRows)
	_, _ = fmt.Fprintf(&b, "%s   %s   %s   %s\n",
		badgeStyle.Render(fmt.Sprintf("Installed: %d", stats.Installed)),
		badgeStyle.Render(fmt.Sprintf("Skills: %d", stats.Skills)),
		badgeStyle.Render(fmt.Sprintf("Projects: %d", stats.Projects)),
		badgeStyle.Render(fmt.Sprintf("Managed: %d", stats.Managed)),
	)
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintln(&b, subtleStyle.Render(m.installedRangeLabel()))
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintln(&b, legend)
	_, _ = fmt.Fprintln(&b)
	summaries := m.installedSummaries()
	visible := m.installedVisibleCount()
	end := m.installedOffset + visible
	if end > len(summaries) {
		end = len(summaries)
	}
	for row, summary := range summaries[m.installedOffset:end] {
		i := m.installedOffset + row
		cursor := " "
		if i == m.installedCursor {
			cursor = uiSelectedCursor
		}
		marker := checkboxStyle.Render("[ ]")
		managedLabel := "unmanaged"
		if summary.ManagedCount > 0 {
			marker = checkedStyle.Render("[M]")
			managedLabel = "managed"
		}
		locationLabel := "location"
		if len(summary.Rows) != 1 {
			locationLabel = "locations"
		}
		title := fmt.Sprintf(
			"%s %s %s    %d %s",
			cursor,
			marker,
			summary.Key,
			len(summary.Rows),
			locationLabel,
		)
		if i == m.installedCursor {
			title = activeRowStyle.Render(title)
		}
		if summary.ManagedCount > 0 {
			title = selectedRowStyle.Render(title)
		}
		metaParts := []string{managedLabel}
		if summary.ManagedCount > 0 {
			metaParts = append(metaParts, fmt.Sprintf("managed: %d", summary.ManagedCount))
		}
		if summary.RegistryCount > 0 {
			metaParts = append(metaParts, fmt.Sprintf("registry: %d", summary.RegistryCount))
		}
		if summary.ProjectCount > 0 {
			metaParts = append(metaParts, fmt.Sprintf("projects: %d", summary.ProjectCount))
		}
		if summary.MissingCount > 0 {
			metaParts = append(metaParts, fmt.Sprintf("missing: %d", summary.MissingCount))
		}
		source := emptyLabel(summary.Row.Source, "-")
		targets := truncate(strings.Join(summary.TargetLabels, ", "), max(12, width-17))
		_, _ = fmt.Fprintln(&b, title)
		_, _ = fmt.Fprintln(&b, subtleStyle.Render("    source: "+source+"   "+strings.Join(metaParts, "   ")))
		_, _ = fmt.Fprintln(&b, subtleStyle.Render("    targets: "+targets))
		if row != end-m.installedOffset-1 {
			_, _ = fmt.Fprintln(&b)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) installedDetailsContent(width int) string {
	rows := m.installedDetailRows()
	if len(rows) == 0 {
		return "No installed locations found."
	}

	var b strings.Builder
	_, _ = b.WriteString(installedDetailsHeader(rows))

	cursor := clampInstalledDetailCursor(m.installedDetailCursor, len(rows))
	start, end := m.installedDetailWindow(len(rows))
	for rowIndex, row := range rows[start:end] {
		i := start + rowIndex
		_, _ = b.WriteString(m.installedDetailRowContent(row, i, cursor, width))
		if rowIndex != end-start-1 {
			_, _ = fmt.Fprintln(&b)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func installedDetailsHeader(rows []InstalledSkill) string {
	var b strings.Builder
	_, _ = fmt.Fprintln(&b, titleStyle.Render(installedSkillLabel(rows[0])))
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintf(&b, "%s\n\n", badgeStyle.Render(fmt.Sprintf("Installed in: %d", len(rows))))
	return b.String()
}

func clampInstalledDetailCursor(cursor, rowCount int) int {
	if cursor >= rowCount {
		return rowCount - 1
	}
	if cursor < 0 {
		return 0
	}
	return cursor
}

func (m model) installedDetailWindow(rowCount int) (start, end int) {
	start = m.installedDetailOffset
	if start > rowCount {
		start = rowCount
	}
	end = start + m.installedDetailVisibleCount()
	if end > rowCount {
		end = rowCount
	}
	return start, end
}

func (m model) installedDetailRowContent(row InstalledSkill, index, cursorIndex, width int) string {
	var b strings.Builder
	_, _ = fmt.Fprintln(&b, m.installedDetailRowTitle(row, index, cursorIndex))
	if strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
		project := "Project root: " + truncate(row.ProjectPath, max(12, width-18))
		_, _ = fmt.Fprintln(&b, "    "+projectBadgeStyle.Render(project))
	}
	_, _ = fmt.Fprintln(&b, subtleStyle.Render("    path: "+truncate(row.Path, max(12, width-10))))
	meta := indent(wrapText(strings.Join(installedDetailMetaParts(row), "   "), max(12, width-4)), "    ")
	_, _ = fmt.Fprint(&b, subtleStyle.Render(meta))
	return b.String()
}

func (m model) installedDetailRowTitle(row InstalledSkill, index, cursorIndex int) string {
	cursor := " "
	if index == cursorIndex {
		cursor = uiSelectedCursor
	}
	marker := checkboxStyle.Render("[ ]")
	if row.Managed == managedYes {
		marker = checkedStyle.Render("[M]")
	}
	title := fmt.Sprintf("%s %s %s", cursor, marker, targetScopeLabel(row.Target, row.Scope))
	if strings.TrimSpace(row.ProjectPath) != "" && row.ProjectPath != "-" {
		title = fmt.Sprintf(
			"%s %s %s %s",
			cursor,
			marker,
			projectBadgeStyle.Render("LOCAL PROJECT"),
			targetScopeLabel(row.Target, row.Scope),
		)
	}
	if index == cursorIndex {
		title = activeRowStyle.Render(title)
	}
	if row.Managed == managedYes {
		title = selectedRowStyle.Render(title)
	}
	return title
}

func installedDetailMetaParts(row InstalledSkill) []string {
	metaParts := []string{"source: " + emptyLabel(row.Source, "-")}
	if strings.TrimSpace(row.ContentHash) != "" && row.ContentHash != "-" {
		metaParts = append(metaParts, "hash: "+row.ContentHash)
	}
	if strings.TrimSpace(row.InstalledAt) != "" && row.InstalledAt != "-" {
		metaParts = append(metaParts, "installed: "+row.InstalledAt)
	}
	if strings.TrimSpace(row.UpdatedAt) != "" && row.UpdatedAt != "-" {
		metaParts = append(metaParts, "updated: "+row.UpdatedAt)
	}
	if row.PathMissing {
		metaParts = append(metaParts, core.ResultMissing)
	}
	return metaParts
}

func (m model) installProgressContent(width int) string {
	item, ok := m.install.progress.currentItem()
	if !ok {
		return "No install step is running."
	}

	title := fmt.Sprintf("Installing %d/%d", m.install.progress.Current, m.install.progress.Total)

	var b strings.Builder
	_, _ = fmt.Fprintln(&b, titleStyle.Render(title))
	_, _ = fmt.Fprintln(&b)
	if m.install.progress.Failed {
		_, _ = fmt.Fprintln(&b, "Stopped")
	} else {
		_, _ = fmt.Fprintf(&b, "Running %s\n", m.spinnerView())
	}
	_, _ = fmt.Fprintf(&b, "Skill %s\n", item.Skill)
	_, _ = fmt.Fprintf(&b, "Target %s\n", item.Choice.Label)
	if strings.TrimSpace(item.Choice.Path) != "" {
		_, _ = fmt.Fprintf(&b, "Path %s\n", truncate(item.Choice.Path, max(12, width-5)))
	}
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintln(&b, installProgressBar(m.install.progress.Completed, m.install.progress.Total, max(12, width-8)))
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintln(&b, "Queue")
	for i, queueItem := range m.install.progress.Items {
		state := queueStatePending
		switch {
		case i < m.install.progress.Completed:
			state = queueStateDone
		case i == m.install.progress.Current-1 && m.install.progress.Failed:
			state = queueStateFailed
		case i == m.install.progress.Current-1:
			state = queueStateRunning + " " + m.spinnerView()
		}
		line := fmt.Sprintf("  %2d. %-10s %s -> %s", i+1, state, queueItem.Skill, queueItem.Choice.Label)
		if i == m.install.progress.Current-1 {
			line = activeRowStyle.Render(line)
		}
		_, _ = fmt.Fprintln(&b, truncate(line, max(12, width)))
	}
	if strings.TrimSpace(m.install.progress.LastLine) != "" {
		_, _ = fmt.Fprintln(&b)
		_, _ = fmt.Fprintln(&b, "Last result")
		_, _ = fmt.Fprintln(&b, indent(wrapText(m.install.progress.LastLine, max(12, width-2)), "  "))
	}
	if strings.TrimSpace(m.install.progress.Error) != "" {
		_, _ = fmt.Fprintln(&b)
		_, _ = fmt.Fprintln(&b, "Error")
		_, _ = fmt.Fprintln(&b, indent(wrapText(m.install.progress.Error, max(12, width-2)), "  "))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) sourceSyncProgressContent(width int) string {
	item, ok := m.sourceProgress.currentItem()
	if !ok {
		return "No source update step is running."
	}

	title := fmt.Sprintf("Updating %d/%d", m.sourceProgress.Current, m.sourceProgress.Total)

	var b strings.Builder
	_, _ = fmt.Fprintln(&b, titleStyle.Render(title))
	_, _ = fmt.Fprintln(&b)
	if m.sourceProgress.Failed {
		_, _ = fmt.Fprintln(&b, "Stopped")
	} else {
		_, _ = fmt.Fprintf(&b, "Running %s\n", m.spinnerView())
	}
	_, _ = fmt.Fprintf(&b, "Source %s\n", item.Name)
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintln(&b, installProgressBar(m.sourceProgress.Completed, m.sourceProgress.Total, max(12, width-8)))
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintln(&b, "Queue")
	for i, queueItem := range m.sourceProgress.Items {
		state := queueStatePending
		switch {
		case i < m.sourceProgress.Completed:
			state = queueStateDone
		case i == m.sourceProgress.Current-1 && m.sourceProgress.Failed:
			state = queueStateFailed
		case i == m.sourceProgress.Current-1:
			state = queueStateRunning + " " + m.spinnerView()
		}
		line := fmt.Sprintf("  %2d. %-10s %s", i+1, state, queueItem.Name)
		if i == m.sourceProgress.Current-1 {
			line = activeRowStyle.Render(line)
		}
		_, _ = fmt.Fprintln(&b, truncate(line, max(12, width)))
	}
	if strings.TrimSpace(m.sourceProgress.LastLine) != "" {
		_, _ = fmt.Fprintln(&b)
		_, _ = fmt.Fprintln(&b, "Last result")
		_, _ = fmt.Fprintln(&b, indent(wrapText(m.sourceProgress.LastLine, max(12, width-2)), "  "))
	}
	if strings.TrimSpace(m.sourceProgress.Error) != "" {
		_, _ = fmt.Fprintln(&b)
		_, _ = fmt.Fprintln(&b, "Error")
		_, _ = fmt.Fprintln(&b, indent(wrapText(m.sourceProgress.Error, max(12, width-2)), "  "))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) busyContent(width int) string {
	status := strings.TrimSpace(m.status)
	if status == "" {
		status = "Working..."
	}
	lines := []string{
		"Running " + m.spinnerView(),
		"",
		wrapText(status, width),
	}
	return strings.Join(lines, "\n")
}

func (m model) installResultContent(_ int) string {
	if len(m.install.result.Targets) == 0 {
		return "No install result available."
	}

	var b strings.Builder
	_, _ = fmt.Fprintln(&b, titleStyle.Render("Installed successfully"))
	_, _ = fmt.Fprintln(&b)
	for targetIndex, target := range m.install.result.Targets {
		if targetIndex > 0 {
			_, _ = fmt.Fprintln(&b)
		}
		_, _ = fmt.Fprintf(&b, "%s %s\n", checkedStyle.Render("[✓]"), target.Label)
		meta := fmt.Sprintf("    %s/%s -> %s", target.Target, target.Scope, target.Root)
		_, _ = fmt.Fprintln(&b, subtleStyle.Render(meta))
		for _, path := range target.SkillPaths {
			_, _ = fmt.Fprintf(&b, "    %s\n", path)
		}
	}
	_, _ = fmt.Fprintln(&b)
	actions := strings.Join([]string{
		badgeStyle.Render("enter back"),
		badgeStyle.Render("t targets"),
		badgeStyle.Render("q quit"),
	}, "  ")
	_, _ = fmt.Fprintln(&b, actions)
	return strings.TrimRight(b.String(), "\n")
}

func (m model) sourcesContent(width int) string {
	if len(m.sources) == 0 {
		return "No sources configured.\n\nPress d for recommended presets or n for a custom source."
	}
	var b strings.Builder
	counts := sourceStatusCounts(m.sources)
	_, _ = fmt.Fprintf(
		&b,
		"Sources: %d   local: %d   fresh: %d   stale: %d   missing: %d   error: %d\n",
		len(m.sources),
		counts[core.SourceStatusLocal],
		counts[core.SourceStatusFresh],
		counts[core.SourceStatusStale],
		counts[core.SourceStatusMissing],
		counts[core.SourceStatusError],
	)
	_, _ = fmt.Fprintln(&b)
	visible := m.sourceVisibleCount()
	end := m.sourceOffset + visible
	if end > len(m.sources) {
		end = len(m.sources)
	}
	for row, source := range m.sources[m.sourceOffset:end] {
		i := m.sourceOffset + row
		if row > 0 {
			_, _ = fmt.Fprintln(&b)
		}
		name := source.Name
		if source.Status != "" {
			name += " [" + source.Status + "]"
		}
		cursor := " "
		if i == m.sourceCursor {
			cursor = uiSelectedCursor
		}
		nameLine := fmt.Sprintf("%s %s", cursor, name)
		if i == m.sourceCursor {
			nameLine = activeRowStyle.Width(width).Render(nameLine)
		} else {
			nameLine = titleStyle.Render(nameLine)
		}
		_, _ = fmt.Fprintln(&b, nameLine)
		lastSync := emptyLabel(source.LastSyncedAt, "-")
		_, _ = fmt.Fprintln(&b, subtleStyle.Render(fmt.Sprintf(
			"    %s %s   last sync: %s   catalog: %s",
			source.Type,
			source.Ref,
			lastSync,
			source.Catalog,
		)))
		_, _ = fmt.Fprintln(&b, "    location: "+truncate(source.Location, max(16, width-14)))
		if strings.TrimSpace(source.CachePath) != "" {
			_, _ = fmt.Fprintln(&b, "    cache: "+truncate(source.CachePath, max(16, width-11)))
		}
		if strings.TrimSpace(source.Message) != "" {
			_, _ = fmt.Fprintln(&b, indent(wrapText(source.Message, max(16, width-4)), "    "))
		}
	}
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintln(&b, helpStyle.Render("j/k move  u update source  U/s update all  d presets  n custom  r reload"))
	return strings.TrimRight(b.String(), "\n")
}

func sourceStatusCounts(sources []SourcePreset) map[string]int {
	counts := map[string]int{
		core.SourceStatusLocal:   0,
		core.SourceStatusFresh:   0,
		core.SourceStatusStale:   0,
		core.SourceStatusMissing: 0,
		core.SourceStatusError:   0,
	}
	for _, source := range sources {
		status := strings.TrimSpace(source.Status)
		if _, ok := counts[status]; ok {
			counts[status]++
		}
	}
	return counts
}

func (m model) updateContent(width int) string {
	lockPresence := core.ResultMissing
	if m.lockStatus.Present {
		lockPresence = "present"
	}
	lockPath := strings.TrimSpace(m.lockStatus.Path)
	if lockPath == "" {
		lockPath = filepath.Join(m.projectDir, "skills.lock.toml")
	}
	lines := []string{
		titleStyle.Render("Project lockfile"),
		"",
		labelLine("Lockfile", lockPresence),
		labelLine("Path", truncate(lockPath, max(12, width-8))),
		fmt.Sprintf("Rows: %d   Missing: %d   Changed: %d   Unchanged: %d   Skipped: %d",
			m.lockStatus.Total,
			m.lockStatus.Missing,
			m.lockStatus.Changed,
			m.lockStatus.Unchanged,
			m.lockStatus.Skipped,
		),
		"",
		helpStyle.Render("r restore project lockfile"),
		"",
		titleStyle.Render("Update commands"),
		"",
		"Self-update is CLI-only so the TUI does not rewrite the binary while it is running.",
		"",
		labelLine("Command", "skillhub update"),
		labelLine("Cascade", "skillhub update --cascade"),
		labelLine("Verbose", "skillhub update --cascade -v"),
		labelLine("Skills", "skillhub installed update"),
		labelLine("Projects", "skillhub installed usage update --projects"),
		labelLine("Restore", "skillhub restore --project "+m.projectDir),
		"",
		wrapText(
			"Use 1 Installed to update target folders. Project usage maintenance stays CLI-only.",
			width,
		),
	}
	return strings.Join(lines, "\n")
}

func (m model) confirmDeleteContent(width int) string {
	row := m.pendingUninstall
	if strings.TrimSpace(row.Skill) == "" {
		return "No managed skill selected."
	}
	lines := []string{
		titleStyle.Render("Remove managed skill?"),
		"",
		labelLine("Skill", row.Skill),
		labelLine("Target", targetScopeLabel(row.Target, row.Scope)),
		labelLine("Path", row.Path),
		"",
		wrapText(
			"This removes the selected managed skill directory, or only its stale registry entry when "+
				"the directory is missing. Unmanaged skills remain CLI-only.", width,
		),
		"",
		checkedStyle.Render("enter/y confirm") + "  " + helpStyle.Render("esc/n cancel"),
	}
	return strings.Join(lines, "\n")
}

func (m model) installScopeContent(width int) string {
	type scopeChoice struct {
		scope       string
		label       string
		description string
		path        string
	}
	choices := []scopeChoice{
		{
			scope:       scopeProject,
			label:       "Project",
			description: "Current project",
			path:        m.projectDir,
		},
		{
			scope:       scopeGlobal,
			label:       "User",
			description: "Global skills directory",
			path:        "~",
		},
	}
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%s\n\n", labelLine("Install queue", strconv.Itoa(len(m.selectedNames()))))
	for i, choice := range choices {
		cursor := " "
		if i == m.install.scopeCursor {
			cursor = uiSelectedCursor
		}
		marker := checkboxStyle.Render("[ ]")
		if m.install.scope == choice.scope {
			marker = checkedStyle.Render("[✓]")
		}
		title := fmt.Sprintf("%s %s %s", cursor, marker, choice.label)
		if i == m.install.scopeCursor {
			title = activeRowStyle.Render(title)
		}
		if m.install.scope == choice.scope {
			title = selectedRowStyle.Render(title)
		}
		_, _ = fmt.Fprintln(&b, title)
		_, _ = fmt.Fprintln(&b, subtleStyle.Render("    "+choice.description))
		_, _ = fmt.Fprintln(&b, subtleStyle.Render("    "+truncate(choice.path, max(12, width-4))))
		if i != len(choices)-1 {
			_, _ = fmt.Fprintln(&b)
		}
	}
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintln(&b, checkedStyle.Render("enter continue")+"  "+helpStyle.Render("esc cancel"))
	return strings.TrimRight(b.String(), "\n")
}

func (m model) installConfirmContent(width int) string {
	names := m.selectedNames()
	choices := m.selectedTargetChoices()
	if len(names) == 0 {
		return "No skills selected."
	}
	if len(choices) == 0 {
		return "No install agents selected."
	}
	result := buildInstallResult(choices, names)
	var b strings.Builder
	_, _ = fmt.Fprintln(&b, labelLine("Scope", installScopeDisplay(m.install.scope)))
	_, _ = fmt.Fprintln(&b, labelLine("Project", truncate(m.projectDir, max(12, width-9))))
	_, _ = fmt.Fprintln(&b, labelLine("Skills", strconv.Itoa(len(names))))
	_, _ = fmt.Fprintln(&b, labelLine("Agents", strconv.Itoa(len(choices))))
	_, _ = fmt.Fprintln(&b)
	for _, name := range names {
		_, _ = fmt.Fprintln(&b, "- "+name)
	}
	_, _ = fmt.Fprintln(&b)
	for targetIndex, target := range result.Targets {
		if targetIndex > 0 {
			_, _ = fmt.Fprintln(&b)
		}
		_, _ = fmt.Fprintf(&b, "%s %s\n", checkedStyle.Render("[✓]"), target.Label)
		root := truncate(target.Root, max(12, width-22))
		_, _ = fmt.Fprintln(&b, subtleStyle.Render(fmt.Sprintf("    scope: %s   root: %s", target.Scope, root)))
		for _, path := range target.SkillPaths {
			_, _ = fmt.Fprintln(&b, "    "+truncate(path, max(12, width-4)))
		}
	}
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintln(&b, checkedStyle.Render("enter/y install")+"  "+helpStyle.Render("esc back"))
	return strings.TrimRight(b.String(), "\n")
}

func (m model) helpContent(_ int) string {
	lines := []string{
		"1/2/3/4/5 switch sections",
		"left/right  switch sections",
		"j/k         move",
		"space       select or toggle where applicable",
		"enter       open or confirm",
		"/           search skills",
		"u           update highlighted install or source entry",
		"U           update all Sources",
		"x           uninstall highlighted managed skill",
		"s           update all Sources",
		"r           reload current section or restore lockfile in Update",
		"esc         back",
		"q           quit",
		"",
		"[M]         managed by Skillhub; update/uninstall available",
		"[ ]         unmanaged local skill; read-only in TUI",
	}
	return strings.Join(lines, "\n")
}

func (m model) addSourceContent(width int) string {
	input := m.sourceInput
	nameInput := m.sourceNameInput
	locationCursor := ""
	nameCursor := ""
	if m.sourceField == 1 {
		nameCursor = "_"
	} else {
		locationCursor = "_"
	}
	return strings.Join([]string{
		labelLine("Location", input+locationCursor),
		labelLine("Name", nameInput+nameCursor),
		"",
		wrapText(
			"Location expects a local path, git URL, or GitHub tree URL. "+
				"Name is optional; use a short id such as mattpocock for generic repository names. Tab switches fields.",
			width,
		),
	}, "\n")
}

func (m model) defaultsContent(width int) string {
	if len(m.defaults) == 0 {
		return "No source defaults available."
	}
	var b strings.Builder
	window := clampCursorWindow(m.defaultCursor, m.defaultOffset, len(m.defaults), m.defaultVisibleCount())
	for row, source := range m.defaults[window.Start:window.End] {
		i := window.Start + row
		cursor := " "
		if i == window.Cursor {
			cursor = uiSelectedCursor
		}
		line := fmt.Sprintf("%s %-20s %-8s %-12s %s",
			cursor,
			truncate(source.Name, 20),
			truncate(source.Type, 8),
			truncate(source.Ref, 12),
			truncate(source.Location, width-46),
		)
		if i == window.Cursor {
			line = activeRowStyle.Width(width).Render(line)
		}
		_, _ = fmt.Fprintln(&b, line)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) helpText() string {
	if m.sourceProgress.Failed {
		return "enter/esc back to sources  q quit"
	}
	if m.install.progress.Failed {
		return "enter/esc back to targets  q quit"
	}
	if m.viewMode == viewHelp {
		return "enter/esc back  q quit"
	}
	if m.viewMode == viewDefaults {
		return "j/k move  enter add preset  n add custom  esc back  q quit"
	}
	if m.viewMode == viewAddSource {
		return "tab field  enter add  esc back  ctrl+c quit"
	}
	if m.viewMode == viewInstallScope {
		return "j/k move  enter continue  esc cancel  q quit"
	}
	if m.viewMode == viewInstallConfirm {
		return "enter/y install  esc back  q quit"
	}
	if m.viewMode == viewDetails {
		return "space queue  i targets  enter/esc back  q quit"
	}
	if m.viewMode == viewInstalledDetails {
		return "j/k move location  u update  x uninstall  esc back  q quit"
	}
	if m.viewMode == viewInstallResult {
		return "enter/b back  t targets  q quit"
	}
	if m.viewMode == viewConfirmDelete {
		return "enter/y confirm  esc/n cancel  q quit"
	}
	if m.viewMode == viewInstalled {
		return "1-5/left-right sections  j/k move  enter locations  u update single  x uninstall single  r reload  ? help  q quit"
	}
	if m.viewMode == viewSources {
		return "1-5/left-right sections  j/k move  u update source  U/s update all  d presets  n custom  r reload  ? help  q quit"
	}
	if m.viewMode == viewUpdate {
		return "1-5/left-right sections  r restore project lockfile  u installed screen  ? help  q quit"
	}
	if m.viewMode == viewTargets {
		if m.install.targetPurpose == targetPurposeInstall {
			return "j/k move  space toggle  enter/i preview  a all  c clear  b scope  r reload  ? help  q quit"
		}
		return "1-5/left-right sections  j/k move  r reload  ? help  q quit"
	}
	return strings.Join([]string{
		"1-5/left-right sections",
		"j/k move",
		"space queue",
		"enter details",
		"/ search",
		"a all",
		"c clear",
		"d presets",
		"n source",
		"t targets",
		"i install",
		"? help",
		"q quit",
	}, "  ")
}

func (m model) targetsPanelTitle() string {
	if m.install.targetPurpose == targetPurposeInstall {
		return "Install agents"
	}
	return labelTargets
}

func (m model) renderPanel(title, body string, width int) string {
	return panelWithMaxHeight(title, body, width, m.bodyHeight)
}

func panelWithMaxHeight(title, body string, width int, maxHeight int) string {
	if width < 24 {
		width = 24
	}
	contentWidth := width - 6
	if contentWidth < 10 {
		contentWidth = 10
	}
	if maxHeight > 0 {
		bodyBudget := maxHeight - panelChromeHeight()
		if bodyBudget < 0 {
			bodyBudget = 0
		}
		body = limitLines(body, bodyBudget)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, panelTitleStyle.Render(title), "", body)
	rendered := panelStyle.Width(contentWidth).Render(content)
	if maxHeight > 0 && lipgloss.Height(rendered) > maxHeight {
		rendered = lipgloss.NewStyle().MaxHeight(maxHeight).Render(rendered)
	}
	return rendered
}

func panelChromeHeight() int {
	return 6
}

func limitLines(value string, maxLines int) string {
	if maxLines < 0 {
		maxLines = 0
	}
	if maxLines == 0 || strings.TrimSpace(value) == "" {
		return ""
	}
	lines := strings.Split(value, "\n")
	if len(lines) <= maxLines {
		return value
	}
	return strings.Join(lines[:maxLines], "\n")
}
