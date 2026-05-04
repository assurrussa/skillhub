package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Skill struct {
	Source      string
	Name        string
	Category    string
	Triggers    string
	Description string
}

type SourcePreset struct {
	Name     string
	Type     string
	Location string
	Ref      string
	Catalog  string
}

const (
	viewSkills    = "skills"
	viewDefaults  = "defaults"
	viewAddSource = "add-source"
)

type model struct {
	repoRoot string

	skills   []Skill
	filtered []int
	selected map[string]bool
	defaults []SourcePreset

	cursor         int
	defaultCursor  int
	offset         int
	width          int
	height         int
	search         string
	sourceInput    string
	searchMode     bool
	loading        bool
	busy           bool
	noSources      bool
	status         string
	viewMode       string
	reloadOnFinish bool

	installScope string
	projectDir   string
}

type skillsLoadedMsg struct {
	skills []Skill
	err    error
}

type defaultsLoadedMsg struct {
	defaults []SourcePreset
	err      error
}

type commandDoneMsg struct {
	action string
	output string
	err    error
}

func Run(repoRoot string) error {
	p := tea.NewProgram(initialModel(repoRoot), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func initialModel(repoRoot string) model {
	return model{
		repoRoot: repoRoot,
		selected: map[string]bool{},
		loading:  true,
		status:   "Loading catalog...",
		viewMode: viewSkills,

		installScope: "global",
		projectDir:   callerCwd(),
	}
}

func (m model) Init() tea.Cmd {
	return loadSkills(m.repoRoot)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorVisible()
		return m, nil
	case skillsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			if isNoSourcesError(msg.err) {
				m.skills = nil
				m.filtered = nil
				m.noSources = true
				m.status = "No sources configured. Press d for presets or n for a custom source."
				return m, nil
			}
			m.status = "Load failed: " + msg.err.Error()
			return m, nil
		}
		m.noSources = false
		m.skills = msg.skills
		m.applyFilter()
		m.status = fmt.Sprintf("Loaded %d skill(s).", len(m.skills))
		return m, nil
	case defaultsLoadedMsg:
		m.loading = false
		m.viewMode = viewDefaults
		if msg.err != nil {
			m.status = "Load source defaults failed: " + msg.err.Error()
			return m, nil
		}
		m.defaults = msg.defaults
		if m.defaultCursor >= len(m.defaults) {
			m.defaultCursor = len(m.defaults) - 1
		}
		if m.defaultCursor < 0 {
			m.defaultCursor = 0
		}
		m.status = fmt.Sprintf("Loaded %d source default(s).", len(m.defaults))
		return m, nil
	case commandDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.reloadOnFinish = false
			m.status = fmt.Sprintf("%s failed: %s", msg.action, msg.err)
			if strings.TrimSpace(msg.output) != "" {
				m.status += " | " + compactOutput(msg.output)
			}
			return m, nil
		}
		m.status = fmt.Sprintf("%s complete.", msg.action)
		if strings.TrimSpace(msg.output) != "" {
			m.status += " " + compactOutput(msg.output)
		}
		if msg.action == "Sync" || m.reloadOnFinish {
			m.reloadOnFinish = false
			m.viewMode = viewSkills
			m.loading = true
			m.status = "Reloading catalog..."
			return m, loadSkills(m.repoRoot)
		}
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	default:
		return m, nil
	}
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if m.searchMode {
		switch msg.String() {
		case "esc":
			m.searchMode = false
			return m, nil
		case "enter":
			m.searchMode = false
			return m, nil
		case "backspace":
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

	if m.loading || m.busy {
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		default:
			return m, nil
		}
	}

	if m.viewMode == viewAddSource {
		return m.updateAddSourceKey(msg)
	}

	if m.viewMode == viewDefaults {
		return m.updateDefaultsKey(msg)
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "/":
		m.searchMode = true
		return m, nil
	case "esc":
		m.search = ""
		m.applyFilter()
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
		return m, nil
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}
		return m, nil
	case " ":
		m.toggleCurrent()
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
		names := m.selectedNames()
		if len(names) == 0 {
			m.status = "Select at least one skill before installing."
			return m, nil
		}
		m.busy = true
		m.status = fmt.Sprintf("Installing %d skill(s) to %s...", len(names), m.installTargetSummary())
		installArgs := append([]string{"install"}, m.installArgs()...)
		installArgs = append(installArgs, names...)
		return m, runSkillCommand(m.repoRoot, "Install", installArgs...)
	case "d":
		m.loading = true
		m.status = "Loading source defaults..."
		return m, loadDefaultSources(m.repoRoot)
	case "n":
		m.sourceInput = ""
		m.viewMode = viewAddSource
		m.status = "Enter a source path or git URL."
		return m, nil
	case "t":
		if m.installScope == "global" {
			m.installScope = "project"
		} else {
			m.installScope = "global"
		}
		m.status = "Target changed to " + m.installTargetSummary() + "."
		return m, nil
	case "r":
		m.loading = true
		m.status = "Reloading catalog..."
		return m, loadSkills(m.repoRoot)
	case "s":
		m.busy = true
		m.status = "Syncing sources..."
		return m, runSourceCommand(m.repoRoot, "Sync", "sync")
	default:
		return m, nil
	}
}

