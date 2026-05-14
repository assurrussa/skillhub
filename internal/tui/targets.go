package tui

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/assurrussa/skillhub/internal/core"
)

func (m model) buildTargetChoices(targets []Target) []InstallTargetChoice {
	choices := []InstallTargetChoice{}
	for _, target := range sortedTargetsForDisplay(targets) {
		if target.ID == targetDirectory {
			choices = append(choices, InstallTargetChoice{
				Key:         targetDirectory,
				Target:      target.ID,
				Label:       target.Label,
				Status:      "CLI only",
				Path:        "use --dir <path>",
				Description: target.Description,
				Supported:   false,
			})
			continue
		}

		if target.Status == targetStatusSupported &&
			target.Adapter == targetAdapterSkillDir &&
			isNativeScopedTarget(target.ID) {
			for _, scope := range []string{scopeGlobal, scopeProject} {
				choices = append(choices, InstallTargetChoice{
					Key:         target.ID + ":" + scope,
					Target:      target.ID,
					Label:       target.Label + " " + scope,
					Scope:       scope,
					Status:      targetStatusSupported,
					Path:        m.targetSkillRoot(target.ID, scope),
					Description: target.Description,
					Supported:   true,
				})
				choice := &choices[len(choices)-1]
				if detection, ok := m.targetStats[choice.Key]; ok {
					choice.Status = detection.Status
					choice.Path = detection.Path
					choice.Exists = detection.Exists
					choice.Skills = detection.Skills
					choice.Managed = detection.Managed
				}
			}
			continue
		}

		choices = append(choices, InstallTargetChoice{
			Key:         target.ID,
			Target:      target.ID,
			Label:       target.Label,
			Status:      "planned",
			Description: target.Description,
			Supported:   false,
		})
	}
	return choices
}

func (m model) buildInstallAgentChoices(targets []Target) []InstallTargetChoice {
	choices := []InstallTargetChoice{}
	for _, target := range sortedTargetsForDisplay(targets) {
		if target.Status != targetStatusSupported ||
			target.Adapter != targetAdapterSkillDir ||
			!isNativeScopedTarget(target.ID) {
			continue
		}
		choice := InstallTargetChoice{
			Key:         target.ID + ":" + m.install.scope,
			Target:      target.ID,
			Label:       target.Label,
			Scope:       m.install.scope,
			Status:      targetStatusSupported,
			Path:        m.targetSkillRoot(target.ID, m.install.scope),
			Description: target.Description,
			Supported:   true,
		}
		if detection, ok := m.targetStats[choice.Key]; ok {
			choice.Status = detection.Status
			choice.Path = detection.Path
			choice.Exists = detection.Exists
			choice.Skills = detection.Skills
			choice.Managed = detection.Managed
		}
		choices = append(choices, choice)
	}
	return choices
}

func sortedTargetsForDisplay(targets []Target) []Target {
	ordered := append([]Target(nil), targets...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return targetStatusRank(ordered[i].Status) < targetStatusRank(ordered[j].Status)
	})
	return ordered
}

func targetStatusRank(status string) int {
	if status == "planned" {
		return 1
	}
	return 0
}

func indexTargetDetections(detections []TargetDetection) map[string]TargetDetection {
	result := map[string]TargetDetection{}
	for _, detection := range detections {
		result[detection.Target+":"+detection.Scope] = detection
	}
	return result
}

func (m *model) seedSelectedTargets() {
	previous := m.install.selectedTargets
	m.install.selectedTargets = map[string]bool{}
	for _, choice := range m.install.targetChoices {
		if previous[choice.Key] && choice.Supported {
			m.install.selectedTargets[choice.Key] = true
		}
	}
	if len(m.install.selectedTargets) > 0 {
		return
	}

	if m.install.targetPurpose == targetPurposeInstall {
		for _, choice := range m.install.targetChoices {
			if choice.Supported && choice.Exists == managedYes {
				m.install.selectedTargets[choice.Key] = true
			}
		}
		if len(m.install.selectedTargets) > 0 {
			return
		}
	}

	defaultKey := "codex:" + m.install.scope
	for _, choice := range m.install.targetChoices {
		if choice.Key == defaultKey && choice.Supported {
			m.install.selectedTargets[choice.Key] = true
			return
		}
	}
	for _, choice := range m.install.targetChoices {
		if choice.Supported {
			m.install.selectedTargets[choice.Key] = true
			return
		}
	}
}

func (m model) selectedTargetChoices() []InstallTargetChoice {
	choices := []InstallTargetChoice{}
	for _, choice := range m.install.targetChoices {
		if choice.Supported && m.install.selectedTargets[choice.Key] {
			choices = append(choices, choice)
		}
	}
	return choices
}

func buildInstallResult(choices []InstallTargetChoice, names []string) InstallResult {
	result := InstallResult{
		SkillNames: append([]string(nil), names...),
		Targets:    make([]InstallTargetResult, 0, len(choices)),
	}
	for _, choice := range choices {
		target := InstallTargetResult{
			Label:  choice.Label,
			Target: choice.Target,
			Scope:  choice.Scope,
			Root:   choice.Path,
		}
		for _, name := range names {
			if target.Root == "" {
				target.SkillPaths = append(target.SkillPaths, installSkillDirName(name))
				continue
			}
			target.SkillPaths = append(target.SkillPaths, filepath.Join(target.Root, installSkillDirName(name)))
		}
		result.Targets = append(result.Targets, target)
	}
	return result
}

func installSkillDirName(name string) string {
	if index := strings.LastIndex(name, "/"); index >= 0 {
		return name[index+1:]
	}
	return name
}

func (m model) selectedNames() []string {
	names := make([]string, 0, len(m.selected))
	for name := range m.selected {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m model) installTargetLabel() string {
	if m.install.scope == scopeProject {
		return "Codex project"
	}
	return "Codex global"
}

func installScopeDisplay(scope string) string {
	if scope == scopeProject {
		return "Project"
	}
	return "User"
}

func (m model) installTargetPath() string {
	return m.targetSkillRoot(targetCodex, m.install.scope)
}

func (m model) targetSkillRoot(target, scope string) string {
	backend, err := newBackend(m.repoRoot)
	if err != nil {
		return ""
	}
	root, _ := backend.TargetRoot(core.TargetRootOptions{
		Target:  target,
		Scope:   scope,
		Project: m.projectDir,
	})
	return root
}

func isNativeScopedTarget(target string) bool {
	switch target {
	case targetCodex, targetClaude, targetGemini, targetOpenCode:
		return true
	default:
		return false
	}
}
