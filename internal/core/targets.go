package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

var targetsTable = Table[Target]{
	Header:  TargetsHeader,
	Columns: 5,
	Parse: func(row []string) (Target, error) {
		return Target{ID: row[0], Label: row[1], Status: row[2], Adapter: row[3], Description: row[4]}, nil
	},
}

func (b *Backend) ListTargets() ([]Target, error) {
	rows, err := targetsTable.ReadFile(b.TargetsFile())
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	plannedSeen := false
	for _, target := range rows {
		if !isValidID(target.ID) {
			return nil, fmt.Errorf("invalid target id: %s", target.ID)
		}
		if target.Status != StatusSupported && target.Status != StatusPlanned {
			return nil, fmt.Errorf("unsupported target status for %s: %s", target.ID, target.Status)
		}
		if target.Status == StatusPlanned {
			plannedSeen = true
		}
		if target.Status == StatusSupported && plannedSeen {
			return nil, fmt.Errorf("supported target must be listed before planned targets: %s", target.ID)
		}
		if target.Label == "" || target.Adapter == "" || target.Description == "" {
			return nil, fmt.Errorf("target row has empty fields for %s", target.ID)
		}
		if seen[target.ID] {
			return nil, fmt.Errorf("duplicate target: %s", target.ID)
		}
		seen[target.ID] = true
	}
	if len(rows) == 0 {
		return nil, errors.New("no targets configured")
	}
	return rows, nil
}

func (b *Backend) DetectTargets(project string) ([]TargetDetection, error) {
	targets, err := b.ListTargets()
	if err != nil {
		return nil, err
	}
	rows := []TargetDetection{}
	for _, target := range targets {
		if target.ID == TargetDirectory {
			rows = append(rows, TargetDetection{
				Target: target.ID, Scope: ScopeCustom, Status: target.Status,
				Path: "-", Exists: "-", Skills: "-", Managed: "-",
			})
			continue
		}
		if target.Status != StatusSupported || target.Adapter != AdapterSkillDir {
			continue
		}
		for _, scope := range []string{ScopeGlobal, ScopeProject} {
			root, err := b.TargetRoot(TargetRootOptions{
				Target: target.ID, Scope: scope, Project: project,
				ScopeWasSet: true,
			})
			if err != nil {
				return nil, err
			}
			skills, managed := b.countSkillDirs(root, scope)
			exists := ManagedNo
			if info, err := os.Stat(root); err == nil && info.IsDir() {
				exists = ManagedYes
			}
			rows = append(rows, TargetDetection{
				Target:  target.ID,
				Scope:   scope,
				Status:  target.Status,
				Path:    root,
				Exists:  exists,
				Skills:  strconv.Itoa(skills),
				Managed: strconv.Itoa(managed),
			})
		}
	}
	return rows, nil
}

func (b *Backend) countSkillDirs(root, scope string) (total, managed int) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, 0
	}
	usage, _ := b.ReadUsage("")
	managedPaths := map[string]bool{}
	for _, row := range usage {
		managedPaths[row.InstalledPath] = true
	}
	for _, entry := range entries {
		dir := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
			continue
		}
		total++
		if scope == ScopeProject {
			if managedPaths[dir] {
				managed++
			}
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, ".skillhub.json")); err == nil {
			managed++
		}
	}
	return total, managed
}

func SortTargetsForDisplay(targets []Target) []Target {
	out := append([]Target(nil), targets...)
	sort.SliceStable(out, func(i, j int) bool {
		return targetRank(out[i].ID) < targetRank(out[j].ID)
	})
	return out
}

func targetRank(target string) int {
	switch target {
	case TargetCodex:
		return 0
	case TargetDirectory:
		return 1
	case TargetClaude:
		return 2
	case TargetGemini:
		return 3
	case "opencode":
		return 4
	default:
		return 20
	}
}
