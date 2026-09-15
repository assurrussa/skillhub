package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type sourceRenameSnapshot struct {
	path   string
	data   []byte
	mode   os.FileMode
	exists bool
}

type sourceRenameWrite struct {
	path string
	data []byte
	mode os.FileMode
}

type sourceRenameMove struct {
	oldPath string
	newPath string
}

type sourceSkillRename struct {
	OldSkill string
	NewSkill string
	Enabled  bool
}

//nolint:gocognit,gocyclo // Keeping transaction preparation and rollback order together makes failure handling auditable.
func (b *Backend) renameSourceSafely(opts SourceRenameOptions) (SourceRenameSummary, error) {
	oldName := normalizeSourceName(opts.OldName)
	newName := normalizeSourceName(opts.NewName)
	if !isValidID(oldName) {
		return SourceRenameSummary{}, fmt.Errorf("invalid source name: %s", oldName)
	}
	if !isValidID(newName) {
		return SourceRenameSummary{}, fmt.Errorf("invalid new source name: %s", newName)
	}
	if oldName == newName {
		return SourceRenameSummary{}, fmt.Errorf("new source name must be different from current name: %s", oldName)
	}

	sources, err := b.ListSources()
	if err != nil {
		return SourceRenameSummary{}, err
	}
	foundIndex := -1
	for i, source := range sources {
		if source.Name == oldName {
			foundIndex = i
		}
		if source.Name == newName {
			return SourceRenameSummary{}, fmt.Errorf("source already exists: %s", newName)
		}
	}
	if foundIndex == -1 {
		return SourceRenameSummary{}, fmt.Errorf("user source not found: %s", oldName)
	}

	skillRename, err := b.aliasDerivedRootSkillRename(sources[foundIndex], oldName, newName)
	if err != nil {
		return SourceRenameSummary{}, err
	}

	// Read every durable dependency before mutating any file. A malformed
	// installed registry must not leave sources.tsv renamed while dependencies
	// still point at the old source.
	usageFile, err := b.InstalledRegistryFile()
	if err != nil {
		return SourceRenameSummary{}, err
	}
	usageRows := []InstalledSkill{}
	usageExists := false
	if _, statErr := os.Stat(usageFile); statErr == nil {
		usageExists = true
		usageRows, err = installedUsageTable.ReadFile(usageFile)
		if err != nil {
			return SourceRenameSummary{}, err
		}
	} else if !os.IsNotExist(statErr) {
		return SourceRenameSummary{}, statErr
	}

	nextSources := append([]Source(nil), sources...)
	nextSources[foundIndex].Name = newName
	nextUsage := append([]InstalledSkill(nil), usageRows...)
	affectedProjects := map[string]bool{}
	installedUpdated := 0
	for i := range nextUsage {
		if nextUsage[i].Source != oldName {
			continue
		}
		nextUsage[i].Source = newName
		if skillRename.Enabled && nextUsage[i].Skill == skillRename.OldSkill {
			nextUsage[i].Skill = skillRename.NewSkill
			nextUsage[i].InstalledPath, err = renamedInstalledSkillPath(
				nextUsage[i].InstalledPath,
				skillRename.OldSkill,
				skillRename.NewSkill,
			)
			if err != nil {
				return SourceRenameSummary{}, err
			}
			nextUsage[i].Path = nextUsage[i].InstalledPath
		}
		nextUsage[i].QualifiedSkill = newName + "/" + nextUsage[i].Skill
		installedUpdated++
		if nextUsage[i].Scope == ScopeProject && validProjectPath(nextUsage[i].ProjectPath) {
			affectedProjects[filepath.Clean(nextUsage[i].ProjectPath)] = true
		}
	}

	moves, err := prepareSourceRenameMoves(usageRows, nextUsage, oldName)
	if err != nil {
		return SourceRenameSummary{}, err
	}

	userFile, err := b.UserSourcesFile()
	if err != nil {
		return SourceRenameSummary{}, err
	}

	snapshots := map[string]sourceRenameSnapshot{}
	capture := func(path string) (sourceRenameSnapshot, error) {
		path = filepath.Clean(path)
		if snapshot, ok := snapshots[path]; ok {
			return snapshot, nil
		}
		snapshot, captureErr := captureSourceRenameSnapshot(path)
		if captureErr != nil {
			return sourceRenameSnapshot{}, captureErr
		}
		snapshots[path] = snapshot
		return snapshot, nil
	}
	if _, err := capture(userFile); err != nil {
		return SourceRenameSummary{}, err
	}
	if usageExists && installedUpdated > 0 {
		if _, err := capture(usageFile); err != nil {
			return SourceRenameSummary{}, err
		}
	}

	writes := []sourceRenameWrite{}
	for i, row := range usageRows {
		if row.Source != oldName || row.Scope == ScopeProject {
			continue
		}
		info, statErr := os.Stat(row.InstalledPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				// Registry-only stale installs are valid state. Rename the registry
				// reference, but do not recreate metadata inside a missing install.
				continue
			}
			return SourceRenameSummary{}, statErr
		}
		if !info.IsDir() {
			return SourceRenameSummary{}, fmt.Errorf("managed install path is not a directory: %s", row.InstalledPath)
		}
		sidecarPath := filepath.Join(row.InstalledPath, ".skillhub.json")
		snapshot, err := capture(sidecarPath)
		if err != nil {
			return SourceRenameSummary{}, err
		}
		nextRow := nextUsage[i]
		data, err := renamedSourceMetadata(snapshot, nextRow, newName)
		if err != nil {
			return SourceRenameSummary{}, err
		}
		writes = append(writes, sourceRenameWrite{
			path: filepath.Join(nextRow.InstalledPath, ".skillhub.json"),
			data: data,
			mode: 0o644,
		})
	}

	projectPaths := map[string]bool{}
	for projectPath := range affectedProjects {
		projectPaths[projectPath] = true
	}
	if validProjectPath(b.callerCWD) {
		projectPaths[filepath.Clean(b.callerCWD)] = true
	}
	lockfilesUpdated := 0
	for projectPath := range projectPaths {
		lockPath := filepath.Join(projectPath, "skills.lock.toml")
		snapshot, err := capture(lockPath)
		if err != nil {
			return SourceRenameSummary{}, err
		}
		if !snapshot.exists {
			continue
		}
		data, changed := rewriteLockfileSource(snapshot.data, oldName, newName, skillRename)
		if !changed {
			continue
		}
		writes = append(writes, sourceRenameWrite{path: lockPath, data: data, mode: snapshot.mode})
		lockfilesUpdated++
	}

	// Apply dependency files first and sources.tsv last. On any ordinary write
	// or directory-move failure, restore captured files and move installations
	// back so a retry starts from the old source/skill identity.
	if usageExists && installedUpdated > 0 {
		if err := installedUsageTable.WriteFile(usageFile, nextUsage); err != nil {
			return SourceRenameSummary{}, rollbackSourceRename(err, snapshots)
		}
	}
	performedMoves := make([]sourceRenameMove, 0, len(moves))
	for _, move := range moves {
		if err := os.Rename(move.oldPath, move.newPath); err != nil {
			return SourceRenameSummary{}, rollbackSourceRename(err, snapshots, performedMoves...)
		}
		performedMoves = append(performedMoves, move)
	}
	for _, write := range writes {
		if err := os.WriteFile(write.path, write.data, write.mode); err != nil {
			return SourceRenameSummary{}, rollbackSourceRename(err, snapshots, performedMoves...)
		}
	}
	if err := sourcesTable.WriteFile(userFile, nextSources); err != nil {
		return SourceRenameSummary{}, rollbackSourceRename(err, snapshots, performedMoves...)
	}

	// Cache state is derived and disposable. Migrate it after durable metadata
	// succeeds; any cache problem is healed by the next sources sync.
	b.migrateSourceCache(oldName, newName)

	out := fmt.Sprintf(
		"Renamed source %s to %s (updated %d installed skill(s), %d project lockfile(s))\n",
		oldName,
		newName,
		installedUpdated,
		lockfilesUpdated,
	)
	summary := SourceRenameSummary{
		OldName:          oldName,
		NewName:          newName,
		InstalledUpdated: installedUpdated,
		LockfilesUpdated: lockfilesUpdated,
		Output:           out,
	}
	if skillRename.Enabled {
		summary.RenamedSkillOld = skillRename.OldSkill
		summary.RenamedSkillNew = skillRename.NewSkill
	}
	return summary, nil
}

