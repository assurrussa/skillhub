package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type metadataFile struct {
	SchemaVersion  string `json:"schema_version"`
	Source         string `json:"source"`
	Skill          string `json:"skill"`
	QualifiedSkill string `json:"qualified_skill"`
	Target         string `json:"target"`
	Scope          string `json:"scope"`
	ProjectPath    string `json:"project_path"`
	TargetRoot     string `json:"target_root"`
	InstalledPath  string `json:"installed_path"`
	SourceRef      string `json:"source_ref"`
	SourceLocation string `json:"source_location"`
	Catalog        string `json:"catalog"`
	ContentHash    string `json:"content_hash"`
	InstalledAt    string `json:"installed_at"`
	UpdatedAt      string `json:"updated_at"`
}

var installedUsageTable = Table[InstalledSkill]{
	Header:  InstalledUsageHeader,
	Columns: 13,
	Parse: func(row []string) (InstalledSkill, error) {
		return InstalledSkill{
			Source: row[0], Skill: row[1], Target: row[2], Scope: row[3],
			ProjectPath: row[4], TargetRoot: row[5], InstalledPath: row[6],
			SourceRef: row[7], SourceLocation: row[8], Catalog: row[9],
			ContentHash: row[10], InstalledAt: row[11], UpdatedAt: row[12],
			QualifiedSkill: row[0] + "/" + row[1], Managed: ManagedYes, Path: row[6],
			RegistryOnly: true,
		}, nil
	},
	Format: func(row InstalledSkill) []string {
		return []string{
			row.Source, row.Skill, row.Target, row.Scope, row.ProjectPath, row.TargetRoot,
			row.InstalledPath, row.SourceRef, row.SourceLocation, row.Catalog,
			row.ContentHash, row.InstalledAt, row.UpdatedAt,
		}
	},
}

