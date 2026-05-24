package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultCatalogPath = "catalog/skills.tsv"

type Backend struct {
	repoRoot  string
	callerCWD string
	env       map[string]string
	now       func() time.Time
}

func New(ctx Context) (*Backend, error) {
	repoRoot := strings.TrimSpace(ctx.RepoRoot)
	absRepo := ""
	if repoRoot != "" {
		var err error
		absRepo, err = filepath.Abs(repoRoot)
		if err != nil {
			return nil, err
		}
	}
	cwd := strings.TrimSpace(ctx.CallerCWD)
	if cwd == "" {
		if envCWD := ctx.Env["SKILLHUB_CALLER_CWD"]; strings.TrimSpace(envCWD) != "" {
			cwd = envCWD
		} else if wd, err := os.Getwd(); err == nil {
			cwd = wd
		} else {
			cwd = "."
		}
	}
	if abs, err := filepath.Abs(cwd); err == nil {
		cwd = abs
	}
	env := map[string]string{}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			env[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	for key, value := range ctx.Env {
		env[key] = value
	}
	now := ctx.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Backend{repoRoot: absRepo, callerCWD: cwd, env: env, now: now}, nil
}

func NewDefault(repoRoot string) (*Backend, error) {
	env := map[string]string{}
	if cwd := os.Getenv("SKILLHUB_CALLER_CWD"); strings.TrimSpace(cwd) != "" {
		env["SKILLHUB_CALLER_CWD"] = cwd
	}
	return New(Context{RepoRoot: repoRoot, Env: env})
}

func (b *Backend) RepoRoot() string {
	return b.repoRoot
}

func (b *Backend) CallerCWD() string {
	return b.callerCWD
}

func (b *Backend) commandDir() string {
	if b.repoRoot != "" {
		return b.repoRoot
	}
	return b.callerCWD
}

func (b *Backend) envValue(name string) string {
	return b.env[name]
}

func (b *Backend) ConfigRoot() (string, error) {
	if value := strings.TrimSpace(b.envValue("SKILLHUB_CONFIG_DIR")); value != "" {
		return value, nil
	}
	if value := strings.TrimSpace(b.envValue("XDG_CONFIG_HOME")); value != "" {
		return filepath.Join(value, "skillhub"), nil
	}
	home := strings.TrimSpace(b.envValue("HOME"))
	if home == "" {
		return "", errors.New("HOME is not set; set SKILLHUB_CONFIG_DIR explicitly")
	}
	return filepath.Join(home, ".config", "skillhub"), nil
}

func (b *Backend) CacheRoot() (string, error) {
	if value := strings.TrimSpace(b.envValue("SKILLHUB_CACHE_DIR")); value != "" {
		return value, nil
	}
	home := strings.TrimSpace(b.envValue("HOME"))
	if home == "" {
		return "", errors.New("HOME is not set; set SKILLHUB_CACHE_DIR explicitly")
	}
	return filepath.Join(home, ".cache", "skillhub"), nil
}

func (b *Backend) DefaultSourcesFile() string {
	return filepath.Join(b.repoRoot, "defaults", "sources.tsv")
}

func (b *Backend) TargetsFile() string {
	return filepath.Join(b.repoRoot, "targets", "targets.tsv")
}

func (b *Backend) UserSourcesFile() (string, error) {
	root, err := b.ConfigRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "sources.tsv"), nil
}

func (b *Backend) InstalledRegistryFile() (string, error) {
	root, err := b.ConfigRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "installed.tsv"), nil
}

func (b *Backend) sourceStateDir() (string, error) {
	root, err := b.CacheRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "source-state"), nil
}

func (b *Backend) sourceSyncStateFile(name string) (string, error) {
	dir, err := b.sourceStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".synced_at"), nil
}

func (b *Backend) generatedSourcePath(name string) (string, error) {
	root, err := b.CacheRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "generated-sources", name), nil
}

func (b *Backend) cachedGitSourcePath(name string) (string, error) {
	root, err := b.CacheRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "sources", name), nil
}

func (b *Backend) AbsPath(path string, base string) (string, error) {
	if strings.TrimSpace(base) == "" {
		base = b.callerCWD
	}
	raw := path
	if !filepath.IsAbs(raw) {
		raw = filepath.Join(base, raw)
	}
	parent := filepath.Dir(raw)
	name := filepath.Base(raw)
	if info, err := os.Stat(parent); err == nil && info.IsDir() {
		parentAbs, err := filepath.Abs(parent)
		if err != nil {
			return "", err
		}
		return filepath.Join(parentAbs, name), nil
	}
	return filepath.Clean(raw), nil
}

func (b *Backend) ProjectDir(project string) (string, error) {
	if strings.TrimSpace(project) == "" {
		project = b.callerCWD
	}
	resolved, err := b.AbsPath(project, b.callerCWD)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(resolved); err == nil && info.IsDir() {
		return filepath.Abs(resolved)
	}
	return resolved, nil
}

func (b *Backend) defaultCodexRoot() (string, error) {
	if dir := strings.TrimSpace(b.envValue("AGENT_SKILLS_DIR")); dir != "" {
		return dir, nil
	}
	home := strings.TrimSpace(b.envValue("HOME"))
	if home == "" {
		return "", errors.New("HOME is not set; set AGENT_SKILLS_DIR or use --target directory --dir <path>")
	}
	return filepath.Join(home, ".agents", "skills"), nil
}