func (b *Backend) aliasDerivedRootSkillRename(source Source, oldName, newName string) (sourceSkillRename, error) {
	sourcePath, err := b.SourcePath(source)
	if err != nil {
		return sourceSkillRename{}, err
	}
	// A missing checkout is not evidence that a source has no root skill.
	// Inspect it before preparing any registry, lockfile, or directory changes.
	if _, err := os.ReadDir(sourcePath); err != nil {
		return sourceSkillRename{}, fmt.Errorf(
			"cannot inspect source %s before rename: %w; restore its path or run skillhub sources sync %s first",
			oldName, err, oldName,
		)
	}
	if source.Type == SourceTypeGit {
		if err := b.cachedGitOriginError(source, sourcePath); err != nil {
			return sourceSkillRename{}, err
		}
	}
	if info, err := os.Stat(filepath.Join(sourcePath, source.Catalog)); err == nil {
		if !info.Mode().IsRegular() {
			return sourceSkillRename{}, fmt.Errorf("source catalog is not a regular file: %s", source.Catalog)
		}
		// Native catalogs own their skill IDs; source aliases do not affect them.
		return sourceSkillRename{}, nil
	} else if !os.IsNotExist(err) {
		return sourceSkillRename{}, err
	}

	rootSkillFile := filepath.Join(sourcePath, "SKILL.md")
	info, err := os.Lstat(rootSkillFile)
	if err != nil {
		if os.IsNotExist(err) {
			return sourceSkillRename{}, nil
		}
		return sourceSkillRename{}, err
	}
	if info.IsDir() {
		return sourceSkillRename{}, nil
	}
	// Do not interpret unreadable frontmatter (including a dangling symlink)
	// as an absent name and accidentally migrate an unrelated skill ID.
	if _, err := os.ReadFile(rootSkillFile); err != nil {
		return sourceSkillRename{}, fmt.Errorf("cannot inspect root skill before rename: %w", err)
	}
	oldSkill := rootGeneratedSkillName(oldName, rootSkillFile)
	newSkill := rootGeneratedSkillName(newName, rootSkillFile)
	if oldSkill == newSkill {
		return sourceSkillRename{}, nil
	}
	skillFiles, err := discoverSkillFiles(sourcePath)
	if err != nil {
		return sourceSkillRename{}, err
	}
	if err := validateGeneratedSkillRename(oldName, newName, sourcePath, skillFiles); err != nil {
		return sourceSkillRename{}, fmt.Errorf("cannot rename source %s to %s: %w", oldName, newName, err)
	}
	return sourceSkillRename{OldSkill: oldSkill, NewSkill: newSkill, Enabled: true}, nil
}