func (b *Backend) ReadUsage(filter string) ([]InstalledSkill, error) {
	file, err := b.InstalledRegistryFile()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return []InstalledSkill{}, nil
	}
	rows, err := installedUsageTable.ReadFile(file)
	if err != nil {
		return nil, err
	}
	if filter == "" {
		return rows, nil
	}
	out := []InstalledSkill{}
	for _, row := range rows {
		if row.Skill != filter && row.QualifiedSkill != filter {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func (b *Backend) ensureInstalledRegistry() (string, error) {
	file, err := b.InstalledRegistryFile()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		if err := installedUsageTable.WriteFile(file, nil); err != nil {
			return "", err
		}
		return file, nil
	}
	if _, err := installedUsageTable.ReadFile(file); err != nil {
		return "", err
	}
	return file, nil
}

func (b *Backend) upsertUsage(row InstalledSkill) error {
	file, err := b.ensureInstalledRegistry()
	if err != nil {
		return err
	}
	existing, err := b.ReadUsage("")
	if err != nil {
		return err
	}
	rows := []InstalledSkill{}
	for _, current := range existing {
		if current.InstalledPath == row.InstalledPath {
			continue
		}
		rows = append(rows, current)
	}
	rows = append(rows, row)
	return installedUsageTable.WriteFile(file, rows)
}

func (b *Backend) removeUsageByPath(installedPath string) error {
	file, err := b.InstalledRegistryFile()
	if err != nil {
		return err
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}
	existing, err := b.ReadUsage("")
	if err != nil {
		return err
	}
	rows := []InstalledSkill{}
	for _, current := range existing {
		if current.InstalledPath != installedPath {
			rows = append(rows, current)
		}
	}
	return installedUsageTable.WriteFile(file, rows)
}

func (b *Backend) usageRowByPath(installedPath string) (InstalledSkill, bool) {
	rows, err := b.ReadUsage("")
	if err != nil {
		return InstalledSkill{}, false
	}
	for _, row := range rows {
		if row.InstalledPath == installedPath {
			return row, true
		}
	}
	return InstalledSkill{}, false
}

func (b *Backend) ListInstalled(opts InstalledListOptions) ([]InstalledSkill, error) {
	target, scope, project, dir := normalizeTargetFlags(opts.Target, opts.Scope, opts.Project, opts.Dir)
	root, err := b.TargetRoot(TargetRootOptions{
		Target:      target,
		Scope:       scope,
		Project:     project,
		Dir:         dir,
		ScopeWasSet: opts.Scope != "",
	})
	if err != nil {
		return nil, err
	}
	metadataScope := scope
	if target == TargetDirectory {
		metadataScope = ScopeCustom
	}
	rows := []InstalledSkill{}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return rows, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		skillDir := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
			continue
		}
		row := InstalledSkill{
			Target: target, Scope: metadataScope, Skill: entry.Name(), Managed: ManagedNo,
			Source: "-", QualifiedSkill: "-", InstalledPath: skillDir, ContentHash: "-",
			InstalledAt: "-", Path: skillDir,
		}
		if metadataScope == ScopeProject {
			if usage, ok := b.usageRowByPath(skillDir); ok {
				row.Managed = ManagedYes
				row.Source = usage.Source
				row.Skill = entry.Name()
				row.QualifiedSkill = usage.Source + "/" + usage.Skill
				row.InstalledPath = usage.InstalledPath
				row.ContentHash = usage.ContentHash
				row.InstalledAt = usage.InstalledAt
				row.ProjectPath = usage.ProjectPath
				row.TargetRoot = usage.TargetRoot
				row.UpdatedAt = usage.UpdatedAt
				row.RegistryOnly = true
			}
			rows = append(rows, row)
			continue
		}
		meta, ok := readMetadata(filepath.Join(skillDir, ".skillhub.json"))
		if ok {
			row.Managed = ManagedYes
			row.Source = meta.Source
			row.Skill = entry.Name()
			row.QualifiedSkill = fallback(meta.QualifiedSkill, meta.Source+"/"+meta.Skill)
			row.InstalledPath = fallback(meta.InstalledPath, skillDir)
			row.ContentHash = fallback(meta.ContentHash, "-")
			row.InstalledAt = fallback(meta.InstalledAt, "-")
			row.ProjectPath = fallback(meta.ProjectPath, "-")
			row.TargetRoot = meta.TargetRoot
			row.UpdatedAt = meta.UpdatedAt
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Skill < rows[j].Skill })
	return rows, nil
}

func normalizeTargetFlags(target, scope, project, dir string) (normalizedTarget, normalizedScope, projectOut, dirOut string) {
	if target == "" {
		target = TargetCodex
	}
	if scope == "" {
		scope = ScopeGlobal
	}
	return target, scope, project, dir
}

func readMetadata(path string) (metadataFile, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return metadataFile{}, false
	}
	var meta metadataFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return metadataFile{}, false
	}
	return meta, true
}

func (b *Backend) writeInstallMetadata(
	installedPath string,
	row InstalledSkill,
	previousInstalledAt string,
	writeSidecar bool,
) error {
	now := b.now().UTC().Format("2006-01-02T15:04:05Z")
	installedAt := previousInstalledAt
	if installedAt == "" || installedAt == "-" {
		installedAt = now
	}
	updatedAt := now
	row.InstalledAt = installedAt
	row.UpdatedAt = updatedAt
	if writeSidecar {
		meta := metadataFile{
			SchemaVersion: "2", Source: row.Source, Skill: row.Skill,
			QualifiedSkill: row.Source + "/" + row.Skill, Target: row.Target, Scope: row.Scope,
			ProjectPath: row.ProjectPath, TargetRoot: row.TargetRoot, InstalledPath: installedPath,
			SourceRef: row.SourceRef, SourceLocation: row.SourceLocation, Catalog: row.Catalog,
			ContentHash: row.ContentHash, InstalledAt: installedAt, UpdatedAt: updatedAt,
		}
		data, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if err := os.WriteFile(filepath.Join(installedPath, ".skillhub.json"), data, 0o644); err != nil {
			return err
		}
	} else {
		_ = os.Remove(filepath.Join(installedPath, ".skillhub.json"))
	}
	row.InstalledPath = installedPath
	row.QualifiedSkill = row.Source + "/" + row.Skill
	return b.upsertUsage(row)
}

