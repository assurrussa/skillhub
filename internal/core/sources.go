package core

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const sourceTTL = 10 * time.Minute

var sourcesTable = Table[Source]{
	Header:  SourcesHeader,
	Columns: 5,
	Parse: func(row []string) (Source, error) {
		return Source{Name: row[0], Type: row[1], Location: row[2], Ref: row[3], Catalog: row[4]}, nil
	},
	Format: func(s Source) []string {
		return []string{s.Name, s.Type, s.Location, s.Ref, s.Catalog}
	},
}

func (b *Backend) ListSources() ([]Source, error) {
	file, err := b.UserSourcesFile()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return []Source{}, nil
	}
	rows, err := sourcesTable.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return validateSources(rows)
}

func (b *Backend) ListDefaultSources() ([]Source, error) {
	rows, err := sourcesTable.ReadFile(b.DefaultSourcesFile())
	if err != nil {
		return nil, err
	}
	return validateSources(rows)
}

func (b *Backend) ListSourceStatuses() ([]SourceStatus, error) {
	sources, err := b.ListSources()
	if err != nil {
		return nil, err
	}
	statuses := make([]SourceStatus, 0, len(sources))
	for _, source := range sources {
		statuses = append(statuses, b.sourceStatus(source))
	}
	return statuses, nil
}

func (b *Backend) sourceStatus(source Source) SourceStatus {
	status := SourceStatus{
		Name:     source.Name,
		Type:     source.Type,
		Ref:      source.Ref,
		Catalog:  source.Catalog,
		Location: source.Location,
	}
	sourcePath, err := b.SourcePath(source)
	if err != nil {
		status.Status = SourceStatusError
		status.Message = err.Error()
		return status
	}
	status.CachePath = sourcePath
	if isLocalSourceOverride(source, b.envValue("SKILLHUB_AGENT_RULES_PATH")) || source.Type == SourceTypePath {
		return b.localSourceStatus(source, sourcePath, status)
	}
	if source.Type != SourceTypeGit {
		status.Status = SourceStatusError
		status.Message = "unsupported source type: " + source.Type
		return status
	}
	return b.gitSourceStatus(source, sourcePath, status)
}

func (b *Backend) localSourceStatus(source Source, sourcePath string, status SourceStatus) SourceStatus {
	if info, err := os.Stat(sourcePath); err != nil || !info.IsDir() {
		status.Status = SourceStatusError
		status.Message = "path source does not exist: " + sourcePath
		return status
	}
	if _, err := os.Stat(filepath.Join(sourcePath, source.Catalog)); err == nil {
		status.Status = SourceStatusLocal
		status.Message = "local source catalog is available"
		return status
	}
	if generatedPath, ok := b.generatedCatalogPath(source.Name, source.Catalog); ok {
		status.Status = SourceStatusLocal
		status.CachePath = generatedPath
		status.Message = "local source materialized catalog is available"
		return status
	}
	if b.sourceHasDiscoverableSkills(sourcePath) {
		status.Status = SourceStatusLocal
		status.Message = "local source has discoverable skills"
		return status
	}
	status.Status = SourceStatusError
	status.Message = missingSourceCatalogError(source.Name, sourcePath, source.Catalog).Error()
	return status
}

func (b *Backend) gitSourceStatus(source Source, sourcePath string, status SourceStatus) SourceStatus {
	syncedAt, hasSync, syncErr := b.readSourceSyncedAt(source.Name)
	if syncErr != nil {
		status.Status = SourceStatusError
		status.Message = syncErr.Error()
		return status
	}
	if hasSync {
		status.LastSyncedAt = syncedAt
	}
	if err := b.cachedGitOriginError(source, sourcePath); err != nil {
		status.Status = SourceStatusError
		status.Message = err.Error()
		return status
	}
	cachePath, hasCatalog := b.cachedCatalogPath(source, sourcePath)
	if !hasCatalog {
		status.Status = SourceStatusMissing
		status.Message = "cache missing. Run: skillhub sources sync " + source.Name
		return status
	}
	status.CachePath = cachePath
	if !hasSync || !b.sourceSyncTimeIsFresh(syncedAt) {
		status.Status = SourceStatusStale
		status.Message = "cache is stale. Run: skillhub sources sync " + source.Name
		return status
	}
	status.Status = SourceStatusFresh
	status.Message = "cache is fresh"
	return status
}