func (m model) updateAddSourceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.viewMode = viewSkills
		m.status = "Returned to skills."
		return m, nil
	case "enter":
		location := strings.TrimSpace(m.sourceInput)
		if location == "" {
			m.status = "Enter a source path or git URL before adding."
			return m, nil
		}
		m.busy = true
		m.reloadOnFinish = true
		m.status = "Adding source " + location + "..."
		return m, runSourceCommand(m.repoRoot, "Add source", "add", location)
	case "backspace":
		if m.sourceInput != "" {
			runes := []rune(m.sourceInput)
			m.sourceInput = string(runes[:len(runes)-1])
		}
		return m, nil
	}
	if msg.Type == tea.KeyRunes {
		m.sourceInput += msg.String()
		return m, nil
	}
	return m, nil
}

func (m model) updateDefaultsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.viewMode = viewSkills
		m.status = "Returned to skills."
		return m, nil
	case "up", "k":
		if m.defaultCursor > 0 {
			m.defaultCursor--
		}
		return m, nil
	case "down", "j":
		if m.defaultCursor < len(m.defaults)-1 {
			m.defaultCursor++
		}
		return m, nil
	case "enter", "a":
		if len(m.defaults) == 0 {
			m.status = "No source defaults available."
			return m, nil
		}
		source := m.defaults[m.defaultCursor]
		m.busy = true
		m.reloadOnFinish = true
		m.status = "Adding source " + source.Name + "..."
		return m, runSourceCommand(m.repoRoot, "Add source", "defaults", "add", source.Name)
	case "n":
		m.sourceInput = ""
		m.viewMode = viewAddSource
		m.status = "Enter a source path or git URL."
		return m, nil
	default:
		return m, nil
	}
}

func (m model) View() string {
	var b strings.Builder

	if m.width == 0 {
		m.width = 80
	}

	fmt.Fprintln(&b, "Skillhub")
	fmt.Fprintln(&b, "========")
	if m.searchMode {
		fmt.Fprintf(&b, "Search: %s_\n", m.search)
	} else {
		fmt.Fprintf(&b, "Search: %s\n", emptyLabel(m.search, "none"))
	}
	fmt.Fprintf(&b, "Selected: %d\n\n", len(m.selected))
	fmt.Fprintf(&b, "Target: %s\n\n", m.installTargetSummary())

	if m.loading {
		fmt.Fprintln(&b, m.status)
		return b.String()
	}

	if m.viewMode == viewAddSource {
		m.writeAddSourceView(&b)
	} else if m.viewMode == viewDefaults {
		m.writeDefaultsView(&b)
	} else if m.noSources {
		fmt.Fprintln(&b, "No sources configured.")
		fmt.Fprintln(&b, "Press d for presets or n for a custom source.")
	} else if len(m.filtered) == 0 {
		fmt.Fprintln(&b, "No skills matched.")
	} else {
		visible := m.visibleCount()
		end := m.offset + visible
		if end > len(m.filtered) {
			end = len(m.filtered)
		}
		for row, idx := range m.filtered[m.offset:end] {
			skill := m.skills[idx]
			cursor := " "
			if m.offset+row == m.cursor {
				cursor = ">"
			}
			checked := " "
			if m.selected[skill.Key()] {
				checked = "x"
			}
			fmt.Fprintf(&b, "%s [%s] %-18s %-28s %-14s %s\n",
				cursor,
				checked,
				truncate(skill.Source, 18),
				truncate(skill.Name, 28),
				truncate(skill.Category, 14),
				truncate(skill.Description, m.descriptionWidth()),
			)
		}
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, m.status)
	fmt.Fprintln(&b)
	if m.viewMode == viewDefaults {
		fmt.Fprintln(&b, "j/k move  enter add preset  n add custom  esc back  q quit")
	} else if m.viewMode == viewAddSource {
		fmt.Fprintln(&b, "enter add  esc back  q quit")
	} else {
		fmt.Fprintln(&b, "j/k move  space select  / search  a all  c clear  d presets  n source  t target  i install  s sync  r reload  q quit")
	}
	return b.String()
}