func validateGeneratedSkillRename(oldName, newName, sourcePath string, skillFiles []string) error {
	oldPlans, err := planGeneratedSkills(oldName, sourcePath, skillFiles)
	if err != nil {
		return err
	}
	newPlans, err := planGeneratedSkills(newName, sourcePath, skillFiles)
	if err != nil {
		return err
	}
	oldIDs := make(map[string]string, len(oldPlans))
	for _, plan := range oldPlans {
		oldIDs[plan.file] = plan.name
	}
	for _, plan := range newPlans {
		if plan.root || oldIDs[plan.file] == plan.name {
			continue
		}
		return fmt.Errorf(
			"renaming the root skill would change %s from %s to %s",
			plan.rel,
			oldIDs[plan.file],
			plan.name,
		)
	}
	return nil
}

func renamedInstalledSkillPath(path, oldSkill, newSkill string) (string, error) {
	path = filepath.Clean(path)
	if filepath.Base(path) != oldSkill {
		return "", fmt.Errorf(
			"cannot migrate renamed root skill %s: installed path does not end with skill id: %s",
			oldSkill,
			path,
		)
	}
	return filepath.Join(filepath.Dir(path), newSkill), nil
}

func prepareSourceRenameMoves(before, after []InstalledSkill, oldSource string) ([]sourceRenameMove, error) {
	moves := []sourceRenameMove{}
	seen := map[string]bool{}
	for i, oldRow := range before {
		if oldRow.Source != oldSource || i >= len(after) {
			continue
		}
		newRow := after[i]
		oldPath := filepath.Clean(oldRow.InstalledPath)
		newPath := filepath.Clean(newRow.InstalledPath)
		if oldPath == newPath || seen[oldPath] {
			continue
		}
		seen[oldPath] = true
		if _, err := os.Lstat(newPath); err == nil {
			return nil, fmt.Errorf("cannot rename installed skill: destination already exists: %s", newPath)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		info, err := os.Stat(oldPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("managed install path is not a directory: %s", oldPath)
		}
		moves = append(moves, sourceRenameMove{oldPath: oldPath, newPath: newPath})
	}
	return moves, nil
}

func validProjectPath(path string) bool {
	path = strings.TrimSpace(path)
	return path != "" && path != "-"
}

func captureSourceRenameSnapshot(path string) (sourceRenameSnapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return sourceRenameSnapshot{path: path, mode: 0o644}, nil
		}
		return sourceRenameSnapshot{}, err
	}
	if info.IsDir() {
		return sourceRenameSnapshot{}, fmt.Errorf("cannot rename source metadata through directory: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return sourceRenameSnapshot{}, err
	}
	return sourceRenameSnapshot{path: path, data: data, mode: info.Mode().Perm(), exists: true}, nil
}

