# Skillhub

CLI and TUI hub for discovering and installing agent skills from registered
sources. Sources define where skills come from; targets define where skills are
installed. The Go backend in `internal/core` owns runtime source, catalog,
install, target, lockfile, restore, usage, and recommendation behavior. The
Go/Cobra CLI renders public command output from that backend, and the Bubble
Tea TUI calls it directly.

Runtime and validation paths use the Go CLI/dev commands. `install.sh` remains
the Unix installer and self-update entrypoint.

## Why Skillhub

- Find skills from configured sources, search the catalog, and install queued
  skills into supported agents and scopes without manually copying files.
- Keep Skillhub-managed skills updateable, including project-local installs.
  When a skill is installed through Skillhub from a source, metadata and usage
  registry entries let Skillhub update that exact global, local project, or
  custom-directory install later.
- Commit portable `skills.lock.toml` files for project-scope installs and
  restore missing or changed project skills without mutating user source
  configuration.

## Install

Install the `skillhub` command. Go and Git are required because the installer
clones or updates the source checkout and builds a Cobra/Bubble Tea binary.

One-line install from GitHub:

```sh
curl -fsSL https://raw.githubusercontent.com/assurrussa/skillhub/main/install.sh | sh
```

Bash process-substitution style also works:

```sh
bash <(curl -fsSL https://raw.githubusercontent.com/assurrussa/skillhub/main/install.sh)
```

Install from an existing checkout:

```sh
sh install.sh
```

By default the installer keeps the source checkout in
`~/.local/share/skillhub` for curl installs. For the command itself it first
uses a writable PATH directory from `/opt/homebrew/bin` or `/usr/local/bin`;
when neither can be used safely, it falls back to `~/.local/bin/skillhub`.
Override locations:

```sh
sh install.sh --bin-dir /tmp/bin
sh install.sh --global
SKILLHUB_BIN_DIR=/tmp/bin sh install.sh
SKILLHUB_GLOBAL_BIN_DIRS=/opt/homebrew/bin:/usr/local/bin sh install.sh
SKILLHUB_HOME=/tmp/skillhub-home sh install.sh
```

If the install directory is not in `PATH`, the installer prints ready-to-run
commands for the current shell and `~/.zshrc`. Use `--global` to install into a
PATH directory such as `/usr/local/bin`; that directory must be writable.

You can also run it directly from a checkout:

```sh
go run ./cmd/skillhub skills list
```

Open the interactive selector:

```sh
skillhub
skillhub tui
```

## Sources

Active sources are user configuration, not tracked repository state:

```text
${XDG_CONFIG_HOME:-$HOME/.config}/skillhub/sources.tsv
```

Fresh installs start with no active sources. Recommended source presets are
listed in `defaults/sources.tsv` and must be added explicitly:

```sh
skillhub sources defaults list
skillhub sources defaults add agent-rules
skillhub sources defaults add mattpocock
skillhub sources list
```

Git sources are cached under `${SKILLHUB_CACHE_DIR:-$HOME/.cache/skillhub}`.
Catalog commands (`skills list/search`, `recommend`, ordinary `install`, and
the TUI Skills screen) are cache-first and do not run implicit `git fetch`,
`git clone`, or `git pull`. Stale cached catalogs remain usable with a warning.
Missing git caches fail with a hint to run `skillhub sources sync <source>`.
Use `skillhub sources sync [source]`, or update sources from the TUI Sources
screen, whenever you need fresh source data.

Sources can either provide a native `catalog/skills.tsv` plus flat
`skills/<name>/SKILL.md` directories, a nested `skills/**/SKILL.md` tree, or
root-level `<name>/SKILL.md` directories. Cataloged sources are used as-is.
Catalog-less sources are materialized into an internal flat cache with names
built from the path:

```text
skills/engineering/tdd/SKILL.md -> engineering_tdd
skills/productivity/grill-me/SKILL.md -> productivity_grill-me
tdd/SKILL.md -> tdd
```

That makes external skill collections installable through the same commands:

```sh
skillhub sources add https://github.com/mattpocock/skills --name mattpocock
skillhub search tdd
skillhub install mattpocock/engineering_tdd --target codex --scope global
# Nested collections still get path-prefixed names:
skillhub install some-source/engineering_tdd --target claude --scope project
```

GitHub tree URLs for branch-specific skill collections are normalized to the
repository URL and branch ref:

```sh
skillhub sources add https://github.com/example/skills-repo/tree/agent-skills/skills/ --name example
# equivalent:
skillhub sources add https://github.com/example/skills-repo --name example --ref agent-skills
```

For local development, point the source at a checkout instead of cloning:

```sh
skillhub sources defaults add agent-rules
SKILLHUB_AGENT_RULES_PATH=../agent-rules skillhub skills list
```

## Commands

```sh
skillhub sources list
skillhub sources list --tsv
skillhub sources status
skillhub sources status --tsv
skillhub sources sync agent-rules
skillhub sources defaults list
skillhub sources defaults add agent-rules
skillhub sources defaults add mattpocock
skillhub sources add ../agent-rules --name local-agent-rules
skillhub sources add https://github.com/mattpocock/skills --name mattpocock
skillhub sources remove local-agent-rules
skillhub targets list
skillhub targets detect
skillhub targets detect --tsv --project /path/to/project
skillhub installed list
skillhub installed list --target claude --scope project --project /path/to/project
skillhub installed update
skillhub installed update --target claude --scope project --project /path/to/project
skillhub installed update --target directory --dir /tmp/skills -v
skillhub installed uninstall rules-selector
skillhub installed uninstall rules-selector --target directory --dir /tmp/skills
skillhub installed usage
skillhub installed usage rules-selector
skillhub installed usage agent-rules/go-project-rules --tsv
skillhub installed usage update --projects
skillhub installed usage update --projects rules-selector -v
skillhub installed usage update --projects --target codex --project /path/to/project agent-rules/go-project-rules
skillhub restore --project /path/to/project
skillhub restore --check --project /path/to/project
skillhub restore --check --project /path/to/project --tsv
skillhub recommend
skillhub recommend --project /path/to/project
skillhub recommend --project /path/to/project --tsv
skillhub skills list
skillhub skills search go
skillhub skills install rules-selector
skillhub skills restore --project /path/to/project
skillhub add rules-selector
skillhub skills install rules-selector --target codex --scope project
skillhub skills install rules-selector --target claude --scope global
skillhub skills install rules-selector --target gemini --scope project
skillhub skills install rules-selector --target opencode --scope global
skillhub skills install rules-selector --target directory --dir /tmp/skills
skillhub skills install --all
skillhub tui
skillhub update
skillhub update --cascade
skillhub update --cascade -v
skillhub version
```

Short aliases are also supported:

```sh
skillhub list
skillhub search go
skillhub install rules-selector
skillhub add rules-selector
skillhub install --all
```

`skillhub add <skill>` is a top-level alias for `skillhub install <skill>`.
The existing `install` command remains supported.

In the TUI:

```text
1/2/3/4/5/6 switch sections
left/right switch sections
j/k       move
space     queue/unqueue skill for install
enter     open highlighted details
/         search skills or filter usage
a         queue all visible skills
c         clear install queue
d         open recommended source presets
n         add custom source path or git URL
t         open target paths
i         install queued skills
u         update highlighted install, usage, or source entry
U         update all visible project usage entries, or all sources in Sources
x         uninstall highlighted managed skill
s         update all sources in Sources
r         reload current section, or restore project lockfile in Update
?         help
q         quit
```

The TUI opens as an installed-first dashboard with sections for installed
skills, catalog skills, managed usage, sources, targets, and update commands.
The Skills section keeps the readable category tree and green `[+]` install
queue marker, and shows compact installed badges for managed skills whose
target directory still exists. Press `i` after queueing skills to open the
install wizard: choose `Project` or `User` scope, select one or more supported
assistants, review the target paths, then confirm. During install the TUI runs
one skill-target step at a time and shows the current step, queue state,
spinner, progress bar, and last result line.

The Installed section lists managed and unmanaged `SKILL.md` directories grouped
by target/scope. Press `u` to update managed skills for the highlighted
target/scope, or `x` to uninstall the highlighted managed skill after
confirmation. Unmanaged uninstall remains CLI-only with explicit `--force`.

The Usage section reads the managed usage registry and groups installs by
`source/skill`. Press `enter` to see every global, project, or custom-directory
location for that skill. Press `/` to filter by source, skill, target, scope,
project root, or install path. Press `u` to update the highlighted skill or
location, and `U` to update all visible recorded project-scope installs.