func (b *Backend) cachedCatalogPath(source Source, sourcePath string) (string, bool) {
	if _, err := os.Stat(filepath.Join(sourcePath, source.Catalog)); err == nil {
		return sourcePath, true
	}
	return b.generatedCatalogPath(source.Name, source.Catalog)
}

func (b *Backend) generatedCatalogPath(sourceName string, sourceCatalog string) (string, bool) {
	generatedPath, err := b.generatedSourcePath(sourceName)
	if err != nil {
		return "", false
	}
	if _, err := os.Stat(filepath.Join(generatedPath, sourceCatalog)); err == nil {
		return generatedPath, true
	}
	return "", false
}

func isLocalSourceOverride(source Source, override string) bool {
	return source.Name == SourceNameAgentRules && strings.TrimSpace(override) != ""
}

func validateSources(rows []Source) ([]Source, error) {
	seen := map[string]bool{}
	for _, source := range rows {
		if !isValidID(source.Name) {
			return nil, fmt.Errorf("invalid source name: %s", source.Name)
		}
		if source.Type != SourceTypeGit && source.Type != SourceTypePath {
			return nil, fmt.Errorf("unsupported source type for %s: %s", source.Name, source.Type)
		}
		if source.Location == "" || source.Ref == "" || source.Catalog == "" {
			return nil, fmt.Errorf("source row has empty fields for %s", source.Name)
		}
		if seen[source.Name] {
			return nil, fmt.Errorf("duplicate source: %s", source.Name)
		}
		seen[source.Name] = true
	}
	return rows, nil
}

func (b *Backend) AddSource(opts SourceAddOptions) (string, error) {
	source, err := b.sourceFromAddOptions(opts)
	if err != nil {
		return "", err
	}
	existing, err := b.ListSources()
	if err != nil {
		return "", err
	}
	for _, existingSource := range existing {
		if existingSource.Name == source.Name {
			return "", fmt.Errorf("source already exists: %s", source.Name)
		}
	}
	if _, _, err := b.SyncCatalogSource(source); err != nil {
		return "", err
	}
	userFile, err := b.UserSourcesFile()
	if err != nil {
		return "", err
	}
	if err := sourcesTable.WriteFile(userFile, append(existing, source)); err != nil {
		return "", err
	}
	return fmt.Sprintf("Added source %s to %s\n", source.Name, userFile), nil
}

func (b *Backend) sourceFromAddOptions(opts SourceAddOptions) (Source, error) {
	location, treeRef, err := normalizeSourceLocation(opts.Location)
	if err != nil {
		return Source{}, err
	}
	rawLocation := location
	pathLocation, err := b.AbsPath(location, b.callerCWD)
	if err != nil {
		return Source{}, err
	}
	sourceType := b.resolveSourceType(opts.Type, pathLocation)
	if sourceType != SourceTypeGit && sourceType != SourceTypePath {
		return Source{}, fmt.Errorf("unsupported source type: %s", sourceType)
	}
	name := sourceNameFromOptions(opts.Name, sourceType, pathLocation, rawLocation)
	if !isValidID(name) {
		return Source{}, fmt.Errorf("invalid source name: %s", name)
	}
	catalog := strings.TrimSpace(opts.Catalog)
	if catalog == "" {
		catalog = defaultCatalogPath
	}
	ref := strings.TrimSpace(opts.Ref)
	location, ref, err = b.finalizeSourceLocation(sourceType, location, pathLocation, catalog, name, ref, treeRef)
	if err != nil {
		return Source{}, err
	}
	return Source{Name: name, Type: sourceType, Location: location, Ref: ref, Catalog: catalog}, nil
}

func normalizeSourceLocation(location string) (sourceLocation string, treeRef string, err error) {
	location = strings.TrimSpace(location)
	if location == "" {
		return "", "", errors.New("source location is required")
	}
	tree, ok := parseGitHubTreeURL(location)
	if ok {
		return tree.repoURL, tree.ref, nil
	}
	return location, "", nil
}

func (b *Backend) resolveSourceType(optionType, pathLocation string) string {
	sourceType := strings.TrimSpace(optionType)
	if sourceType != "" {
		return sourceType
	}
	if info, err := os.Stat(pathLocation); err == nil && info.IsDir() {
		return SourceTypePath
	}
	return SourceTypeGit
}

