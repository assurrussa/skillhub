package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/assurrussa/skillhub/internal/core"
)

const (
	devFlagDir          = "--dir"
	devFlagTarget       = "--target"
	devTargetDirectory  = "directory"
	devCommandInstalled = "installed"
	devSmokeSkill       = "go-project-rules"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printUsage(os.Stdout)
		return nil
	}

	root, err := findRepoRoot()
	if err != nil {
		return err
	}

	switch args[0] {
	case "verify":
		return runVerify(root)
	case "smoke-temp":
		return runSmokeTemp(root)
	case "tui-temp":
		return runTUITemp(root)
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printUsage(out *os.File) {
	_, _ = fmt.Fprintln(out, "Usage:")
	_, _ = fmt.Fprintln(out, "  go run ./cmd/skillhub-dev verify")
	_, _ = fmt.Fprintln(out, "  go run ./cmd/skillhub-dev smoke-temp")
	_, _ = fmt.Fprintln(out, "  go run ./cmd/skillhub-dev tui-temp")
}

func runVerify(root string) error {
	backend, err := core.NewDefault(root)
	if err != nil {
		return err
	}
	sources, err := backend.ListDefaultSources()
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return errors.New("no default source presets configured")
	}
	targets, err := backend.ListTargets()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "Validated %d default source preset(s).\n", len(sources))
	_, _ = fmt.Fprintf(os.Stdout, "Validated %d install target(s).\n", len(targets))

	ctx := context.Background()
	if err := runInherit(ctx, root, nil, "go", "test", "./..."); err != nil {
		return err
	}
	if err := runInherit(ctx, root, nil, "go", "vet", "./..."); err != nil {
		return err
	}
	return runInherit(ctx, root, nil, "git", "diff", "--check")
}

func runSmokeTemp(root string) error {
	state, err := newSmokeTempState(root)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "Using temp dir: %s\n", state.tmp)
	if err := state.installGlobalSkill(); err != nil {
		return err
	}
	if err := state.installProjectSkills(); err != nil {
		return err
	}
	if err := state.checkUsageUpdates(); err != nil {
		return err
	}
	if err := state.checkRecommendations(); err != nil {
		return err
	}
	return state.checkDirectoryLifecycle()
}

type smokeTempState struct {
	root      string
	tmp       string
	project   string
	projectB  string
	configDir string
	homeDir   string
	skillsDir string
	baseEnv   []string
}

func newSmokeTempState(root string) (smokeTempState, error) {
	tmp, err := os.MkdirTemp("", "skillhub-smoke-*")
	if err != nil {
		return smokeTempState{}, err
	}
	project := filepath.Join(tmp, "project")
	projectB := filepath.Join(tmp, "project-b")
	dirs := []string{
		filepath.Join(project, "docs"),
		filepath.Join(project, "pkg"),
		filepath.Join(project, "cmd", "service"),
		projectB,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return smokeTempState{}, err
		}
	}
	files := map[string]string{
		filepath.Join(project, "go.mod"):                    "module example.com/project\n\ngo 1.26.0\n",
		filepath.Join(project, "go.work"):                   "go 1.26.0\n\nuse .\n",
		filepath.Join(project, "README.md"):                 "# Temp Go service\n\nOpenAPI contracts and architecture docs.\n",
		filepath.Join(project, "docs", "openapi.yaml"):      "openapi: 3.0.0\ninfo:\n  title: API\n  version: v1\n",
		filepath.Join(project, "pkg", "public.go"):          "package pkg\n",
		filepath.Join(project, "cmd", "service", "main.go"): "package main\nfunc main() {}\n",
	}
	for path, data := range files {
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			return smokeTempState{}, err
		}
	}

	configDir := filepath.Join(tmp, "config")
	homeDir := filepath.Join(tmp, "home")
	baseEnv := make([]string, 0, 4)
	baseEnv = append(baseEnv, "HOME="+homeDir)
	baseEnv = append(baseEnv, "SKILLHUB_CONFIG_DIR="+configDir)
	baseEnv = append(baseEnv, "SKILLHUB_AGENT_RULES_PATH="+agentRulesPath(root))

	return smokeTempState{
		root:      root,
		tmp:       tmp,
		project:   project,
		projectB:  projectB,
		configDir: configDir,
		homeDir:   homeDir,
		skillsDir: filepath.Join(tmp, "skills"),
		baseEnv:   baseEnv,
	}, nil
}