The Sources section shows active source freshness, last sync time, cache path,
configured ref/catalog/location, recommended presets, custom source entry, and
source updates. Press `j/k` to move between active sources, `u` to update the
highlighted source, or `U`/`s` to update all active sources. Updates run one
source at a time with a progress panel, queue state, spinner, progress bar, and
last result/error. The Skills screen reads only the local source cache; opening
the catalog never performs git network work. The Update section intentionally
does not run self-update from inside the TUI; it shows project lockfile status,
restores the current project lockfile with `r`,
and shows the exact CLI commands for `skillhub update`, cascade update, managed
skill update, and restore.

Update `skillhub` itself without reinstalling skills:

```sh
skillhub update
skillhub version
```

`skillhub update` refreshes the source checkout and rebuilds the installed
binary. Installed skills are not modified.

Update `skillhub` and managed installed skills together:

```sh
skillhub update --cascade
skillhub update --cascade -v
```

Cascade update first updates the `skillhub` command, then updates managed
installed skills across supported global targets and project targets for the
current directory, then updates all recorded project-scope installs from the
managed usage registry. It only touches Skillhub-managed skill directories:
sidecar-backed global/custom installs and registry-backed project installs.
Use `-v` or `--verbose` to print per-skill target, hash, and result details.

Sources are stored outside the installed checkout:

```text
${XDG_CONFIG_HOME:-$HOME/.config}/skillhub/sources.tsv
```

Override this location for tests or isolated setups:

```sh
SKILLHUB_CONFIG_DIR=/tmp/skillhub-config skillhub sources add ../agent-rules --name local-agent-rules
```

Installed skills default to the Codex-compatible global target:

```text
~/.agents/skills/<skill-name>
```

Override the install target:

```sh
AGENT_SKILLS_DIR=/tmp/skills skillhub skills install rules-selector
```

Targets are listed in `targets/targets.tsv`:

```sh
skillhub targets list
skillhub targets detect
skillhub targets detect --tsv --project /path/to/project
```

Supported v1 targets:

- `codex --scope global`: installs to `~/.agents/skills`.
- `codex --scope project`: installs to `<project>/.agents/skills`.
- `claude --scope global`: installs to `~/.claude/skills`.
- `claude --scope project`: installs to `<project>/.claude/skills`.
- `gemini --scope global`: installs to `~/.gemini/skills`.
- `gemini --scope project`: installs to `<project>/.gemini/skills`.
- `opencode --scope global`: installs to `~/.config/opencode/skills`, or
  `$OPENCODE_CONFIG_DIR/skills` when `OPENCODE_CONFIG_DIR` is set.
- `opencode --scope project`: installs to `<project>/.opencode/skills`.
- `directory --dir <path>`: installs directly to an explicit skills directory.

Other assistants are listed as planned targets so the CLI and future TUI can
show the roadmap without writing into unverified formats.

Global and custom-directory installs write sidecar metadata to the installed
skill directory:

```text
<target-root>/<skill-name>/.skillhub.json
```

Project-scope installs do not write machine-local metadata inside the project.
Their update metadata lives only in the user config registry:

```text
${XDG_CONFIG_HOME:-$HOME/.config}/skillhub/installed.tsv
```

The registry tracks Skillhub-managed installs created or updated by this
version. Project-scope rows include the resolved project path; global and
custom directory rows use `project_path=-`. It is not an import index for old
manual installs. Project skill content such as `SKILL.md`, references, and docs
can be committed normally; Skillhub adds a small `.gitignore` block for old
`.skillhub.json` files and temporary update directories.

Project-scope installs also write a portable project lockfile:

```text
<project>/skills.lock.toml
```

The lockfile stores relative project paths such as `project_path = "."`,
`target_root = ".agents/skills"`, and
`installed_path = ".agents/skills/<skill>"`. It records the source location,
ref, catalog path, target, content hash, and timestamps needed to restore the
same project skills on another machine. Global and custom-directory installs do
not create a project lockfile.

Restore project skills from the lockfile:

```sh
skillhub restore --project /path/to/project
skillhub restore --check --project /path/to/project
skillhub restore --check --project /path/to/project --tsv
skillhub skills restore --project /path/to/project
```

Restore uses the source location, type, ref, and catalog recorded in
`skills.lock.toml` directly and does not add sources to user config. It installs
missing entries, refreshes changed entries, skips unavailable sources or missing
catalog entries with a summary, and never deletes extra project skills. If the
last project-scope skill is uninstalled, Skillhub removes the empty lockfile.