func sourceNameFromOptions(optionName, sourceType, pathLocation, rawLocation string) string {
	name := strings.TrimSpace(optionName)
	if name != "" {
		return name
	}
	if sourceType == SourceTypePath {
		return deriveSourceName(pathLocation)
	}
	return deriveSourceName(rawLocation)
}

func (b *Backend) finalizeSourceLocation(
	sourceType string,
	location string,
	pathLocation string,
	catalog string,
	name string,
	ref string,
	treeRef string,
) (sourceLocation string, sourceRef string, err error) {
	if sourceType == SourceTypePath {
		return b.finalizePathSource(pathLocation, catalog, name, ref)
	}
	if ref == "" {
		if treeRef != "" {
			ref = treeRef
		} else {
			ref = "main"
		}
	}
	return location, ref, nil
}

func (b *Backend) finalizePathSource(
	pathLocation string,
	catalog string,
	name string,
	ref string,
) (sourceLocation string, sourceRef string, err error) {
	info, err := os.Stat(pathLocation)
	if err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("path source does not exist: %s", pathLocation)
	}
	if _, err := os.Stat(filepath.Join(pathLocation, catalog)); err != nil && !b.sourceHasDiscoverableSkills(pathLocation) {
		return "", "", fmt.Errorf("path source %s is missing catalog: %s", name, filepath.Join(pathLocation, catalog))
	}
	if ref == "" {
		ref = "-"
	}
	return pathLocation, ref, nil
}

func (b *Backend) AddDefaultSource(name string) (string, error) {
	defaults, err := b.ListDefaultSources()
	if err != nil {
		return "", err
	}
	var wanted Source
	for _, source := range defaults {
		if source.Name == name {
			wanted = source
			break
		}
	}
	if wanted.Name == "" {
		return "", fmt.Errorf("default source not found: %s", name)
	}
	existing, err := b.ListSources()
	if err != nil {
		return "", err
	}
	for _, source := range existing {
		if source.Name == name {
			return "", fmt.Errorf("source already exists: %s", name)
		}
	}
	if _, _, err := b.SyncCatalogSource(wanted); err != nil {
		return "", err
	}
	userFile, err := b.UserSourcesFile()
	if err != nil {
		return "", err
	}
	if err := sourcesTable.WriteFile(userFile, append(existing, wanted)); err != nil {
		return "", err
	}
	return fmt.Sprintf("Added default source %s to %s\n", name, userFile), nil
}

func (b *Backend) RemoveSource(name string) (string, error) {
	if !isValidID(name) {
		return "", fmt.Errorf("invalid source name: %s", name)
	}
	existing, err := b.ListSources()
	if err != nil {
		return "", err
	}
	next := make([]Source, 0, len(existing))
	found := false
	for _, source := range existing {
		if source.Name == name {
			found = true
			continue
		}
		next = append(next, source)
	}
	if !found {
		return "", fmt.Errorf("user source not found: %s", name)
	}
	userFile, err := b.UserSourcesFile()
	if err != nil {
		return "", err
	}
	if err := sourcesTable.WriteFile(userFile, next); err != nil {
		return "", err
	}
	if err := b.clearSourceCache(name); err != nil {
		return "", err
	}
	return fmt.Sprintf("Removed source %s from %s\n", name, userFile), nil
}

func (b *Backend) SyncSources(name string) (string, error) {
	sources, err := b.ListSources()
	if err != nil {
		return "", err
	}
	var out strings.Builder
	synced := 0
	for _, source := range sources {
		if name != "" && source.Name != name {
			continue
		}
		path, _, err := b.SyncCatalogSource(source)
		if err != nil {
			return out.String(), err
		}
		_, _ = fmt.Fprintf(&out, "Synced %s to %s\n", source.Name, path)
		synced++
	}
	if synced == 0 {
		if name != "" {
			return out.String(), fmt.Errorf("no source matched: %s", name)
		}
		return out.String(), errors.New("no sources configured. Run: skillhub sources defaults list")
	}
	return out.String(), nil
}