func (m model) writeAddSourceView(b *strings.Builder) {
	fmt.Fprintln(b, "Add source")
	fmt.Fprintln(b)
	fmt.Fprintf(b, "Path or git URL: %s_\n", m.sourceInput)
	fmt.Fprintln(b)
	fmt.Fprintln(b, "Name defaults to the path or repo basename. Use CLI flags for custom name/type/ref/catalog.")
}

func (m model) writeDefaultsView(b *strings.Builder) {
	if len(m.defaults) == 0 {
		fmt.Fprintln(b, "No source defaults available.")
		return
	}
	for i, source := range m.defaults {
		cursor := " "
		if i == m.defaultCursor {
			cursor = ">"
		}
		fmt.Fprintf(b, "%s %-20s %-8s %-12s %s\n",
			cursor,
			truncate(source.Name, 20),
			truncate(source.Type, 8),
			truncate(source.Ref, 12),
			truncate(source.Location, m.defaultsLocationWidth()),
		)
	}
}

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
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.offset = 0
	m.ensureCursorVisible()
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
}

func (m model) visibleCount() int {
	if m.height <= 11 {
		return 10
	}
	return m.height - 11
}

func (m model) descriptionWidth() int {
	if m.width <= 90 {
		return 24
	}
	return m.width - 70
}

func (m model) defaultsLocationWidth() int {
	if m.width <= 70 {
		return 32
	}
	return m.width - 46
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

func (s Skill) Key() string {
	if strings.TrimSpace(s.Source) == "" {
		return s.Name
	}
	return s.Source + "/" + s.Name
}

func (m model) selectedNames() []string {
	names := make([]string, 0, len(m.selected))
	for name := range m.selected {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m model) installArgs() []string {
	if m.installScope == "project" {
		return []string{"--target", "codex", "--scope", "project", "--project", m.projectDir}
	}
	return nil
}

func (m model) installTargetSummary() string {
	return fmt.Sprintf("codex/%s -> %s", m.installScope, m.installTargetPath())
}

func (m model) installTargetPath() string {
	if m.installScope == "project" {
		return filepath.Join(m.projectDir, ".agents", "skills")
	}
	if dir := os.Getenv("AGENT_SKILLS_DIR"); strings.TrimSpace(dir) != "" {
		return dir
	}
	if home := os.Getenv("HOME"); strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".agents", "skills")
	}
	return "~/.agents/skills"
}

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
		output, err := runScript(repoRoot, "scripts/skills.sh", "list", "--tsv")
		if err != nil {
			return skillsLoadedMsg{err: commandError(err, output)}
		}
		skills, err := parseSkillsTSV(output)
		return skillsLoadedMsg{skills: skills, err: err}
	}
}

func loadDefaultSources(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/sources.sh", "defaults", "list", "--tsv")
		if err != nil {
			return defaultsLoadedMsg{err: commandError(err, output)}
		}
		defaults, err := parseDefaultSourcesTSV(output)
		return defaultsLoadedMsg{defaults: defaults, err: err}
	}
}

func runSkillCommand(repoRoot, action string, args ...string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/skills.sh", args...)
		return commandDoneMsg{action: action, output: output, err: commandError(err, output)}
	}
}

func runSourceCommand(repoRoot, action string, args ...string) tea.Cmd {
	return func() tea.Msg {
		output, err := runScript(repoRoot, "scripts/sources.sh", args...)
		return commandDoneMsg{action: action, output: output, err: commandError(err, output)}
	}
}

func runScript(repoRoot, script string, args ...string) (string, error) {
	cmd := exec.Command("sh", append([]string{filepath.Join(repoRoot, script)}, args...)...)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	if os.Getenv("SKILLHUB_CALLER_CWD") == "" {
		cmd.Env = append(cmd.Env, "SKILLHUB_CALLER_CWD="+callerCwd())
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func parseSkillsTSV(input string) ([]Skill, error) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 || lines[0] != "source\tname\tcategory\ttriggers\tdescription" {
		return nil, fmt.Errorf("unexpected skills TSV header")
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
		return nil, fmt.Errorf("unexpected source defaults TSV header")
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

func commandError(err error, output string) error {
	if err == nil {
		return nil
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, compactOutput(output))
}

func isNoSourcesError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "No sources configured")
}

func compactOutput(output string) string {
	lines := strings.Fields(strings.ReplaceAll(output, "\n", " "))
	if len(lines) == 0 {
		return ""
	}
	return truncate(strings.Join(lines, " "), 140)
}

func truncate(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func emptyLabel(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
