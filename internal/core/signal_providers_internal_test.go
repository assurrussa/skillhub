package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSignalCollectorNormalizesTokensAndKeepsFirstEvidence(t *testing.T) {
	collector := newSignalCollector()

	collector.Add(" Go ", "go.mod", 28)
	collector.Add("go", "Go source files", 18)
	collector.Add("   ", "ignored", 99)
	collector.Add("Docs", " docs/ ", 10)

	signals := collector.Signals()
	if len(signals) != 2 {
		t.Fatalf("expected 2 signals, got %#v", signals)
	}
	if got := signals["go"]; got.Evidence != fileNameGoMod || got.Weight != 28 {
		t.Fatalf("expected first normalized go signal to win, got %#v", got)
	}
	if got := signals["docs"]; got.Evidence != "docs/" || got.Weight != 10 {
		t.Fatalf("expected trimmed docs evidence, got %#v", got)
	}

	signals["go"] = signal{Evidence: "mutated", Weight: 1}
	if got := collector.Signals()["go"]; got.Evidence != "go.mod" || got.Weight != 28 {
		t.Fatalf("collector should return a defensive copy, got %#v", got)
	}
}

func TestDefaultSignalProvidersAreDeterministic(t *testing.T) {
	providers := defaultSignalProviders()
	got := make([]string, 0, len(providers))
	for _, provider := range providers {
		got = append(got, provider.Name())
	}
	want := []string{"basic", "documentation", "go", "contracts", "library"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("provider order changed: got %#v want %#v", got, want)
	}

	providers[0] = newSignalProvider("changed", func(_ string, _ *signalCollector) {})
	if got := defaultSignalProviders()[0].Name(); got != "basic" {
		t.Fatalf("default providers should be returned as a copy, got %q", got)
	}
}

func TestCollectProjectSignalsPreservesFirstProviderEvidence(t *testing.T) {
	projectDir := t.TempDir()
	for _, dir := range []string{"docs", "pkg", "cmd/service"} {
		if err := os.MkdirAll(filepath.Join(projectDir, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	files := map[string]string{
		"README.md":           "# Project\n",
		fileNameGoMod:         "module example.com/project\n",
		fileNameGoWork:        "go 1.26.0\n\nuse .\n",
		"docs/openapi.yaml":   "openapi: 3.0.0\n",
		"pkg/public.go":       "package pkg\n",
		"cmd/service/main.go": "package main\nfunc main() {}\n",
	}
	for name, content := range files {
		path := filepath.Join(projectDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	signals := collectProjectSignals(projectDir)
	for token, want := range map[string]signal{
		"documentation":  {Evidence: "README", Weight: 22},
		"go":             {Evidence: "go.mod", Weight: 28},
		"library":        {Evidence: "go.work", Weight: 20},
		"public surface": {Evidence: "pkg/", Weight: 26},
	} {
		if got := signals[token]; got != want {
			t.Fatalf("signal %q = %#v, want %#v", token, got, want)
		}
	}
}

func TestRecommendEngineUsesExplicitSignalProviders(t *testing.T) {
	projectDir := t.TempDir()
	engine := newRecommendEngine([]signalProvider{
		newSignalProvider("custom", func(gotProjectDir string, collector *signalCollector) {
			if gotProjectDir != projectDir {
				t.Fatalf("provider received project dir %q, want %q", gotProjectDir, projectDir)
			}
			collector.Add("custom-framework", "custom.config", 77)
		}),
	})

	rows := engine.recommend(projectDir, []Skill{
		{Source: "local", Name: "general", Category: "workflow", Triggers: "repo"},
		{Source: "local", Name: "custom-rules", Category: "tooling", Triggers: "custom-framework"},
	})

	if len(rows) != 1 {
		t.Fatalf("expected one provider-backed recommendation, got %#v", rows)
	}
	if rows[0].Skill != "custom-rules" ||
		rows[0].Score != 77 ||
		rows[0].Reason != "matched custom.config" ||
		rows[0].InstallArg != "local/custom-rules" {
		t.Fatalf("unexpected recommendation row: %#v", rows[0])
	}
}