func (b *Backend) SourcePath(source Source) (string, error) {
	if isLocalSourceOverride(source, b.envValue("SKILLHUB_AGENT_RULES_PATH")) {
		return b.envValue("SKILLHUB_AGENT_RULES_PATH"), nil
	}
	switch source.Type {
	case SourceTypePath:
		if filepath.IsAbs(source.Location) {
			return source.Location, nil
		}
		return filepath.Join(b.repoRoot, source.Location), nil
	case SourceTypeGit:
		return b.cachedGitSourcePath(source.Name)
	default:
		return "", fmt.Errorf("unsupported source type for %s: %s", source.Name, source.Type)
	}
}

func (b *Backend) CatalogSource(source Source) (path, warning string, err error) {
	sourcePath, err := b.SourcePath(source)
	if err != nil {
		return "", "", err
	}
	if isLocalSourceOverride(source, b.envValue("SKILLHUB_AGENT_RULES_PATH")) {
		if info, err := os.Stat(sourcePath); err != nil || !info.IsDir() {
			return "", "", fmt.Errorf("local source override does not exist: %s", sourcePath)
		}
		return sourcePath, "", nil
	}
	switch source.Type {
	case SourceTypePath:
		if info, err := os.Stat(sourcePath); err != nil || !info.IsDir() {
			return "", "", fmt.Errorf("path source does not exist: %s", sourcePath)
		}
		path, err = b.materializeSource(source.Name, sourcePath, source.Catalog, true)
		return path, "", err
	case SourceTypeGit:
		if err := b.cachedGitOriginError(source, sourcePath); err != nil {
			return "", "", err
		}
		materialized, err := b.materializeSource(source.Name, sourcePath, source.Catalog, false)
		if err == nil {
			if _, statErr := os.Stat(filepath.Join(materialized, source.Catalog)); statErr == nil {
				if !b.sourceSyncIsFresh(source.Name) {
					warning := fmt.Sprintf(
						"Warning: using stale cache for source %s. Run: skillhub sources sync %s\n",
						source.Name,
						source.Name,
					)
					return materialized, warning, nil
				}
				return materialized, "", nil
			}
		}
		return "", "", fmt.Errorf(
			"%w\nGit source cache is missing for %s. Run: skillhub sources sync %s",
			err,
			source.Name,
			source.Name,
		)
	default:
		return "", "", fmt.Errorf("unsupported source type for %s: %s", source.Name, source.Type)
	}
}

func (b *Backend) SyncCatalogSource(source Source) (path, warning string, err error) {
	sourcePath, err := b.syncSource(source)
	if err != nil {
		return "", "", err
	}
	path, err = b.materializeSource(source.Name, sourcePath, source.Catalog, true)
	return path, "", err
}

func (b *Backend) syncSource(source Source) (string, error) {
	sourcePath, err := b.SourcePath(source)
	if err != nil {
		return "", err
	}
	if isLocalSourceOverride(source, b.envValue("SKILLHUB_AGENT_RULES_PATH")) {
		if info, err := os.Stat(sourcePath); err != nil || !info.IsDir() {
			return "", fmt.Errorf("local source override does not exist: %s", sourcePath)
		}
		return sourcePath, nil
	}
	switch source.Type {
	case SourceTypePath:
		if info, err := os.Stat(sourcePath); err != nil || !info.IsDir() {
			return "", fmt.Errorf("path source does not exist: %s", sourcePath)
		}
		return sourcePath, nil
	case SourceTypeGit:
		if err := b.syncGitSource(source, sourcePath); err != nil {
			return "", err
		}
		return sourcePath, nil
	default:
		return "", fmt.Errorf("unsupported source type for %s: %s", source.Name, source.Type)
	}
}

func (b *Backend) syncGitSource(source Source, sourcePath string) error {
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		return err
	}
	if err := b.ensureGitSourceOrigin(source, sourcePath); err != nil {
		return err
	}
	if err := b.fetchOrCloneGitSource(source, sourcePath); err != nil {
		return err
	}
	if _, err := b.gitOutput(sourcePath, "checkout", "--quiet", source.Ref); err != nil {
		return err
	}
	if b.gitWorktreeHasBranch(sourcePath) {
		if _, err := b.gitOutput(sourcePath, "pull", "--quiet", "--ff-only"); err != nil {
			return err
		}
	}
	return b.markSourceSynced(source.Name)
}

func (b *Backend) gitWorktreeHasBranch(sourcePath string) bool {
	branch, err := b.gitOutput(sourcePath, "symbolic-ref", "--quiet", "--short", "HEAD")
	return err == nil && strings.TrimSpace(branch) != ""
}

