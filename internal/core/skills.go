package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (b *Backend) ListSkills(query string) ([]Skill, string, error) {
	sources, err := b.ListSources()
	if err != nil {
		return nil, "", err
	}
	if len(sources) == 0 {
		return nil, "", errors.New("no sources configured. Run: skillhub sources defaults list")
	}
	out := []Skill{}
	var warnings strings.Builder
	loadedSources := 0
	sourceErrors := []error{}
	for _, source := range sources {
		sourcePath, warning, err := b.CatalogSource(source)
		if warning != "" {
			_, _ = warnings.WriteString(warning)
		}
		if err != nil {
			sourceErrors = append(sourceErrors, fmt.Errorf("%s: %w", source.Name, err))
			_, _ = fmt.Fprintf(&warnings, "Warning: source %s unavailable: %s\n", source.Name, err)
			continue
		}
		rows, err := CatalogTable.ReadFile(filepath.Join(sourcePath, source.Catalog))
		if err != nil {
			sourceErrors = append(sourceErrors, fmt.Errorf("%s: %w", source.Name, err))
			_, _ = fmt.Fprintf(&warnings, "Warning: source %s unavailable: %s\n", source.Name, err)
			continue
		}
		loadedSources++
		for _, row := range rows {
			if !isValidID(row.Name) {
				return nil, warnings.String(), fmt.Errorf("invalid catalog skill name from %s: %s", source.Name, row.Name)
			}
			skill := Skill{
				Source:      source.Name,
				Name:        row.Name,
				Category:    row.Category,
				Triggers:    row.Triggers,
				Description: row.Description,
			}
			if query != "" && !skillMatches(skill, query) {
				continue
			}
			out = append(out, skill)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].Name < out[j].Name
	})
	if len(out) == 0 {
		if loadedSources == 0 && len(sourceErrors) > 0 {
			return nil, warnings.String(), allSourceCatalogsUnavailableError(sourceErrors)
		}
		if query != "" {
			return nil, warnings.String(), fmt.Errorf("no skills matched query: %s", query)
		}
		return nil, warnings.String(), errors.New("no skills found")
	}
	return out, warnings.String(), nil
}

func allSourceCatalogsUnavailableError(sourceErrors []error) error {
	if len(sourceErrors) == 1 {
		return sourceErrors[0]
	}
	parts := make([]string, 0, len(sourceErrors))
	for _, err := range sourceErrors {
		parts = append(parts, err.Error())
	}
	return fmt.Errorf("no source catalogs available:\n%s", strings.Join(parts, "\n"))
}

func skillMatches(skill Skill, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		skill.Source, skill.Name, skill.Category, skill.Triggers, skill.Description,
	}, " "))
	return strings.Contains(haystack, strings.ToLower(query))
}

type catalogMatch struct {
	source     Source
	sourcePath string
	skill      string
}

type warningCollector struct {
	seen map[string]bool
	out  strings.Builder
}

func (w *warningCollector) Add(warning string) {
	if warning == "" {
		return
	}
	if w.seen == nil {
		w.seen = map[string]bool{}
	}
	if w.seen[warning] {
		return
	}
	w.seen[warning] = true
	_, _ = w.out.WriteString(warning)
}

func (w *warningCollector) String() string {
	if w == nil {
		return ""
	}
	return w.out.String()
}

func (b *Backend) findInstallMatch(wanted string, sources []Source, warnings *warningCollector) (catalogMatch, error) {
	wantedSource := ""
	wantedSkill := wanted
	if strings.Contains(wanted, "/") {
		parts := strings.SplitN(wanted, "/", 2)
		wantedSource = normalizeSourceName(parts[0])
		wantedSkill = parts[1]
		if !isValidID(wantedSource) {
			return catalogMatch{}, fmt.Errorf("invalid source name in qualified skill: %s", wanted)
		}
		if wantedSkill == "" || strings.Contains(wantedSkill, "/") {
			return catalogMatch{}, fmt.Errorf("invalid qualified skill name: %s", wanted)
		}
	}
	if !isValidID(wantedSkill) {
		return catalogMatch{}, fmt.Errorf("invalid skill name: %s", wantedSkill)
	}
	matchCount := 0
	consideredSources := 0
	var match catalogMatch
	for _, source := range sources {
		if wantedSource != "" && source.Name != wantedSource {
			continue
		}
		consideredSources++
		sourcePath, warning, err := b.CatalogSource(source)
		warnings.Add(warning)
		if err != nil {
			return catalogMatch{}, err
		}
		rows, err := CatalogTable.ReadFile(filepath.Join(sourcePath, source.Catalog))
		if err != nil {
			return catalogMatch{}, err
		}
		for _, row := range rows {
			if row.Name == wantedSkill {
				matchCount++
				match = catalogMatch{source: source, sourcePath: sourcePath, skill: wantedSkill}
				break
			}
		}
	}
	if wantedSource != "" && consideredSources == 0 {
		return catalogMatch{}, fmt.Errorf("unknown source: %s", wantedSource)
	}
	if matchCount == 0 {
		return catalogMatch{}, fmt.Errorf("unknown skill: %s", wanted)
	}
	if matchCount > 1 {
		return catalogMatch{}, fmt.Errorf("skill name is ambiguous across sources: %s", wanted)
	}
	return match, nil
}

