package tui

import (
	"fmt"
	"strings"
)

func isNoSourcesError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no sources configured")
}

func compactOutput(output string) string {
	lines := strings.Fields(strings.ReplaceAll(output, "\n", " "))
	if len(lines) == 0 {
		return ""
	}
	return truncate(strings.Join(lines, " "), 140)
}

func truncate(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func emptyLabel(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func labelLine(label, value string) string {
	return subtleStyle.Render(fmt.Sprintf("%-9s", label)) + " " + value
}

func wrapText(value string, width int) string {
	value = strings.TrimSpace(value)
	if value == "" || width <= 0 {
		return value
	}
	words := strings.Fields(value)
	if len(words) == 0 {
		return ""
	}
	lines := []string{}
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) > width {
			lines = append(lines, current)
			current = word
			continue
		}
		current += " " + word
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n")
}

func indent(value, prefix string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
