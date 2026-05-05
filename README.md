# Skillhub

CLI and TUI hub for discovering and installing agent skills from registered
sources. Sources define where skills come from; targets define where skills are
installed. The Go/Cobra command owns the user-facing interface; the POSIX shell
scripts remain the portable backend used by the CLI and TUI.

## Why Skillhub

- Find skills from configured sources, search the catalog, and install selected
  skills into supported agents and scopes without manually copying files.
- Keep Skillhub-managed skills updateable, including project-local installs.
  When a skill is installed through Skillhub from a source, metadata and usage
  registry entries let Skillhub update that exact global, local project, or
  custom-directory install later.

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
sh bin/skillhub skills list
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
skillhub sources list
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
skillhub sources sync agent-rules
skillhub sources defaults list
skillhub sources defaults add agent-rules
skillhub sources add ../agent-rules --name local-agent-rules
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
skillhub recommend
skillhub recommend --project /path/to/project
skillhub recommend --project /path/to/project --tsv
skillhub skills list
skillhub skills search go
skillhub skills install rules-selector
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
skillhub install --all
```

In the TUI:

```text
1/2/3/4/5/6 switch sections
j/k       move
space     select/unselect
enter     open highlighted details
/         search
a         select all visible skills
c         clear selection
d         open recommended source presets
n         add custom source path or git URL
t         choose install targets
i         choose targets for selected skills
u         update highlighted install or usage entry
x         uninstall highlighted managed skill
s         sync sources
r         reload catalog
?         help
q         quit
```

The TUI opens as a dashboard with sections for catalog skills, installed skills,
managed usage, sources, targets, and update commands. The Skills section keeps
the readable category tree and green `[✓]` selection marker. Press `i` after
selecting skills to choose one or more supported assistants, then press `enter`
to install to all selected targets.

The Installed section lists managed and unmanaged `SKILL.md` directories grouped
by target/scope. Press `u` to update managed skills for the highlighted
target/scope, or `x` to uninstall the highlighted managed skill after
confirmation. Unmanaged uninstall remains CLI-only with explicit `--force`.

The Usage section reads the managed usage registry and groups installs by
`source/skill`. Press `enter` to see every global, project, or custom-directory
location for that skill. Press `u` to update recorded project-scope installs for
the highlighted skill.

The Sources section shows active sources, recommended presets, custom source
entry, and source sync. The Update section intentionally does not run
self-update from inside the TUI; it shows the exact CLI commands for `skillhub
update`, cascade update, and managed skill update.

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
managed usage registry. It only touches skill directories with `.skillhub.json`.
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

Successful installs write metadata to the installed skill directory:

```text
<target-root>/<skill-name>/.skillhub.json
```

Managed install usage is also recorded in user config:

```text
${XDG_CONFIG_HOME:-$HOME/.config}/skillhub/installed.tsv
```

The registry tracks Skillhub-managed installs created or updated by this
version. Project-scope rows include the resolved project path; global and
custom directory rows use `project_path=-`. It is not an import index for old
manual installs.

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
```

`installed update` syncs the source recorded in `.skillhub.json`, compares the
installed skill hash with the current source skill, and rewrites only changed
managed skills. If the source or catalog entry is missing, the skill is skipped
and left installed.

`installed usage update --projects` updates only project-scope installs recorded
in `installed.tsv`. `skillhub update --cascade` also runs this project-usage
update after the regular managed target update.

Recommend a minimal project skill set without installing anything:

```sh
skillhub recommend
skillhub recommend --project /path/to/project
skillhub recommend --project /path/to/project --tsv
```

Recommendations are based on active source catalogs and project signals such as
`AGENTS.md`, README/docs, Go workspaces, package manifests, and public
OpenAPI/protobuf contracts. With no active sources, the command prints the same
onboarding message as `skills list`.

Uninstall managed skills:

```sh
skillhub installed uninstall rules-selector
skillhub installed uninstall rules-selector --target claude --scope project --project /path/to/project
skillhub installed uninstall rules-selector --target directory --dir /tmp/skills
```

By default uninstall removes only Skillhub-managed directories that contain
`.skillhub.json`. Use `--force` only when intentionally removing an unmanaged
skill directory that still contains `SKILL.md`.

`targets detect` reports whether each supported target path exists, how many
`SKILL.md` directories it contains, and how many of those are managed by
Skillhub metadata.

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
- Extend the TUI Usage manager with filters, update-all actions for a
  skill/source, and bulk operations across recorded project installs.
- Add real target adapters for more assistants after their current on-disk
  formats and supported paths are verified.
- Improve `skillhub recommend` so project analysis ranks skills more precisely
  from repository signals, local rules, and active source catalogs.

## Validation

```sh
make verify
make smoke-temp
make tui-temp
```

Raw commands:

```sh
sh scripts/check.sh
SKILLHUB_CONFIG_DIR=/tmp/skillhub-config sh bin/skillhub sources defaults add agent-rules
SKILLHUB_CONFIG_DIR=/tmp/skillhub-config SKILLHUB_AGENT_RULES_PATH=../agent-rules sh bin/skillhub skills list
```