func (s smokeTempState) isolatedEnv() []string {
	return []string{"HOME=" + s.homeDir, "SKILLHUB_CONFIG_DIR=" + s.configDir}
}

func (s smokeTempState) installGlobalSkill() error {
	if err := s.run(s.baseEnv, "sources", "defaults", "add", "agent-rules"); err != nil {
		return err
	}
	if err := s.run(s.baseEnv, "search", "go"); err != nil {
		return err
	}
	env := appendEnv(s.baseEnv, "AGENT_SKILLS_DIR="+s.skillsDir)
	if err := s.run(env, "install", devSmokeSkill); err != nil {
		return err
	}
	usage, err := s.usageOutput()
	if err != nil {
		return err
	}
	writeOutput(usage)
	if err := requireContains("installed usage", usage, devSmokeSkill); err != nil {
		return err
	}
	for _, path := range []string{
		filepath.Join(s.skillsDir, devSmokeSkill, "SKILL.md"),
		filepath.Join(s.skillsDir, devSmokeSkill, ".skillhub.json"),
	} {
		if err := requireFile(path); err != nil {
			return err
		}
	}
	return nil
}

func (s smokeTempState) installProjectSkills() error {
	installs := []struct {
		target  string
		project string
	}{
		{target: "codex", project: s.project},
		{target: "claude", project: s.projectB},
	}
	for _, install := range installs {
		err := s.run(
			s.baseEnv,
			"install",
			devSmokeSkill,
			"--target",
			install.target,
			"--scope",
			"project",
			"--project",
			install.project,
		)
		if err != nil {
			return err
		}
	}
	for _, want := range []string{s.project, s.projectB} {
		usage, err := s.usageOutput()
		if err != nil {
			return err
		}
		writeOutput(usage)
		if err := requireContains("installed usage", usage, want); err != nil {
			return err
		}
	}
	return nil
}

func (s smokeTempState) usageOutput() (string, error) {
	return s.capture(
		s.isolatedEnv(),
		devCommandInstalled,
		"usage",
		devSmokeSkill,
		"--tsv",
	)
}

func (s smokeTempState) checkUsageUpdates() error {
	usageUpdate, err := runSkillhubCapture(
		context.Background(),
		s.root,
		s.baseEnv,
		devCommandInstalled,
		"usage",
		"update",
		"--projects",
		devSmokeSkill,
		"-v",
	)
	if err != nil {
		return err
	}
	writeOutput(usageUpdate)
	for _, want := range []string{
		"Checking codex/project " + devSmokeSkill,
		"Checking claude/project " + devSmokeSkill,
	} {
		if err := requireContains("usage update", usageUpdate, want); err != nil {
			return err
		}
	}
	return s.checkTargetedUsageUpdate()
}

func (s smokeTempState) checkTargetedUsageUpdate() error {
	targeted, err := runSkillhubCapture(
		context.Background(),
		s.root,
		s.baseEnv,
		devCommandInstalled,
		"usage",
		"update",
		"--projects",
		"--target",
		"codex",
		"--project",
		s.project,
		devSmokeSkill,
		"-v",
	)
	if err != nil {
		return err
	}
	writeOutput(targeted)
	if err := requireContains(
		"targeted usage update",
		targeted,
		"Checking codex/project "+devSmokeSkill,
	); err != nil {
		return err
	}
	if strings.Contains(targeted, "Checking claude/project "+devSmokeSkill) {
		return errors.New("targeted usage update included claude/project row")
	}
	return nil
}

func (s smokeTempState) checkRecommendations() error {
	if err := s.run(s.baseEnv, "recommend", "--project", s.project); err != nil {
		return err
	}
	recommend, err := s.capture(
		s.baseEnv,
		"recommend",
		"--project",
		s.project,
		"--tsv",
	)
	if err != nil {
		return err
	}
	writeOutput(recommend)
	for _, want := range []string{
		"agent-rules\tgo-project-rules\t",
	} {
		if err := requireContains("recommend", recommend, want); err != nil {
			return err
		}
	}
	return nil
}

