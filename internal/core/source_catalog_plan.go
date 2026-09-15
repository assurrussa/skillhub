package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	generatedCatalogVersionFile = ".skillhub-catalog-version"
	generatedCatalogVersion     = "legacy-owned-ids-v1\n"
)

type generatedSkillPlan struct {
	file       string
	rel        string
	name       string
	legacyName string
	root       bool
}

// planGeneratedSkills reserves IDs used by previous releases before assigning
// shorter names. An ID must never change owners just because another plugin's
// frontmatter happens to match it. Planning is read-only and linear in skills.
func planGeneratedSkills(sourceName, sourcePath string, skillFiles []string) ([]generatedSkillPlan, error) {
	plans := make([]generatedSkillPlan, 0, len(skillFiles))
	preferredCounts := make(map[string]int, len(skillFiles))
	preferredByPath := make(map[string]string, len(skillFiles))
	legacyOwners := make(map[string]string, len(skillFiles))
	for _, file := range skillFiles {
		name, rel, root, ok, err := generatedSkillIdentity(sourceName, sourcePath, file)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		preferredCounts[name]++
		preferredByPath[rel] = name
		legacyName := strings.ReplaceAll(rel, "/", "_")
		if isValidID(legacyName) {
			if previous, exists := legacyOwners[legacyName]; exists {
				if preferredByPath[previous] == legacyName && name == legacyName {
					return nil, fmt.Errorf(
						"duplicate generated skill name %s in source %s: %s and %s",
						legacyName,
						sourceName,
						previous,
						rel,
					)
				}
				return nil, fmt.Errorf("ambiguous legacy skill name %s in source %s: %s and %s", legacyName, sourceName, previous, rel)
			}
			legacyOwners[legacyName] = rel
		} else {
			// No valid legacy ID could have been installed from this path.
			legacyName = ""
		}
		plans = append(plans, generatedSkillPlan{file: file, rel: rel, name: name, legacyName: legacyName, root: root})
	}

	for i := range plans {
		plan := &plans[i]
		owner, reserved := legacyOwners[plan.name]
		preferredCollision := preferredCounts[plan.name] > 1
		if (!reserved || owner != plan.rel) && (reserved || preferredCollision) {
			if plan.legacyName == "" {
				return nil, fmt.Errorf("skill %s in source %s conflicts with legacy ID owned by %s", plan.rel, sourceName, owner)
			}
			// a/foo retains a_foo; b/a_foo is installed as b_a_foo, not a_foo.
			plan.name = plan.legacyName
		}
	}
	assignedOwners := make(map[string]string, len(plans))
	for _, plan := range plans {
		if previous, exists := assignedOwners[plan.name]; exists {
			return nil, fmt.Errorf(
				"duplicate generated skill name %s in source %s: %s and %s",
				plan.name,
				sourceName,
				previous,
				plan.rel,
			)
		}
		assignedOwners[plan.name] = plan.rel
	}
	return plans, nil
}

func buildPlannedCatalogRows(sourceName, sourcePath, tmpPath string, skillFiles []string) ([]CatalogRow, error) {
	plans, err := planGeneratedSkills(sourceName, sourcePath, skillFiles)
	if err != nil {
		return nil, err
	}
	rows := make([]CatalogRow, 0, len(plans)*2)
	for _, plan := range plans {
		if err := copyGeneratedSkillDir(filepath.Dir(plan.file), filepath.Join(tmpPath, "skills", plan.name), plan.root); err != nil {
			return nil, err
		}
		row := CatalogRow{
			Name:        plan.name,
			Category:    generatedSkillCategory(plan.rel),
			Triggers:    generatedSkillTriggers(plan.file, plan.rel),
			Description: generatedSkillDescription(plan.file, plan.rel),
		}
		rows = append(rows, row)
		if plan.legacyName == "" || plan.legacyName == plan.name {
			continue
		}
		legacyPath := filepath.Join(tmpPath, "skills", plan.legacyName)
		if err := copyGeneratedSkillDir(filepath.Dir(plan.file), legacyPath, plan.root); err != nil {
			return nil, err
		}
		row.Name = plan.legacyName
		row.Category = generatedSkillCompatibilityCategory
		rows = append(rows, row)
	}
	return rows, nil
}

func generatedCatalogIsCurrent(path string) bool {
	version, err := os.ReadFile(filepath.Join(path, generatedCatalogVersionFile))
	return err == nil && string(version) == generatedCatalogVersion
}