func (b *Backend) Install(opts InstallOptions) (output string, warning string, err error) {
	target, scope := installTargetScope(opts)
	root, err := b.installTargetRoot(opts, target, scope)
	if err != nil {
		return "", "", err
	}
	metadataScope := installMetadataScope(target, scope)
	projectPath, err := b.installProjectPath(opts.Project, metadataScope)
	if err != nil {
		return "", "", err
	}
	sources, err := b.ListSources()
	if err != nil {
		return "", "", err
	}
	if len(sources) == 0 {
		return "", "", errors.New("no sources configured. Run: skillhub sources defaults list")
	}
	var warnings warningCollector
	names, err := b.installNames(opts, sources, &warnings)
	if err != nil {
		return "", warnings.String(), err
	}
	var out strings.Builder
	for _, name := range names {
		err := b.installOneSkill(installOneOptions{
			Name:          name,
			Sources:       sources,
			Root:          root,
			Target:        target,
			MetadataScope: metadataScope,
			ProjectPath:   projectPath,
			Output:        &out,
			Warnings:      &warnings,
		})
		if err != nil {
			return out.String(), warnings.String(), err
		}
		if metadataScope == ScopeProject {
			if err := b.persistProjectMetadata(projectPath); err != nil {
				return out.String(), warnings.String(), err
			}
		}
	}
	return out.String(), warnings.String(), nil
}

func (b *Backend) persistProjectMetadata(projectPath string) error {
	if err := b.WriteProjectLockfileFromRegistry(projectPath); err != nil {
		return err
	}
	return b.EnsureProjectGitignoreMetadata(projectPath)
}

func installTargetScope(opts InstallOptions) (target string, scope string) {
	target = opts.Target
	if target == "" {
		target = TargetCodex
	}
	scope = opts.Scope
	if scope == "" {
		scope = ScopeGlobal
	}
	return target, scope
}

func (b *Backend) installTargetRoot(opts InstallOptions, target, scope string) (string, error) {
	return b.TargetRoot(TargetRootOptions{
		Target: target, Scope: scope, Project: opts.Project, Dir: opts.Dir,
		ScopeWasSet: opts.Scope != "",
	})
}

func installMetadataScope(target, scope string) string {
	if target == TargetDirectory {
		return ScopeCustom
	}
	return scope
}

func (b *Backend) installProjectPath(projectOption, metadataScope string) (string, error) {
	if metadataScope != ScopeProject {
		return "-", nil
	}
	return b.ProjectDir(projectOption)
}

func (b *Backend) installNames(opts InstallOptions, sources []Source, warnings *warningCollector) ([]string, error) {
	names := append([]string(nil), opts.Names...)
	if opts.All {
		var err error
		names, err = b.allInstallNames(sources, warnings)
		if err != nil {
			return nil, err
		}
	}
	if err := validateInstallNames(names); err != nil {
		return nil, err
	}
	return names, nil
}

func (b *Backend) allInstallNames(sources []Source, warnings *warningCollector) ([]string, error) {
	names := []string{}
	seen := map[string]bool{}
	for _, source := range sources {
		sourcePath, warning, err := b.CatalogSource(source)
		warnings.Add(warning)
		if err != nil {
			return nil, err
		}
		rows, err := CatalogTable.ReadFile(filepath.Join(sourcePath, source.Catalog))
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if seen[row.Name] {
				return nil, duplicateInstallTargetNameError(row.Name)
			}
			seen[row.Name] = true
			names = append(names, source.Name+"/"+row.Name)
		}
	}
	return names, nil
}

func duplicateInstallTargetNameError(name string) error {
	return fmt.Errorf(
		"multiple cataloged skills install to the same target name: %s\n"+
			"Install one explicitly as <source>/%s instead of using --all",
		name,
		name,
	)
}

func validateInstallNames(names []string) error {
	seenNames := map[string]bool{}
	for _, name := range names {
		installName := installNameFromWanted(name)
		if !isValidID(installName) {
			return fmt.Errorf("invalid skill name: %s", installName)
		}
		if seenNames[installName] {
			return fmt.Errorf("multiple requested skills install to the same target name: %s", installName)
		}
		seenNames[installName] = true
	}
	return nil
}

type installOneOptions struct {
	Name          string
	Sources       []Source
	Root          string
	Target        string
	MetadataScope string
	ProjectPath   string
	Output        *strings.Builder
	Warnings      *warningCollector
}

func (b *Backend) installOneSkill(opts installOneOptions) error {
	match, err := b.findInstallMatch(opts.Name, opts.Sources, opts.Warnings)
	if err != nil {
		return err
	}
	sourceSkillDir := filepath.Join(match.sourcePath, "skills", match.skill)
	if _, err := os.Stat(filepath.Join(sourceSkillDir, "SKILL.md")); err != nil {
		return missingCatalogedSkillError(match, sourceSkillDir)
	}
	if err := os.MkdirAll(opts.Root, 0o755); err != nil {
		return err
	}
	installedPath := filepath.Join(opts.Root, match.skill)
	if err := b.ensureInstallCanReplace(installedPath, opts.MetadataScope); err != nil {
		return err
	}
	if err := copyDir(sourceSkillDir, installedPath); err != nil {
		return err
	}
	return b.recordInstalledSkill(match, opts, installedPath)
}