Inspect installed skills without modifying them:

```sh
skillhub installed list
skillhub installed list --target claude --scope project --project /path/to/project
skillhub installed list --target directory --dir /tmp/skills --tsv
```

Look up managed usage across targets and projects:

```sh
skillhub installed usage
skillhub installed usage rules-selector
skillhub installed usage agent-rules/go-project-rules --tsv
```

Update managed installed skills:

```sh
skillhub installed update
skillhub installed update --target claude --scope project --project /path/to/project
skillhub installed update --target directory --dir /tmp/skills -v
skillhub installed usage update --projects
skillhub installed usage update --projects rules-selector -v
skillhub installed usage update --projects --target codex --project /path/to/project agent-rules/go-project-rules
```

`installed update` syncs the source recorded in sidecar metadata for
global/custom installs or in the central registry for project installs, compares
the installed skill hash with the current source skill, and rewrites only
changed managed skills. If the source or catalog entry is missing, the skill is
skipped and left installed.

`installed usage update --projects` updates only project-scope installs recorded
in `installed.tsv`. Use `--target` and `--project` to narrow the recorded rows
without changing the default all-project behavior. `skillhub update --cascade`
also runs this project-usage update after the regular managed target update.

Recommend a minimal project skill set without installing anything:

```sh
skillhub recommend
skillhub recommend --project /path/to/project
skillhub recommend --project /path/to/project --tsv
```

Recommendations are ranked from active source catalog categories/triggers and
repository evidence such as `AGENTS.md`, local assistant rules, README/docs,
Go modules/workspaces, package manifests, public package indicators, and
OpenAPI/protobuf contracts. Human output includes scores and evidence-based
reasons; `--tsv` keeps the stable `source skill reason install_arg` columns in
ranked order. With no active sources, the command prints the same onboarding
message as `skills list`.

Uninstall managed skills:

```sh
skillhub installed uninstall rules-selector
skillhub installed uninstall rules-selector --target claude --scope project --project /path/to/project
skillhub installed uninstall rules-selector --target directory --dir /tmp/skills
```

By default uninstall removes only Skillhub-managed directories, either from
sidecar metadata or from the central registry. Use `--force` only when
intentionally removing an unmanaged skill directory that still contains
`SKILL.md`.

`targets detect` reports whether each supported target path exists, how many
`SKILL.md` directories it contains, and how many of those are managed by
Skillhub metadata or registry rows.

Project-scope installs use the caller working directory unless `--project` is
provided:

```sh
skillhub skills install rules-selector --target codex --scope project
skillhub skills install rules-selector --target codex --scope project --project /path/to/project
skillhub skills install rules-selector --target claude --scope global
skillhub skills install rules-selector --target gemini --scope project
skillhub skills install rules-selector --target opencode --scope global
skillhub skills install rules-selector --target directory --dir /tmp/skills
```

If two sources publish the same skill name, install one explicitly as
`<source>/<skill-name>`:

```sh
skillhub skills install agent-rules/go-project-rules
```

## Future Work

The current implementation is intentionally conservative. Useful next slices:

- Import existing `.skillhub.json` metadata into `installed.tsv` so installs
  created before the usage registry can be managed without reinstalling.
- Add richer TUI Usage manager actions such as source-level bulk operations,
  saved filters, and dry-run summaries across recorded project installs.
- Add real target adapters for more assistants after their current on-disk
  formats and supported paths are verified.
- Add more concrete `skillhub recommend` language/framework providers and
  source-specific heuristics while keeping the no-install recommendation flow.

## Validation

```sh
go run ./cmd/skillhub-dev verify
go run ./cmd/skillhub-dev smoke-temp
go run ./cmd/skillhub-dev tui-temp
```

Unix convenience wrappers:

```sh
make verify
make smoke-temp
make tui-temp
```

Focused raw checks:

```sh
go test ./...
go vet ./...
git diff --check
SKILLHUB_CONFIG_DIR=/tmp/skillhub-config go run ./cmd/skillhub sources defaults add agent-rules
SKILLHUB_CONFIG_DIR=/tmp/skillhub-config SKILLHUB_AGENT_RULES_PATH=../agent-rules go run ./cmd/skillhub skills list
```