func (b *Backend) defaultClaudeRoot() (string, error) {
	home := strings.TrimSpace(b.envValue("HOME"))
	if home == "" {
		return "", errors.New("HOME is not set; use --target directory --dir <path>")
	}
	return filepath.Join(home, ".claude", "skills"), nil
}

func (b *Backend) defaultGeminiRoot() (string, error) {
	home := strings.TrimSpace(b.envValue("HOME"))
	if home == "" {
		return "", errors.New("HOME is not set; use --target directory --dir <path>")
	}
	return filepath.Join(home, ".gemini", "skills"), nil
}

func (b *Backend) defaultAntigravityRoot() (string, error) {
	home := strings.TrimSpace(b.envValue("HOME"))
	if home == "" {
		return "", errors.New("HOME is not set; use --target directory --dir <path>")
	}
	return filepath.Join(home, ".gemini", "antigravity", "skills"), nil
}

func (b *Backend) defaultOpenCodeRoot() (string, error) {
	if value := strings.TrimSpace(b.envValue("OPENCODE_CONFIG_DIR")); value != "" {
		return filepath.Join(value, "skills"), nil
	}
	home := strings.TrimSpace(b.envValue("HOME"))
	if home == "" {
		return "", errors.New("HOME is not set; set OPENCODE_CONFIG_DIR or use --target directory --dir <path>")
	}
	return filepath.Join(home, ".config", "opencode", "skills"), nil
}

type TargetRootOptions struct {
	Target      string
	Scope       string
	Project     string
	Dir         string
	ScopeWasSet bool
}

func (b *Backend) TargetRoot(opts TargetRootOptions) (string, error) {
	target, scope := normalizeTargetRootOptions(opts)
	return b.skillDirTargetRoot(opts, target, scope)
}

func normalizeTargetRootOptions(opts TargetRootOptions) (target string, scope string) {
	target = strings.TrimSpace(opts.Target)
	if target == "" {
		target = TargetCodex
	}
	scope = strings.TrimSpace(opts.Scope)
	if scope == "" {
		scope = ScopeGlobal
	}
	return target, scope
}

func (b *Backend) skillDirTargetRoot(opts TargetRootOptions, target, scope string) (string, error) {
	switch target {
	case TargetCodex:
		if err := validateTargetScope(scope); err != nil {
			return "", err
		}
		return b.scopedSkillDirRoot(scope, opts.Project, ".agents")
	case TargetClaude:
		if err := validateTargetScope(scope); err != nil {
			return "", err
		}
		return b.scopedSkillDirRoot(scope, opts.Project, ".claude")
	case TargetGemini:
		if err := validateTargetScope(scope); err != nil {
			return "", err
		}
		return b.scopedSkillDirRoot(scope, opts.Project, ".gemini")
	case TargetAntigravity:
		if err := validateTargetScope(scope); err != nil {
			return "", err
		}
		if scope == ScopeProject {
			return b.scopedSkillDirRoot(scope, opts.Project, ".agents")
		}
		return b.defaultAntigravityRoot()
	case "opencode":
		if err := validateTargetScope(scope); err != nil {
			return "", err
		}
		if scope == ScopeProject {
			project, err := b.ProjectDir(opts.Project)
			if err != nil {
				return "", err
			}
			return filepath.Join(project, ".opencode", "skills"), nil
		}
		return b.defaultOpenCodeRoot()
	case TargetDirectory:
		if opts.ScopeWasSet {
			return "", errors.New("target directory does not support --scope")
		}
		if strings.TrimSpace(opts.Dir) == "" {
			return "", errors.New("target directory requires --dir <path>")
		}
		return b.AbsPath(opts.Dir, b.callerCWD)
	default:
		return "", fmt.Errorf("target %s is not implemented by the skill-dir adapter", target)
	}
}

func validateTargetScope(scope string) error {
	if scope == ScopeGlobal || scope == ScopeProject {
		return nil
	}
	return fmt.Errorf("invalid scope %q: must be %s or %s", scope, ScopeGlobal, ScopeProject)
}

func (b *Backend) scopedSkillDirRoot(scope, projectOption, dotDir string) (string, error) {
	if scope == ScopeProject {
		project, err := b.ProjectDir(projectOption)
		if err != nil {
			return "", err
		}
		return filepath.Join(project, dotDir, "skills"), nil
	}
	switch dotDir {
	case ".agents":
		return b.defaultCodexRoot()
	case ".claude":
		return b.defaultClaudeRoot()
	case ".gemini":
		return b.defaultGeminiRoot()
	default:
		return "", fmt.Errorf("unsupported skill directory root: %s", dotDir)
	}
}

func projectRelativePath(projectPath, absolutePath string) string {
	projectPath = filepath.Clean(projectPath)
	absolutePath = filepath.Clean(absolutePath)
	if absolutePath == projectPath {
		return "."
	}
	rel, err := filepath.Rel(projectPath, absolutePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return absolutePath
	}
	return filepath.ToSlash(rel)
}

func isValidID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func inferSourceType(location string) string {
	if strings.Contains(location, "://") || strings.HasPrefix(location, "git@") || strings.HasSuffix(location, ".git") {
		return SourceTypeGit
	}
	return SourceTypePath
}
