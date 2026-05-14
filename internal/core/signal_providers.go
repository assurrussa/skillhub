package core

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	fileNameGoMod  = "go.mod"
	fileNameGoWork = "go.work"
)

type signal struct {
	Evidence string
	Weight   int
}

type signalCollector struct {
	signals map[string]signal
}

type signalProvider interface {
	Name() string
	CollectSignals(projectDir string, collector *signalCollector)
}

type signalProviderFunc struct {
	name    string
	collect func(projectDir string, collector *signalCollector)
}

func newSignalProvider(name string, collect func(projectDir string, collector *signalCollector)) signalProvider {
	return signalProviderFunc{name: name, collect: collect}
}

func (f signalProviderFunc) Name() string {
	return f.name
}

func (f signalProviderFunc) CollectSignals(projectDir string, collector *signalCollector) {
	f.collect(projectDir, collector)
}

func defaultSignalProviders() []signalProvider {
	return append([]signalProvider(nil), defaultProjectSignalProviders...)
}

var defaultProjectSignalProviders = []signalProvider{
	newSignalProvider("basic", collectBasicProjectSignals),
	newSignalProvider("documentation", collectDocumentationSignals),
	newSignalProvider("go", collectGoSignals),
	newSignalProvider("contracts", collectContractSignals),
	newSignalProvider("library", collectLibrarySignals),
}

func newSignalCollector() *signalCollector {
	return &signalCollector{signals: map[string]signal{}}
}

func (c *signalCollector) Add(token, evidence string, weight int) {
	token = normalizeSignalToken(token)
	if token == "" {
		return
	}
	if _, ok := c.signals[token]; ok {
		return
	}
	c.signals[token] = signal{Evidence: strings.TrimSpace(evidence), Weight: weight}
}

func (c *signalCollector) Signals() map[string]signal {
	signals := make(map[string]signal, len(c.signals))
	for token, signal := range c.signals {
		signals[token] = signal
	}
	return signals
}

func normalizeSignalToken(token string) string {
	return strings.ToLower(strings.TrimSpace(token))
}

func collectProjectSignals(projectDir string) map[string]signal {
	return collectProjectSignalsWithProviders(projectDir, defaultSignalProviders())
}

func collectProjectSignalsWithProviders(projectDir string, providers []signalProvider) map[string]signal {
	collector := newSignalCollector()
	for _, provider := range providers {
		provider.CollectSignals(projectDir, collector)
	}
	return collector.Signals()
}

func collectBasicProjectSignals(projectDir string, collector *signalCollector) {
	files := []string{
		"AGENTS.md", "CLAUDE.md", "GEMINI.md", "README.md", "README.rst", "README.txt",
		"Taskfile.yml", "Makefile", "package.json", "composer.json", "pyproject.toml",
		"Cargo.toml", fileNameGoMod, fileNameGoWork,
	}
	for _, file := range files {
		if exists(filepath.Join(projectDir, file)) {
			collector.Add("workflow", file, 12)
			collector.Add("repo orientation", file, 10)
		}
	}
	if exists(filepath.Join(projectDir, "AGENTS.md")) ||
		exists(filepath.Join(projectDir, "CLAUDE.md")) ||
		exists(filepath.Join(projectDir, "GEMINI.md")) ||
		isDir(filepath.Join(projectDir, ".cursor", "rules")) {
		collector.Add("verification", "local agent rules", 10)
		collector.Add("scope", "local agent rules", 8)
		collector.Add("review", "local agent rules", 6)
	}
}

