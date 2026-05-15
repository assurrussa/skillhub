package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type lockRow struct {
	Source         string
	SourceType     string
	Skill          string
	QualifiedSkill string
	Target         string
	Scope          string
	ProjectPath    string
	TargetRoot     string
	InstalledPath  string
	SourceRef      string
	SourceLocation string
	Catalog        string
	ContentHash    string
	InstalledAt    string
	UpdatedAt      string
}

func (b *Backend) LockfilePath(project string) (string, error) {
	projectPath, err := b.ProjectDir(project)
	if err != nil {
		return "", err
	}
	return filepath.Join(projectPath, "skills.lock.toml"), nil
}

func (b *Backend) WriteProjectLockfileFromRegistry(projectPath string) error {
	usage, err := b.ReadUsage("")
	if err != nil {
		return err
	}
	rows := []InstalledSkill{}
	for _, row := range usage {
		if row.Scope == ScopeProject && row.ProjectPath == projectPath {
			rows = append(rows, row)
		}
	}
	lockfile := filepath.Join(projectPath, "skills.lock.toml")
	if len(rows) == 0 {
		if err := os.Remove(lockfile); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Target != rows[j].Target {
			return rows[i].Target < rows[j].Target
		}
		return rows[i].Skill < rows[j].Skill
	})
	var bld strings.Builder
	_, _ = bld.WriteString("lockfile_version = 1\n")
	for _, row := range rows {
		sourceType := b.sourceTypeForLock(row.Source, row.SourceLocation)
		_, _ = bld.WriteString("\n[[skills]]\n")
		writeTomlField(&bld, "source", row.Source)
		writeTomlField(&bld, "source_type", sourceType)
		writeTomlField(&bld, "skill", row.Skill)
		writeTomlField(&bld, "qualified_skill", row.Source+"/"+row.Skill)
		writeTomlField(&bld, "target", row.Target)
		writeTomlField(&bld, "scope", ScopeProject)
		writeTomlField(&bld, "project_path", ".")
		writeTomlField(&bld, "target_root", projectRelativePath(projectPath, row.TargetRoot))
		writeTomlField(&bld, "installed_path", projectRelativePath(projectPath, row.InstalledPath))
		writeTomlField(&bld, "source_ref", row.SourceRef)
		writeTomlField(&bld, "source_location", row.SourceLocation)
		writeTomlField(&bld, "catalog", row.Catalog)
		writeTomlField(&bld, "content_hash", row.ContentHash)
		writeTomlField(&bld, "installed_at", row.InstalledAt)
		writeTomlField(&bld, "updated_at", row.UpdatedAt)
	}
	return os.WriteFile(lockfile, []byte(bld.String()), 0o644)
}

func (b *Backend) sourceTypeForLock(sourceName, sourceLocation string) string {
	sources, err := b.ListSources()
	if err == nil {
		for _, source := range sources {
			if source.Name == sourceName {
				return source.Type
			}
		}
	}
	return inferSourceType(sourceLocation)
}

func writeTomlField(b *strings.Builder, key, value string) {
	_, _ = fmt.Fprintf(b, "%s = \"%s\"\n", key, tomlEscape(value))
}