func (b *Backend) ensureGitSourceOrigin(source Source, sourcePath string) error {
	if !b.gitDirExists(sourcePath) {
		return nil
	}
	matches, _ := b.cachedGitOriginMatches(source, sourcePath)
	if matches {
		return nil
	}
	return b.removeCachedSourcePath(source.Name, sourcePath)
}

func (b *Backend) fetchOrCloneGitSource(source Source, sourcePath string) error {
	if b.gitDirExists(sourcePath) {
		_, err := b.gitOutput(sourcePath, "fetch", "--quiet", "--prune")
		return err
	}
	_, err := b.gitOutput(b.repoRoot, "clone", "--quiet", source.Location, sourcePath)
	return err
}

func (b *Backend) gitDirExists(sourcePath string) bool {
	info, err := os.Stat(filepath.Join(sourcePath, ".git"))
	return err == nil && info.IsDir()
}

func (b *Backend) removeCachedSourcePath(sourceName, sourcePath string) error {
	cacheRoot, err := b.CacheRoot()
	if err != nil {
		return err
	}
	cacheSourcesDir := filepath.Join(cacheRoot, "sources")
	if !strings.HasPrefix(sourcePath, cacheSourcesDir+string(os.PathSeparator)) {
		return fmt.Errorf("refusing to replace non-cache source path for %s: %s", sourceName, sourcePath)
	}
	return os.RemoveAll(sourcePath)
}