func collectDocumentationSignals(projectDir string, collector *signalCollector) {
	if exists(filepath.Join(projectDir, "README.md")) ||
		exists(filepath.Join(projectDir, "README.rst")) ||
		exists(filepath.Join(projectDir, "README.txt")) {
		collector.Add("documentation", "README", 22)
		collector.Add("readme", "README", 24)
	}
	if isDir(filepath.Join(projectDir, "docs")) {
		collector.Add("docs", "docs/", 24)
		collector.Add("documentation", "docs/", 18)
	}
	if hasFileMatching(projectDir, 4, func(name string) bool {
		lower := strings.ToLower(name)
		return strings.Contains(lower, "architecture") ||
			strings.Contains(lower, "runbook") ||
			strings.Contains(lower, "audit") ||
			strings.Contains(lower, "report")
	}) {
		collector.Add("architecture", "architecture/runbook/audit/report docs", 18)
		collector.Add("runbook", "architecture/runbook/audit/report docs", 10)
		collector.Add("audit", "architecture/runbook/audit/report docs", 8)
		collector.Add("report", "architecture/runbook/audit/report docs", 8)
	}
}

func collectGoSignals(projectDir string, collector *signalCollector) {
	if exists(filepath.Join(projectDir, "go.mod")) {
		collector.Add("go", "go.mod", 28)
		collector.Add("golang", "go.mod", 12)
		collector.Add("go.mod", "go.mod", 48)
		if data, err := os.ReadFile(filepath.Join(projectDir, "go.mod")); err == nil &&
			strings.Contains(string(data), "\nreplace ") {
			collector.Add("replace", "go.mod replace", 28)
			collector.Add("external consumer", "go.mod replace", 12)
		}
	}
	if exists(filepath.Join(projectDir, "go.work")) {
		collector.Add("go", "go.work", 18)
		collector.Add("go.work", "go.work", 44)
		collector.Add("reusable module", "go.work", 32)
		collector.Add("library", "go.work", 20)
	}
	if hasFileMatching(projectDir, 4, func(name string) bool { return strings.HasSuffix(name, ".go") }) {
		collector.Add("go", "Go source files", 18)
		collector.Add("golang", "Go source files", 8)
	}
}

func collectContractSignals(projectDir string, collector *signalCollector) {
	if hasFileMatching(projectDir, 5, func(name string) bool {
		lower := strings.ToLower(name)
		return strings.HasSuffix(lower, ".proto") ||
			strings.Contains(lower, "openapi") ||
			strings.Contains(lower, "swagger")
	}) {
		collector.Add("documentation", "OpenAPI/protobuf contracts", 28)
		collector.Add("architecture", "OpenAPI/protobuf contracts", 16)
		collector.Add("docs", "OpenAPI/protobuf contracts", 12)
		collector.Add("contract change", "OpenAPI/protobuf contracts", 10)
	}
}

func collectLibrarySignals(projectDir string, collector *signalCollector) {
	if isDir(filepath.Join(projectDir, "pkg")) {
		collector.Add("public surface", "pkg/", 26)
		collector.Add("library", "pkg/", 20)
		collector.Add("reusable module", "pkg/", 18)
	}
	if data, err := os.ReadFile(filepath.Join(projectDir, "package.json")); err == nil &&
		(strings.Contains(string(data), `"main"`) ||
			strings.Contains(string(data), `"exports"`) ||
			strings.Contains(string(data), `"types"`)) {
		collector.Add("public surface", "package.json exports", 26)
		collector.Add("library", "package.json exports", 18)
		collector.Add("reusable module", "package.json exports", 16)
	}
	if exists(filepath.Join(projectDir, "composer.json")) ||
		exists(filepath.Join(projectDir, "pyproject.toml")) ||
		exists(filepath.Join(projectDir, "Cargo.toml")) {
		collector.Add("library", "package manifest", 18)
		collector.Add("reusable module", "package manifest", 14)
	}
}

func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasFileMatching(root string, maxDepth int, match func(string) bool) bool {
	root = filepath.Clean(root)
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if found {
			return nil
		}
		if path == root {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		depth := strings.Count(filepath.ToSlash(rel), "/") + 1
		if d.IsDir() {
			if depth > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if depth <= maxDepth && match(d.Name()) {
			found = true
		}
		return nil
	})
	return found
}