func hashSkillDir(skillDir string) (string, error) {
	info, statErr := os.Stat(skillDir)
	if os.IsNotExist(statErr) || (statErr == nil && !info.IsDir()) {
		return "-", nil
	}
	if statErr != nil {
		return "", statErr
	}
	files := []string{}
	if err := filepath.WalkDir(skillDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".skillhub.json" {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)
	hash := sha256.New()
	for _, file := range files {
		rel, err := filepath.Rel(skillDir, file)
		if err != nil {
			return "", err
		}
		_, _ = hash.Write([]byte("./" + filepath.ToSlash(rel) + "\n"))
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		_, _ = hash.Write(data)
		_, _ = hash.Write([]byte("\n"))
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func installUsesSidecar(scope string) bool {
	return scope != ScopeProject
}

func fallback(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

func (b *Backend) UpdateInstalled(opts InstalledUpdateOptions) (UpdateSummary, error) {
	if opts.AllSupported {
		return b.updateAllSupported(opts.Project, opts.Verbose)
	}
	target := opts.Target
	if target == "" {
		target = TargetCodex
	}
	scope := opts.Scope
	if scope == "" {
		scope = ScopeGlobal
	}
	root, err := b.TargetRoot(TargetRootOptions{
		Target: target, Scope: scope, Project: opts.Project, Dir: opts.Dir,
		ScopeWasSet: opts.Scope != "",
	})
	if err != nil {
		return UpdateSummary{}, err
	}
	metadataScope := scope
	if target == TargetDirectory {
		metadataScope = ScopeCustom
	}
	projectPath := "-"
	if metadataScope == ScopeProject {
		projectPath, err = b.ProjectDir(opts.Project)
		if err != nil {
			return UpdateSummary{}, err
		}
	}
	return b.updateTargetRoot(root, target, metadataScope, projectPath, opts.Verbose, true)
}

func (b *Backend) UpdateUsage(opts UsageUpdateOptions) (UpdateSummary, error) {
	if !opts.Projects {
		return UpdateSummary{}, errors.New("usage update currently requires --projects")
	}
	projectFilter := ""
	if opts.Project != "" {
		projectPath, err := b.ProjectDir(opts.Project)
		if err != nil {
			return UpdateSummary{}, err
		}
		projectFilter = projectPath
	}
	usage, err := b.ReadUsage("")
	if err != nil {
		return UpdateSummary{}, err
	}
	var out strings.Builder
	summary := UpdateSummary{}
	processed := 0
	for _, row := range usage {
		if row.Scope != ScopeProject {
			continue
		}
		if opts.Target != "" && row.Target != opts.Target {
			continue
		}
		if projectFilter != "" && row.ProjectPath != projectFilter {
			continue
		}
		if !usageFilterListMatches(row.Source, row.Skill, opts.Filters) {
			continue
		}
		processed++
		if _, err := os.Stat(filepath.Join(row.InstalledPath, "SKILL.md")); err != nil {
			_, _ = fmt.Fprintf(&out, "Skipped %s/project %s: managed install missing at %s\n", row.Target, row.Skill, row.InstalledPath)
			summary.Skipped++
			continue
		}
		result, line := b.updateManagedSkill(managedSkillUpdate{
			skillDir:    row.InstalledPath,
			targetRoot:  row.TargetRoot,
			target:      row.Target,
			scope:       row.Scope,
			projectPath: row.ProjectPath,
			sourceName:  row.Source,
			skillName:   row.Skill,
			installedAt: row.InstalledAt,
			verbose:     opts.Verbose,
		})
		if line != "" {
			_, _ = out.WriteString(line)
		}
		summary.addUpdateResult(result)
	}
	if processed == 0 {
		_, _ = out.WriteString("No managed project-scope skill usage matched registry filters.\n")
	}
	_, _ = fmt.Fprintf(&out,
		"Updated project usage: updated=%d unchanged=%d skipped=%d failed=%d\n",
		summary.Updated, summary.Unchanged, summary.Skipped, summary.Failed)
	summary.Output = out.String()
	if summary.Failed > 0 {
		return summary, errors.New("project usage update failed")
	}
	return summary, nil
}

func (b *Backend) updateAllSupported(project string, verbose bool) (UpdateSummary, error) {
	targets, err := b.ListTargets()
	if err != nil {
		return UpdateSummary{}, err
	}
	projectPath, _ := b.ProjectDir(project)
	var out strings.Builder
	total := UpdateSummary{}
	touched := false
	for _, target := range targets {
		if target.ID == TargetDirectory || target.Status != StatusSupported || target.Adapter != AdapterSkillDir {
			continue
		}
		for _, scope := range []string{ScopeGlobal, ScopeProject} {
			root, err := b.TargetRoot(TargetRootOptions{
				Target:      target.ID,
				Scope:       scope,
				Project:     project,
				ScopeWasSet: true,
			})
			if err != nil {
				return total, err
			}
			rowProject := "-"
			if scope == ScopeProject {
				rowProject = projectPath
			}
			if _, managed := b.countSkillDirs(root, scope); managed == 0 {
				if verbose {
					_, _ = fmt.Fprintf(&out, "No managed skills in %s/%s: %s\n", target.ID, scope, root)
				}
				continue
			}
			touched = true
			summary, err := b.updateTargetRoot(root, target.ID, scope, rowProject, verbose, false)
			_, _ = out.WriteString(summary.Output)
			total.Updated += summary.Updated
			total.Unchanged += summary.Unchanged
			total.Skipped += summary.Skipped
			total.Failed += summary.Failed
			if err != nil {
				total.Output = out.String()
				return total, err
			}
		}
	}
	if !touched {
		_, _ = out.WriteString("No managed installed skills found across supported targets.\n")
	}
	_, _ = fmt.Fprintf(&out,
		"Updated: %d, unchanged: %d, skipped: %d, failed: %d\n",
		total.Updated, total.Unchanged, total.Skipped, total.Failed)
	total.Output = out.String()
	if total.Failed > 0 {
		return total, errors.New("installed update failed")
	}
	return total, nil
}

func (b *Backend) updateTargetRoot(root, target, scope, projectPath string, verbose, printEmpty bool) (UpdateSummary, error) {
	var out strings.Builder
	summary := UpdateSummary{}
	managed := 0
	entries, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) {
		return summary, err
	}
	for _, entry := range entries {
		skillDir := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
			continue
		}
		var source, skill, installedAt string
		if scope == ScopeProject {
			usage, ok := b.usageRowByPath(skillDir)
			if !ok {
				continue
			}
			source, skill, installedAt = usage.Source, usage.Skill, usage.InstalledAt
			projectPath = usage.ProjectPath
		} else {
			meta, ok := readMetadata(filepath.Join(skillDir, ".skillhub.json"))
			if !ok {
				continue
			}
			source, skill, installedAt = meta.Source, meta.Skill, meta.InstalledAt
			if projectPath == "" {
				projectPath = meta.ProjectPath
			}
		}
		managed++
		result, line := b.updateManagedSkill(managedSkillUpdate{
			skillDir:    skillDir,
			targetRoot:  root,
			target:      target,
			scope:       scope,
			projectPath: projectPath,
			sourceName:  source,
			skillName:   skill,
			installedAt: installedAt,
			verbose:     verbose,
		})
		if line != "" {
			_, _ = out.WriteString(line)
		}
		summary.addUpdateResult(result)
	}
	if managed == 0 {
		if printEmpty {
			_, _ = fmt.Fprintf(&out, "No managed installed skills found in %s\n", root)
		} else if verbose {
			_, _ = fmt.Fprintf(&out, "No managed skills in %s/%s: %s\n", target, scope, root)
		}
	} else {
		_, _ = fmt.Fprintf(&out, "Summary %s/%s: updated=%d unchanged=%d skipped=%d failed=%d\n",
			target, scope, summary.Updated, summary.Unchanged, summary.Skipped, summary.Failed)
	}
	if printEmpty {
		_, _ = fmt.Fprintf(&out, "Updated: %d, unchanged: %d, skipped: %d, failed: %d\n",
			summary.Updated, summary.Unchanged, summary.Skipped, summary.Failed)
	}
	summary.Output = out.String()
	if summary.Failed > 0 {
		return summary, errors.New("installed update failed")
	}
	return summary, nil
}

type managedSkillUpdate struct {
	skillDir    string
	targetRoot  string
	target      string
	scope       string
	projectPath string
	sourceName  string
	skillName   string
	installedAt string
	verbose     bool
}

func (b *Backend) updateManagedSkill(req managedSkillUpdate) (status, output string) {
	skillDir := req.skillDir
	targetRoot := req.targetRoot
	target := req.target
	scope := req.scope
	projectPath := req.projectPath
	sourceName := req.sourceName
	skillName := req.skillName
	installedAt := req.installedAt
	verbose := req.verbose

	if sourceName == "" || !isValidID(sourceName) {
		return ResultSkipped, fmt.Sprintf(
			"Skipped %s/%s %s: invalid metadata source %s\n",
			target, scope, filepath.Base(skillDir), sourceName)
	}
	if skillName == "" {
		skillName = filepath.Base(skillDir)
	}
	if !isValidID(skillName) {
		return ResultSkipped, fmt.Sprintf(
			"Skipped %s/%s %s: invalid metadata skill %s\n",
			target, scope, filepath.Base(skillDir), skillName)
	}
	sources, err := b.ListSources()
	if err != nil {
		return ResultSkipped, fmt.Sprintf("Skipped %s/%s %s: %v\n", target, scope, skillName, err)
	}
	var source Source
	for _, candidate := range sources {
		if candidate.Name == sourceName {
			source = candidate
			break
		}
	}
	if source.Name == "" {
		return ResultSkipped, fmt.Sprintf("Skipped %s/%s %s: source not configured: %s\n", target, scope, skillName, sourceName)
	}
	var out strings.Builder
	if verbose {
		_, _ = fmt.Fprintf(&out, "Checking %s/%s %s from %s\n", target, scope, skillName, sourceName)
	}
	sourcePath, _, err := b.SyncCatalogSource(source)
	if err != nil {
		_, _ = fmt.Fprintf(&out, "Skipped %s/%s %s: %v\n", target, scope, skillName, err)
		return ResultSkipped, out.String()
	}
	catalogRows, err := CatalogTable.ReadFile(filepath.Join(sourcePath, source.Catalog))
	if err != nil {
		_, _ = fmt.Fprintf(&out,
			"Skipped %s/%s %s: source catalog missing: %s\n",
			target, scope, skillName, filepath.Join(sourcePath, source.Catalog))
		return ResultSkipped, out.String()
	}
	found := false
	for _, row := range catalogRows {
		if row.Name == skillName {
			found = true
			break
		}
	}
	if !found {
		_, _ = fmt.Fprintf(&out, "Skipped %s/%s %s: skill no longer cataloged in source %s\n", target, scope, skillName, sourceName)
		return ResultSkipped, out.String()
	}
	sourceSkillDir := filepath.Join(sourcePath, "skills", skillName)
	if _, err := os.Stat(filepath.Join(sourceSkillDir, "SKILL.md")); err != nil {
		_, _ = fmt.Fprintf(&out,
			"Skipped %s/%s %s: source SKILL.md missing: %s\n",
			target, scope, skillName, filepath.Join(sourceSkillDir, "SKILL.md"))
		return ResultSkipped, out.String()
	}
	oldHash, _ := hashSkillDir(skillDir)
	tmpDir := filepath.Join(targetRoot, ".skillhub-update-"+skillName)
	_ = os.RemoveAll(tmpDir)
	if err := copyDir(sourceSkillDir, tmpDir); err != nil {
		_, _ = fmt.Fprintf(&out, "Failed %s/%s %s: could not copy source skill\n", target, scope, skillName)
		return ResultFailed, out.String()
	}
	newHash, _ := hashSkillDir(tmpDir)
	writeSidecar := installUsesSidecar(scope)
	if oldHash != "-" && newHash != "-" && oldHash == newHash {
		_ = os.RemoveAll(tmpDir)
		row := InstalledSkill{
			Source: sourceName, Skill: skillName, Target: target, Scope: scope,
			ProjectPath: projectPath, TargetRoot: targetRoot, InstalledPath: skillDir,
			SourceRef: source.Ref, SourceLocation: source.Location, Catalog: source.Catalog,
			ContentHash: oldHash,
		}
		if err := b.writeManagedSkillState(skillDir, row, installedAt, writeSidecar); err != nil {
			_, _ = fmt.Fprintf(&out, "Failed %s/%s %s: could not persist install metadata: %v\n",
				target, scope, skillName, err)
			return ResultFailed, out.String()
		}
		if verbose {
			_, _ = fmt.Fprintf(&out, "Unchanged %s/%s %s (%s)\n", target, scope, skillName, oldHash)
		}
		return "unchanged", out.String()
	}
	backupDir := filepath.Join(targetRoot, ".skillhub-backup-"+skillName)
	_ = os.RemoveAll(backupDir)
	if err := os.Rename(skillDir, backupDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		_, _ = fmt.Fprintf(&out, "Failed %s/%s %s: could not move current install aside\n", target, scope, skillName)
		return ResultFailed, out.String()
	}
	if err := os.Rename(tmpDir, skillDir); err != nil {
		_ = os.Rename(backupDir, skillDir)
		_ = os.RemoveAll(tmpDir)
		_, _ = fmt.Fprintf(&out, "Failed %s/%s %s: could not replace installed skill\n", target, scope, skillName)
		return ResultFailed, out.String()
	}
	_ = os.RemoveAll(backupDir)
	contentHash, _ := hashSkillDir(skillDir)
	row := InstalledSkill{
		Source: sourceName, Skill: skillName, Target: target, Scope: scope,
		ProjectPath: projectPath, TargetRoot: targetRoot, InstalledPath: skillDir,
		SourceRef: source.Ref, SourceLocation: source.Location, Catalog: source.Catalog,
		ContentHash: contentHash,
	}
	if err := b.writeManagedSkillState(skillDir, row, installedAt, writeSidecar); err != nil {
		_, _ = fmt.Fprintf(&out, "Failed %s/%s %s: updated content but could not persist install metadata: %v\n",
			target, scope, skillName, err)
		return ResultFailed, out.String()
	}
	if verbose {
		_, _ = fmt.Fprintf(&out, "Updated %s/%s %s %s -> %s\n", target, scope, skillName, oldHash, contentHash)
	} else {
		_, _ = fmt.Fprintf(&out, "Updated %s/%s %s\n", target, scope, skillName)
	}
	return "updated", out.String()
}

func (b *Backend) writeManagedSkillState(skillDir string, row InstalledSkill, installedAt string, writeSidecar bool) error {
	if err := b.writeInstallMetadata(skillDir, row, installedAt, writeSidecar); err != nil {
		return err
	}
	if row.Scope != ScopeProject {
		return nil
	}
	if err := b.WriteProjectLockfileFromRegistry(row.ProjectPath); err != nil {
		return err
	}
	return b.EnsureProjectGitignoreMetadata(row.ProjectPath)
}

func (s *UpdateSummary) addUpdateResult(result string) {
	switch result {
	case "updated":
		s.Updated++
	case "unchanged":
		s.Unchanged++
	case ResultFailed:
		s.Failed++
	default:
		s.Skipped++
	}
}

func usageFilterListMatches(sourceName, skillName string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if strings.Contains(filter, "/") {
			if filter == sourceName+"/"+skillName {
				return true
			}
			continue
		}
		if filter == skillName {
			return true
		}
	}
	return false
}