func (b *Backend) gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	cmd.Env = envSlice(b.env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("git %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func (b *Backend) markSourceSynced(name string) error {
	file, err := b.sourceSyncStateFile(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(fmt.Sprintf("%d\n", b.now().Unix())), 0o644)
}

func (b *Backend) sourceSyncIsFresh(name string) bool {
	syncedAt, ok, err := b.readSourceSyncedAt(name)
	if err != nil || !ok {
		return false
	}
	return b.sourceSyncTimeIsFresh(syncedAt)
}

func (b *Backend) readSourceSyncedAt(name string) (time.Time, bool, error) {
	file, err := b.sourceSyncStateFile(name)
	if err != nil {
		return time.Time{}, false, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	seconds, parseErr := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if parseErr != nil {
		return time.Time{}, false, fmt.Errorf("invalid source sync timestamp for %s: %w", name, parseErr)
	}
	return time.Unix(seconds, 0).UTC(), true, nil
}

func (b *Backend) sourceSyncTimeIsFresh(syncedAt time.Time) bool {
	if b.now().Before(syncedAt) {
		return true
	}
	return b.now().Sub(syncedAt) < sourceTTL
}

func (b *Backend) cachedGitOriginError(source Source, sourcePath string) error {
	if _, err := os.Stat(filepath.Join(sourcePath, ".git")); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	matches, err := b.cachedGitOriginMatches(source, sourcePath)
	if err != nil {
		return err
	}
	if matches {
		return nil
	}
	return fmt.Errorf(
		"cached git origin for %s does not match configured location. Run: skillhub sources sync %s",
		source.Name,
		source.Name,
	)
}

func (b *Backend) cachedGitOriginMatches(source Source, sourcePath string) (bool, error) {
	origin, err := b.gitOutput(sourcePath, "config", "--get", "remote.origin.url")
	if err != nil {
		return false, err
	}
	resolvedOrigin, err := b.resolvedGitURL(strings.TrimSpace(origin))
	if err != nil {
		return false, err
	}
	// The cache repo may have its own URL rewrites; those must not validate a mismatched origin.
	cacheResolvedOrigin, err := b.gitOutput(sourcePath, "remote", "get-url", "origin")
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(cacheResolvedOrigin) != resolvedOrigin {
		return false, nil
	}
	configured, err := b.resolvedGitURL(source.Location)
	if err != nil {
		return false, err
	}
	return resolvedOrigin == configured, nil
}

func (b *Backend) resolvedGitURL(rawURL string) (string, error) {
	resolved, err := b.gitOutput(b.repoRoot, "ls-remote", "--get-url", rawURL)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resolved), nil
}

func (b *Backend) clearSourceCache(name string) error {
	if root, err := b.CacheRoot(); err == nil {
		if err := os.RemoveAll(filepath.Join(root, "sources", name)); err != nil {
			return err
		}
		if err := os.RemoveAll(filepath.Join(root, "generated-sources", name)); err != nil {
			return err
		}
	}
	if state, err := b.sourceSyncStateFile(name); err == nil {
		if err := os.Remove(state); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (b *Backend) materializeSource(sourceName, sourcePath, sourceCatalog string, force bool) (string, error) {
	if _, err := os.Stat(filepath.Join(sourcePath, sourceCatalog)); err == nil {
		generatedPath, _ := b.generatedSourcePath(sourceName)
		_ = os.RemoveAll(generatedPath)
		return sourcePath, nil
	}
	generatedPath, err := b.generatedSourcePath(sourceName)
	if err != nil {
		return "", err
	}
	if !force {
		if _, err := os.Stat(filepath.Join(generatedPath, sourceCatalog)); err == nil {
			return generatedPath, nil
		}
	}
	if !b.sourceHasDiscoverableSkills(sourcePath) {
		return "", missingSourceCatalogError(sourceName, sourcePath, sourceCatalog)
	}
	tmpPath := fmt.Sprintf("%s.%d", generatedPath, os.Getpid())
	if err := os.RemoveAll(tmpPath); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(tmpPath, "catalog"), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(tmpPath, "skills"), 0o755); err != nil {
		return "", err
	}
	skillFiles, err := discoverSkillFiles(sourcePath)
	if err != nil {
		return "", err
	}
	sort.Strings(skillFiles)
	catalogRows, err := buildGeneratedCatalogRows(sourceName, sourcePath, tmpPath, skillFiles)
	if err != nil {
		return "", err
	}
	return replaceGeneratedSource(tmpPath, generatedPath, sourceCatalog, catalogRows)
}

func missingSourceCatalogError(sourceName, sourcePath, sourceCatalog string) error {
	return fmt.Errorf(
		"source %s is missing catalog: %s and has no discoverable skills under %s/skills or %s/*/SKILL.md",
		sourceName,
		filepath.Join(sourcePath, sourceCatalog),
		sourcePath,
		sourcePath,
	)
}

func buildGeneratedCatalogRows(sourceName, sourcePath, tmpPath string, skillFiles []string) ([]CatalogRow, error) {
	seen := map[string]string{}
	rows := []CatalogRow{}
	for _, skillFile := range skillFiles {
		row, ok, err := generatedCatalogRow(sourceName, sourcePath, tmpPath, skillFile, seen)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func generatedCatalogRow(
	sourceName string,
	sourcePath string,
	tmpPath string,
	skillFile string,
	seen map[string]string,
) (CatalogRow, bool, error) {
	skillDir := filepath.Dir(skillFile)
	rel := generatedSkillRelativeDir(sourcePath, skillDir)
	if rel == "." || hasHiddenPathSegment(rel) {
		return CatalogRow{}, false, nil
	}
	flatName := strings.ReplaceAll(rel, "/", "_")
	if !isValidID(flatName) {
		return CatalogRow{}, false, fmt.Errorf("invalid generated skill name from %s: %s", rel, flatName)
	}
	if previous, ok := seen[flatName]; ok {
		return CatalogRow{}, false, fmt.Errorf(
			"duplicate generated skill name %s in source %s: %s and %s", flatName, sourceName, previous, rel,
		)
	}
	seen[flatName] = rel
	if err := copyDir(skillDir, filepath.Join(tmpPath, "skills", flatName)); err != nil {
		return CatalogRow{}, false, err
	}
	return CatalogRow{
		Name:        flatName,
		Category:    generatedSkillCategory(rel),
		Triggers:    generatedSkillTriggers(skillFile, rel),
		Description: generatedSkillDescription(skillFile, rel),
	}, true, nil
}

func generatedSkillRelativeDir(sourcePath, skillDir string) string {
	skillsRoot := filepath.Join(sourcePath, "skills")
	var rel string
	if strings.HasPrefix(skillDir, skillsRoot+string(os.PathSeparator)) {
		rel, _ = filepath.Rel(skillsRoot, skillDir)
	} else {
		rel, _ = filepath.Rel(sourcePath, skillDir)
	}
	return filepath.ToSlash(rel)
}

func generatedSkillCategory(rel string) string {
	if idx := strings.Index(rel, "/"); idx >= 0 {
		return rel[:idx]
	}
	return rel
}

func generatedSkillTriggers(skillFile, rel string) string {
	triggers := strings.ReplaceAll(rel, "/", ",")
	fmName := skillFrontmatterValue(skillFile, "name")
	if fmName != "" && !commaListContains(triggers, fmName) {
		triggers += "," + fmName
	}
	return triggers
}

func generatedSkillDescription(skillFile, rel string) string {
	description := skillFrontmatterValue(skillFile, "description")
	if description == "" {
		return "Skill from " + rel
	}
	return description
}

func replaceGeneratedSource(tmpPath, generatedPath, sourceCatalog string, catalogRows []CatalogRow) (string, error) {
	if err := CatalogTable.WriteFile(filepath.Join(tmpPath, sourceCatalog), catalogRows); err != nil {
		return "", err
	}
	if err := os.RemoveAll(generatedPath); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, generatedPath); err != nil {
		return "", err
	}
	return generatedPath, nil
}

func (b *Backend) sourceHasDiscoverableSkills(sourcePath string) bool {
	files, err := discoverSkillFiles(sourcePath)
	return err == nil && len(files) > 0
}

func discoverSkillFiles(sourcePath string) ([]string, error) {
	files := []string{}
	skillsRoot := filepath.Join(sourcePath, "skills")
	if info, err := os.Stat(skillsRoot); err == nil && info.IsDir() {
		if err := filepath.WalkDir(skillsRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if d.Name() == "SKILL.md" {
				files = append(files, path)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	entries, err := os.ReadDir(sourcePath)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == "skills" {
				continue
			}
			candidate := filepath.Join(sourcePath, entry.Name(), "SKILL.md")
			if _, err := os.Stat(candidate); err == nil {
				files = append(files, candidate)
			}
		}
	}
	sort.Strings(files)
	dedup := files[:0]
	seen := map[string]bool{}
	for _, file := range files {
		if !seen[file] {
			seen[file] = true
			dedup = append(dedup, file)
		}
	}
	return dedup, nil
}

func skillFrontmatterValue(skillFile, wanted string) string {
	file, err := os.Open(skillFile)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return ""
	}
	first := scanner.Text()
	if strings.HasPrefix(first, "--- ") {
		key := wanted + ":"
		if idx := strings.Index(first, key); idx >= 0 {
			value := strings.TrimSpace(first[idx+len(key):])
			if end := strings.LastIndex(value, "---"); end >= 0 {
				value = strings.TrimSpace(value[:end])
			}
			return value
		}
	}
	if first != "---" {
		return ""
	}
	prefix := wanted + ":"
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			return ""
		}
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func commaListContains(list, item string) bool {
	for _, part := range strings.Split(list, ",") {
		if strings.TrimSpace(part) == item {
			return true
		}
	}
	return false
}

func hasHiddenPathSegment(rel string) bool {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

func deriveSourceName(location string) string {
	clean := strings.TrimRight(location, "/")
	base := filepath.Base(clean)
	return strings.TrimSuffix(base, ".git")
}

type githubTreeURL struct {
	repoURL string
	ref     string
	path    string
}

func parseGitHubTreeURL(input string) (githubTreeURL, bool) {
	clean := strings.TrimRight(strings.Split(strings.Split(input, "#")[0], "?")[0], "/")
	var scheme string
	switch {
	case strings.HasPrefix(clean, "https://github.com/"):
		scheme = "https"
	case strings.HasPrefix(clean, "http://github.com/"):
		scheme = "http"
	default:
		return githubTreeURL{}, false
	}
	path := strings.TrimPrefix(strings.TrimPrefix(clean, "https://github.com/"), "http://github.com/")
	parts := strings.Split(path, "/")
	if len(parts) < 4 || parts[2] != "tree" {
		return githubTreeURL{}, false
	}
	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")
	ref := parts[3]
	if owner == "" || repo == "" || ref == "" {
		return githubTreeURL{}, false
	}
	return githubTreeURL{
		repoURL: fmt.Sprintf("%s://github.com/%s/%s", scheme, owner, repo),
		ref:     ref,
		path:    strings.Join(parts[4:], "/"),
	}, true
}

func envSlice(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for key, value := range env {
		out = append(out, key+"="+value)
	}
	sort.Strings(out)
	return out
}

func copyDir(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