func renamedSourceMetadata(snapshot sourceRenameSnapshot, row InstalledSkill, newName string) ([]byte, error) {
	meta := metadataFile{}
	if snapshot.exists {
		if err := json.Unmarshal(snapshot.data, &meta); err != nil {
			return nil, fmt.Errorf("invalid install metadata %s: %w", snapshot.path, err)
		}
	} else {
		meta = metadataFile{
			SchemaVersion:  "2",
			Target:         row.Target,
			Scope:          row.Scope,
			ProjectPath:    row.ProjectPath,
			TargetRoot:     row.TargetRoot,
			SourceRef:      row.SourceRef,
			SourceLocation: row.SourceLocation,
			Catalog:        row.Catalog,
			ContentHash:    row.ContentHash,
			InstalledAt:    row.InstalledAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}
	meta.Source = newName
	meta.Skill = row.Skill
	meta.QualifiedSkill = newName + "/" + row.Skill
	meta.InstalledPath = row.InstalledPath
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

//nolint:gocognit // The line-preserving lockfile rewrite deliberately handles every field in one block state machine.
func rewriteLockfileSource(data []byte, oldName, newName string, skillRenames ...sourceSkillRename) ([]byte, bool) {
	lines := strings.Split(string(data), "\n")
	rename := sourceSkillRename{}
	if len(skillRenames) > 0 {
		rename = skillRenames[0]
	}
	changed := false
	blockStart := -1
	processBlock := func(start, end int) {
		if start < 0 || start >= end {
			return
		}
		oldSourceLine := fmt.Sprintf("source = %q", oldName)
		oldSkillLine := fmt.Sprintf("skill = %q", rename.OldSkill)
		sourceMatches := false
		skillMatches := false
		for i := start; i < end; i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == oldSourceLine {
				sourceMatches = true
			}
			if rename.Enabled && trimmed == oldSkillLine {
				skillMatches = true
			}
		}
		if !sourceMatches {
			return
		}
		for i := start; i < end; i++ {
			line := lines[i]
			trimmed := strings.TrimSpace(line)
			switch {
			case trimmed == oldSourceLine:
				lines[i] = strings.Replace(line, oldSourceLine, fmt.Sprintf("source = %q", newName), 1)
				changed = true
			case rename.Enabled && skillMatches && trimmed == oldSkillLine:
				lines[i] = strings.Replace(line, oldSkillLine, fmt.Sprintf("skill = %q", rename.NewSkill), 1)
				changed = true
			case strings.HasPrefix(trimmed, "qualified_skill = "):
				oldQualified := oldName + "/"
				newQualified := newName + "/"
				if rename.Enabled && skillMatches {
					oldQualified = oldName + "/" + rename.OldSkill
					newQualified = newName + "/" + rename.NewSkill
				}
				oldQuoted := fmt.Sprintf("qualified_skill = %q", oldQualified)
				if rename.Enabled && skillMatches && trimmed == oldQuoted {
					lines[i] = strings.Replace(line, oldQuoted, fmt.Sprintf("qualified_skill = %q", newQualified), 1)
					changed = true
				} else if strings.Contains(line, `"`+oldName+`/`) {
					lines[i] = strings.Replace(line, `"`+oldName+`/`, `"`+newName+`/`, 1)
					changed = true
				}
			case rename.Enabled && skillMatches && strings.HasPrefix(trimmed, "installed_path = "):
				if rewritten, ok := rewriteLockfileInstalledPath(line, rename.OldSkill, rename.NewSkill); ok {
					lines[i] = rewritten
					changed = true
				}
			}
		}
	}
	for i, line := range lines {
		if strings.TrimSpace(line) != "[[skills]]" {
			continue
		}
		if blockStart >= 0 {
			processBlock(blockStart, i)
		}
		blockStart = i + 1
	}
	if blockStart >= 0 {
		processBlock(blockStart, len(lines))
	}
	if !changed {
		return data, false
	}
	return []byte(strings.Join(lines, "\n")), true
}

func rewriteLockfileInstalledPath(line, oldSkill, newSkill string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	const prefix = `installed_path = "`
	if !strings.HasPrefix(trimmed, prefix) || !strings.HasSuffix(trimmed, `"`) {
		return line, false
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(trimmed, prefix), `"`)
	value := tomlUnescape(raw)
	clean := filepath.Clean(filepath.FromSlash(value))
	if filepath.Base(clean) != oldSkill {
		return line, false
	}
	newValue := filepath.ToSlash(filepath.Join(filepath.Dir(clean), newSkill))
	if filepath.Dir(clean) == "." {
		newValue = newSkill
	}
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	return indent + `installed_path = "` + tomlEscape(newValue) + `"`, true
}

func rollbackSourceRename(cause error, snapshots map[string]sourceRenameSnapshot, moves ...sourceRenameMove) error {
	var rollbackErr error
	for i := len(moves) - 1; i >= 0; i-- {
		move := moves[i]
		if err := os.Rename(move.newPath, move.oldPath); err != nil && rollbackErr == nil {
			rollbackErr = err
		}
	}
	for _, snapshot := range snapshots {
		var err error
		if snapshot.exists {
			err = os.WriteFile(snapshot.path, snapshot.data, snapshot.mode)
		} else {
			err = os.Remove(snapshot.path)
			if os.IsNotExist(err) {
				err = nil
			}
		}
		if err != nil && rollbackErr == nil {
			rollbackErr = err
		}
	}
	if rollbackErr != nil {
		return fmt.Errorf("source rename failed: %w; rollback also failed: %w", cause, rollbackErr)
	}
	return cause
}