func (s smokeTempState) checkDirectoryLifecycle() error {
	commands := [][]string{
		{devCommandInstalled, "list", devFlagTarget, devTargetDirectory, devFlagDir, s.skillsDir},
		{devCommandInstalled, "update", devFlagTarget, devTargetDirectory, devFlagDir, s.skillsDir},
		{devCommandInstalled, "uninstall", devSmokeSkill, devFlagTarget, devTargetDirectory, devFlagDir, s.skillsDir},
	}
	for _, args := range commands {
		if err := s.run(s.isolatedEnv(), args...); err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(s.skillsDir, devSmokeSkill)); !os.IsNotExist(err) {
		if err == nil {
			return errors.New("expected directory install to be removed")
		}
		return err
	}
	return s.run(s.isolatedEnv(), "targets", "detect", "--project", s.project)
}

func (s smokeTempState) run(env []string, args ...string) error {
	return runSkillhub(context.Background(), s.root, env, args...)
}

func (s smokeTempState) capture(env []string, args ...string) (string, error) {
	return runSkillhubCapture(context.Background(), s.root, env, args...)
}

func runTUITemp(root string) error {
	ctx := context.Background()
	tmp, err := os.MkdirTemp("", "skillhub-tui-*")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "Using temp dir: %s\n", tmp)
	_, _ = fmt.Fprintf(os.Stdout, "Config: %s\n", filepath.Join(tmp, "config"))
	_, _ = fmt.Fprintf(os.Stdout, "Install dir: %s\n", filepath.Join(tmp, "skills"))
	env := make([]string, 0, 4)
	env = append(env, "HOME="+filepath.Join(tmp, "home"))
	env = append(env, "SKILLHUB_CONFIG_DIR="+filepath.Join(tmp, "config"))
	env = append(env, "SKILLHUB_AGENT_RULES_PATH="+agentRulesPath(root))
	if err := runSkillhub(ctx, root, env, "sources", "defaults", "add", "agent-rules"); err != nil {
		return err
	}
	env = appendEnv(env, "AGENT_SKILLS_DIR="+filepath.Join(tmp, "skills"))
	return runSkillhub(ctx, root, env, "tui")
}

func runSkillhub(ctx context.Context, root string, env []string, args ...string) error {
	return runInherit(ctx, root, env, "go", append([]string{"run", "./cmd/skillhub"}, args...)...)
}

func runSkillhubCapture(ctx context.Context, root string, env []string, args ...string) (string, error) {
	return runCapture(ctx, root, env, "go", append([]string{"run", "./cmd/skillhub"}, args...)...)
}

func runInherit(ctx context.Context, dir string, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCapture(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return stdout.String(), err
}

func appendEnv(base []string, value string) []string {
	env := make([]string, 0, len(base)+1)
	env = append(env, base...)
	env = append(env, value)
	return env
}

func agentRulesPath(root string) string {
	value := strings.TrimSpace(os.Getenv("SKILLHUB_AGENT_RULES_PATH"))
	if value == "" {
		value = filepath.Join("..", "agent-rules")
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, value)
}

func requireContains(label, got, want string) error {
	if strings.Contains(got, want) {
		return nil
	}
	return fmt.Errorf("%s output missing %q", label, want)
}

func writeOutput(output string) {
	_, _ = fmt.Fprint(os.Stdout, output)
}

func requireFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected file, got directory: %s", path)
	}
	return nil
}

func findRepoRoot() (string, error) {
	if env := strings.TrimSpace(os.Getenv("SKILLHUB_REPO")); env != "" {
		return cleanRepoRoot(env)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := filepath.Clean(wd); ; dir = filepath.Dir(dir) {
		if hasRepoFiles(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir || strings.TrimSpace(parent) == "" {
			break
		}
	}
	return "", errors.New("skillhub repository not found; run from the checkout or set SKILLHUB_REPO")
}

func cleanRepoRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !hasRepoFiles(abs) {
		return "", fmt.Errorf("invalid skillhub repository %s: missing required skillhub files", abs)
	}
	return abs, nil
}

func hasRepoFiles(dir string) bool {
	for _, rel := range []string{
		"go.mod",
		filepath.Join("cmd", "skillhub", "main.go"),
		filepath.Join("defaults", "sources.tsv"),
		filepath.Join("targets", "targets.tsv"),
	} {
		// #nosec G703 -- rel comes from the fixed repository file list above.
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			return false
		}
	}
	return true
}
