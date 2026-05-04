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

type model struct {
	repoRoot string

	skills   []Skill
	filtered []int
	selected map[string]bool

	cursor     int
	offset     int
	width      int
	height     int
	search     string
	searchMode bool
	loading    bool
	busy       bool
	status     string
}

type skillsLoadedMsg struct {
	skills []Skill
	err    error
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
			m.status = "Load failed: " + msg.err.Error()
			return m, nil
		}
		m.skills = msg.skills
		m.applyFilter()
		m.status = fmt.Sprintf("Loaded %d skill(s).", len(m.skills))
		return m, nil
	case commandDoneMsg:
		m.busy = false
		if msg.err != nil {
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
		if msg.action == "Sync" {
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
			m.selected[m.skills[idx].Name] = true
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
		m.status = fmt.Sprintf("Installing %d skill(s)...", len(names))
		return m, runSkillCommand(m.repoRoot, "Install", append([]string{"install"}, names...)...)
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

	if m.loading {
		fmt.Fprintln(&b, "Loading catalog...")
		return b.String()
	}

	if len(m.filtered) == 0 {
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
			if m.selected[skill.Name] {
				checked = "x"
			}
			fmt.Fprintf(&b, "%s [%s] %-28s %-14s %s\n",
				cursor,
				checked,
				truncate(skill.Name, 28),
				truncate(skill.Category, 14),
				truncate(skill.Description, m.descriptionWidth()),
			)
		}
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, m.status)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "j/k move  space select  / search  a all  c clear  i install  s sync  r reload  q quit")
	return b.String()
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
	if m.height <= 9 {
		return 10
	}
	return m.height - 9
}

func (m model) descriptionWidth() int {
	if m.width <= 70 {
		return 24
	}
	return m.width - 51
}

func (m *model) toggleCurrent() {
	if len(m.filtered) == 0 {
		return
	}
	skill := m.skills[m.filtered[m.cursor]]
	if m.selected[skill.Name] {
		delete(m.selected, skill.Name)
		m.status = "Unselected " + skill.Name + "."
	} else {
		m.selected[skill.Name] = true
		m.status = "Selected " + skill.Name + "."
	}
}

func (m model) selectedNames() []string {
	names := make([]string, 0, len(m.selected))
	for name := range m.selected {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
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