func (b *Backend) ensureInstallCanReplace(installedPath, metadataScope string) error {
	info, err := os.Stat(installedPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("refusing to overwrite non-directory install path: %s", installedPath)
	}
	if _, registryManaged := b.usageRowByPath(installedPath); registryManaged {
		return nil
	}
	if metadataScope != ScopeProject {
		if _, sidecarManaged := readMetadata(filepath.Join(installedPath, ".skillhub.json")); sidecarManaged {
			return nil
		}
	}
	return fmt.Errorf(
		"refusing to overwrite unmanaged skill: %s\nuninstall it with --force or remove it before installing",
		installedPath,
	)
}

func missingCatalogedSkillError(match catalogMatch, sourceSkillDir string) error {
	return fmt.Errorf(
		"source %s cataloged %s but SKILL.md is missing: %s",
		match.source.Name,
		match.skill,
		filepath.Join(sourceSkillDir, "SKILL.md"),
	)
}

func (b *Backend) recordInstalledSkill(match catalogMatch, opts installOneOptions, installedPath string) error {
	contentHash, err := hashSkillDir(installedPath)
	if err != nil {
		return err
	}
	row := InstalledSkill{
		Source: match.source.Name, Skill: match.skill, Target: opts.Target, Scope: opts.MetadataScope,
		ProjectPath: opts.ProjectPath, TargetRoot: opts.Root, InstalledPath: installedPath,
		SourceRef: match.source.Ref, SourceLocation: match.source.Location, Catalog: match.source.Catalog,
		ContentHash: contentHash,
	}
	if err := b.writeInstallMetadata(installedPath, row, "", installUsesSidecar(opts.MetadataScope)); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.Output, "Installed %s from %s to %s\n", match.skill, match.source.Name, installedPath)
	return nil
}

func installNameFromWanted(wanted string) string {
	if strings.Contains(wanted, "/") {
		return wanted[strings.LastIndex(wanted, "/")+1:]
	}
	return wanted
}

func (b *Backend) Uninstall(opts UninstallOptions) (string, error) {
	skill := installNameFromWanted(opts.Skill)
	if !isValidID(skill) {
		return "", fmt.Errorf("invalid installed skill name: %s", skill)
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
		return "", err
	}
	skillDir := filepath.Join(root, skill)
	metadataScope := scope
	if target == TargetDirectory {
		metadataScope = ScopeCustom
	}
	usage, registryManaged := b.usageRowByPath(skillDir)
	if _, err := os.Stat(skillDir); err != nil {
		return b.uninstallMissingSkill(
			skill, root, skillDir, metadataScope, opts, usage, registryManaged, err,
		)
	}
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		return "", fmt.Errorf("refusing to uninstall non-skill directory: %s", skillDir)
	}
	_, sidecarManaged := readMetadata(filepath.Join(skillDir, ".skillhub.json"))
	if metadataScope == ScopeProject {
		sidecarManaged = false
	}
	if !registryManaged && !sidecarManaged && !opts.Force {
		return "", fmt.Errorf("refusing to uninstall unmanaged skill: %s\nuse --force to remove it anyway", skillDir)
	}
	if err := os.RemoveAll(skillDir); err != nil {
		return "", err
	}
	if err := b.removeUsageByPath(skillDir); err != nil {
		return "", err
	}
	if err := b.refreshProjectLockfileAfterUninstall(metadataScope, opts, usage); err != nil {
		return "", err
	}
	return fmt.Sprintf("Uninstalled %s from %s\n", skill, root), nil
}

func (b *Backend) uninstallMissingSkill(
	skill, root, skillDir, metadataScope string,
	opts UninstallOptions,
	usage InstalledSkill,
	registryManaged bool,
	statErr error,
) (string, error) {
	if !os.IsNotExist(statErr) || !registryManaged {
		return "", fmt.Errorf("installed skill not found: %s", skillDir)
	}
	if err := b.removeUsageByPath(skillDir); err != nil {
		return "", err
	}
	if err := b.refreshProjectLockfileAfterUninstall(metadataScope, opts, usage); err != nil {
		return "", err
	}
	return fmt.Sprintf("Removed stale managed registry entry for %s from %s\n", skill, root), nil
}

func (b *Backend) refreshProjectLockfileAfterUninstall(
	metadataScope string,
	opts UninstallOptions,
	usage InstalledSkill,
) error {
	if metadataScope != ScopeProject {
		return nil
	}
	projectPath := strings.TrimSpace(usage.ProjectPath)
	if projectPath == "" || projectPath == "-" {
		var err error
		projectPath, err = b.ProjectDir(opts.Project)
		if err != nil {
			return err
		}
	}
	return b.WriteProjectLockfileFromRegistry(projectPath)
}
