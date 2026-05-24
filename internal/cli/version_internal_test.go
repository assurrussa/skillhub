package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUpdateInstallPlanDefaultsToRunningBinaryDir(t *testing.T) {
	wantDir, err := runningBinaryDir()
	if err != nil {
		t.Fatalf("resolve running binary dir: %v", err)
	}

	plan, err := updateInstallPlan("/tmp/install.sh", "")
	if err != nil {
		t.Fatalf("build update plan: %v", err)
	}
	want := []string{"/tmp/install.sh", updateUpdateFlag, updateBinDirFlag, wantDir}
	assertStringSlice(t, plan.ScriptArgs, want)
	if plan.UpdatedCommand != filepath.Join(wantDir, "skillhub") {
		t.Fatalf("expected updated command in running binary dir, got %q", plan.UpdatedCommand)
	}
}

func TestUpdateInstallPlanPreservesExplicitBinDir(t *testing.T) {
	binDir := t.TempDir()
	plan, err := updateInstallPlan("/tmp/install.sh", binDir)
	if err != nil {
		t.Fatalf("build update plan: %v", err)
	}
	want := []string{"/tmp/install.sh", updateUpdateFlag, updateBinDirFlag, binDir}
	assertStringSlice(t, plan.ScriptArgs, want)
	if plan.UpdatedCommand != filepath.Join(binDir, "skillhub") {
		t.Fatalf("expected updated command in explicit bin dir, got %q", plan.UpdatedCommand)
	}
}

func TestUpdateCommandPrintsCurrentAndUpdatedVersions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("update command executes install.sh through sh")
	}
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	script := filepath.Join(tmp, "install.sh")
	scriptData := `set -eu
bin_dir=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --bin-dir)
      bin_dir="$2"
      shift 2
      ;;
    --update)
      shift
      ;;
    *)
      shift
      ;;
  esac
done
mkdir -p "$bin_dir"
cat > "$bin_dir/skillhub" <<'SCRIPT'
#!/bin/sh
printf '%s\n' 'skillhub v9.9.9'
printf '%s\n' 'commit: updated'
printf '%s\n' 'built: test'
printf '%s\n' 'repo: test'
printf '%s\n' 'install_script: test'
SCRIPT
chmod 755 "$bin_dir/skillhub"
printf 'fake installer wrote %s\n' "$bin_dir/skillhub"
`
	if err := os.WriteFile(script, []byte(scriptData), 0o644); err != nil {
		t.Fatalf("write fake installer: %v", err)
	}
	restoreVersionGlobals(t, script)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := updateCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{updateBinDirFlag, binDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("update command failed: %v\nstderr:\n%s", err, errOut.String())
	}

	output := out.String()
	for _, want := range []string{
		"Current version:\n",
		"skillhub v-current\n",
		"commit: current\n",
		"fake installer wrote " + filepath.Join(binDir, "skillhub") + "\n",
		"Updated version:\n",
		"skillhub v9.9.9\n",
		"commit: updated\n",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected args %#v, got %#v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected args %#v, got %#v", want, got)
		}
	}
}

func restoreVersionGlobals(t *testing.T, script string) {
	t.Helper()
	previousVersion := version
	previousCommit := commit
	previousBuilt := built
	previousRepoPath := repoPath
	previousInstallScript := installScript
	version = "v-current"
	commit = "current"
	built = "current-built"
	repoPath = "current-repo"
	installScript = script
	t.Cleanup(func() {
		version = previousVersion
		commit = previousCommit
		built = previousBuilt
		repoPath = previousRepoPath
		installScript = previousInstallScript
	})
}
