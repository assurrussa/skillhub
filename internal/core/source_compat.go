package core

import (
	"path/filepath"
	"strings"
)

const generatedSkillCompatibilityCategory = "__skillhub_compat__"

func deriveSourceAlias(location string) string {
	if owner, repo, ok := githubSourceOwnerRepo(location); ok {
		return owner + "-" + repo
	}
	clean := strings.TrimRight(strings.TrimSpace(location), "/")
	base := filepath.Base(clean)
	return strings.TrimSuffix(base, ".git")
}

func githubSourceOwnerRepo(location string) (owner, repo string, ok bool) {
	clean := strings.TrimSpace(location)
	clean = strings.SplitN(clean, "#", 2)[0]
	clean = strings.SplitN(clean, "?", 2)[0]
	clean = strings.TrimRight(clean, "/")

	var path string
	switch {
	case strings.HasPrefix(clean, "https://github.com/"):
		path = strings.TrimPrefix(clean, "https://github.com/")
	case strings.HasPrefix(clean, "http://github.com/"):
		path = strings.TrimPrefix(clean, "http://github.com/")
	case strings.HasPrefix(clean, "git@github.com:"):
		path = strings.TrimPrefix(clean, "git@github.com:")
	case strings.HasPrefix(clean, "ssh://git@github.com/"):
		path = strings.TrimPrefix(clean, "ssh://git@github.com/")
	default:
		return "", "", false
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	owner = normalizeSourceName(parts[0])
	repo = normalizeSourceName(strings.TrimSuffix(parts[1], ".git"))
	if !isValidID(owner) || !isValidID(repo) {
		return "", "", false
	}
	return owner, repo, true
}

func generatedSkillFlatName(sourcePath, skillFile, rel string) string {
	legacyName := strings.ReplaceAll(rel, "/", "_")
	skillDir := filepath.Dir(skillFile)
	if _, pluginLayout := pluginSkillRelativeDir(sourcePath, skillDir); !pluginLayout {
		return legacyName
	}

	// Canonical plugin bundles such as humanlayer/skills already satisfy the
	// Agent Skills name-to-parent-directory contract. Preserve that upstream
	// name instead of duplicating the plugin prefix. Older bundles may use
	// non-ID frontmatter names (for example react:components); keep the previous
	// path-prefixed materialization for those sources to avoid regressions.
	dirName := normalizeSourceName(filepath.Base(skillDir))
	frontmatterName := normalizeSourceName(skillFrontmatterValue(skillFile, "name"))
	if frontmatterName == "" {
		frontmatterName = dirName
	}
	// Legacy plugin IDs always contain an underscore. Keep that namespace
	// path-prefixed even before a collision exists, so adding another plugin
	// later cannot reassign an already-installed shortened ID.
	if isValidID(dirName) && !strings.Contains(dirName, "_") && frontmatterName == dirName {
		return dirName
	}
	return legacyName
}

func isGeneratedSkillCompatibilityRow(row CatalogRow) bool {
	return row.Category == generatedSkillCompatibilityCategory
}

func (b *Backend) shouldHideGeneratedSkillCompatibilityRow(source Source, sourcePath string, row CatalogRow) bool {
	if !isGeneratedSkillCompatibilityRow(row) {
		return false
	}
	generatedPath, err := b.generatedSourcePath(source.Name)
	if err != nil {
		return false
	}
	return filepath.Clean(sourcePath) == filepath.Clean(generatedPath)
}