func tomlEscape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func tomlUnescape(value string) string {
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, `\\`, `\`)
	return value
}

func readLockfile(path string) ([]lockRow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rows := []lockRow{}
	current := map[string]string{}
	emit := func() {
		if current["skill"] == "" {
			return
		}
		rows = append(rows, lockRow{
			Source: current["source"], SourceType: current["source_type"], Skill: current["skill"],
			QualifiedSkill: current["qualified_skill"], Target: current["target"], Scope: current["scope"],
			ProjectPath: current["project_path"], TargetRoot: current["target_root"], InstalledPath: current["installed_path"],
			SourceRef: current["source_ref"], SourceLocation: current["source_location"], Catalog: current["catalog"],
			ContentHash: current["content_hash"], InstalledAt: current["installed_at"], UpdatedAt: current["updated_at"],
		})
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || line == "lockfile_version = 1" {
			continue
		}
		if line == "[[skills]]" {
			emit()
			current = map[string]string{}
			continue
		}
		key, value, ok := strings.Cut(line, " = ")
		if !ok || !strings.HasPrefix(value, `"`) || !strings.HasSuffix(value, `"`) {
			continue
		}
		current[key] = tomlUnescape(strings.TrimSuffix(strings.TrimPrefix(value, `"`), `"`))
	}
	emit()
	return rows, nil
}

func (b *Backend) Restore(opts RestoreOptions) (RestoreSummary, error) {
	projectPath, err := b.ProjectDir(opts.Project)
	if err != nil {
		return RestoreSummary{}, err
	}
	lockfile := filepath.Join(projectPath, "skills.lock.toml")
	if _, err := os.Stat(lockfile); os.IsNotExist(err) {
		return RestoreSummary{}, nil
	}
	lockRows, err := readLockfile(lockfile)
	if err != nil {
		return RestoreSummary{}, err
	}
	summary := RestoreSummary{}
	for _, row := range lockRows {
		status, reason, sourcePath := b.restoreLockStatus(projectPath, row)
		shouldApply := summary.addRestoreStatus(RestoreRow{
			Source: row.Source, Skill: row.Skill, Target: row.Target, Status: status,
			InstalledPath: row.InstalledPath, ContentHash: row.ContentHash, Reason: reason,
		}, opts.Check)
		if !shouldApply {
			continue
		}
		if err := b.restoreLockRow(projectPath, row, sourcePath); err != nil {
			summary.markRestoreFailure(err)
			continue
		}
		if status == ResultMissing {
			summary.Installed++
		} else {
			summary.Updated++
		}
	}
	if summary.Failed > 0 {
		return summary, fmt.Errorf("restore failed for %d skill(s)", summary.Failed)
	}
	if !opts.Check {
		if err := b.persistProjectMetadata(projectPath); err != nil {
			return summary, err
		}
	}
	return summary, nil
}

func (s *RestoreSummary) addRestoreStatus(row RestoreRow, check bool) bool {
	s.Rows = append(s.Rows, row)
	switch row.Status {
	case ResultSkipped:
		s.Skipped++
		return false
	case "ok":
		s.Unchanged++
		return false
	case ResultMissing:
		if check {
			s.Installed++
			return false
		}
	case ResultChanged:
		if check {
			s.Updated++
			return false
		}
	}
	return true
}

func (s *RestoreSummary) markRestoreFailure(err error) {
	s.Failed++
	if len(s.Rows) == 0 {
		return
	}
	last := len(s.Rows) - 1
	s.Rows[last].Status = ResultFailed
	s.Rows[last].Reason = sanitizeTSVField(err.Error())
}

func (b *Backend) restoreLockRow(projectPath string, row lockRow, sourcePath string) error {
	targetRoot, err := projectLockfilePath(projectPath, row.TargetRoot)
	if err != nil {
		return err
	}
	installedPath, err := projectLockfilePath(projectPath, row.InstalledPath)
	if err != nil {
		return err
	}
	sourceSkillDir := filepath.Join(sourcePath, "skills", row.Skill)
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return err
	}
	if err := copyDir(sourceSkillDir, installedPath); err != nil {
		return err
	}
	hash, err := hashSkillDir(installedPath)
	if err != nil {
		return err
	}
	installedAt := row.InstalledAt
	if installedAt == "" {
		installedAt = "-"
	}
	usage := InstalledSkill{
		Source: row.Source, Skill: row.Skill, Target: row.Target, Scope: ScopeProject,
		ProjectPath: projectPath, TargetRoot: targetRoot, InstalledPath: installedPath,
		SourceRef: row.SourceRef, SourceLocation: row.SourceLocation, Catalog: row.Catalog,
		ContentHash: hash,
	}
	return b.writeInstallMetadata(installedPath, usage, installedAt, false)
}

func (b *Backend) restoreLockStatus(projectPath string, row lockRow) (status, reason, sourcePath string) {
	if _, err := projectLockfilePath(projectPath, row.TargetRoot); err != nil {
		return ResultSkipped, sanitizeTSVField(err.Error()), "-"
	}
	installedPath, err := projectLockfilePath(projectPath, row.InstalledPath)
	if err != nil {
		return ResultSkipped, sanitizeTSVField(err.Error()), "-"
	}
	source := Source{Name: row.Source, Type: row.SourceType, Location: row.SourceLocation, Ref: row.SourceRef, Catalog: row.Catalog}
	sourcePath, _, err = b.CatalogSource(source)
	if err != nil {
		return ResultSkipped, sanitizeTSVField(err.Error()), "-"
	}
	catalogFile := filepath.Join(sourcePath, row.Catalog)
	catalogRows, err := CatalogTable.ReadFile(catalogFile)
	if err != nil {
		return ResultSkipped, "source catalog missing: " + catalogFile, "-"
	}
	found := false
	for _, catalogRow := range catalogRows {
		if catalogRow.Name == row.Skill {
			found = true
			break
		}
	}
	if !found {
		return ResultSkipped, "skill no longer cataloged in source " + row.Source, "-"
	}
	sourceSkillDir := filepath.Join(sourcePath, "skills", row.Skill)
	if _, err := os.Stat(filepath.Join(sourceSkillDir, "SKILL.md")); err != nil {
		return ResultSkipped, "source SKILL.md missing: " + filepath.Join(sourceSkillDir, "SKILL.md"), "-"
	}
	sourceHash, _ := hashSkillDir(sourceSkillDir)
	if _, err := os.Stat(filepath.Join(installedPath, "SKILL.md")); err != nil {
		return ResultMissing, "missing installed skill", sourcePath
	}
	currentHash, _ := hashSkillDir(installedPath)
	if sourceHash != "-" && currentHash != sourceHash {
		return ResultChanged, "installed hash differs from source", sourcePath
	}
	if row.ContentHash != "" && row.ContentHash != "-" && currentHash != row.ContentHash {
		return ResultChanged, "installed hash differs from lockfile", sourcePath
	}
	return "ok", "installed skill matches lockfile", sourcePath
}

func projectLockfilePath(projectPath, lockPath string) (string, error) {
	lockPath = strings.TrimSpace(lockPath)
	if lockPath == "" {
		return "", errors.New("lockfile path is empty")
	}
	if filepath.IsAbs(lockPath) {
		return "", fmt.Errorf("lockfile path must be relative to project: %s", lockPath)
	}
	clean := filepath.Clean(filepath.FromSlash(lockPath))
	if clean == "." {
		return projectPath, nil
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("lockfile path escapes project: %s", lockPath)
	}
	resolved := filepath.Join(projectPath, clean)
	rel, err := filepath.Rel(projectPath, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("lockfile path escapes project: %s", lockPath)
	}
	return resolved, nil
}

func (b *Backend) EnsureProjectGitignoreMetadata(projectPath string) error {
	if projectPath == "" || projectPath == "-" {
		return nil
	}
	cmd := exec.CommandContext(context.Background(), "git", "-C", projectPath, "rev-parse", "--show-toplevel")
	cmd.Env = envSlice(b.env)
	output, _ := cmd.Output()
	gitRoot := strings.TrimSpace(string(output))
	if gitRoot == "" {
		return nil
	}
	gitignore := filepath.Join(gitRoot, ".gitignore")
	existing := ""
	if data, err := os.ReadFile(gitignore); err == nil {
		existing = string(data)
	}
	lines := []string{}
	skip := false
	for _, line := range strings.Split(existing, "\n") {
		if line == "# >>> skillhub local metadata >>>" {
			skip = true
			continue
		}
		if line == "# <<< skillhub local metadata <<<" {
			skip = false
			continue
		}
		if !skip {
			lines = append(lines, line)
		}
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	block := []string{
		"# >>> skillhub local metadata >>>",
		"skills/**/.skillhub.json",
		"skills/.skillhub-*/",
		".*/skills/**/.skillhub.json",
		".*/skills/.skillhub-*/",
		"# <<< skillhub local metadata <<<",
	}
	var content strings.Builder
	if len(lines) > 0 {
		_, _ = content.WriteString(strings.Join(lines, "\n"))
		_, _ = content.WriteString("\n\n")
	}
	_, _ = content.WriteString(strings.Join(block, "\n"))
	_ = content.WriteByte('\n')
	return os.WriteFile(gitignore, []byte(content.String()), 0o644)
}
