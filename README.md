# Skillhub

CLI and TUI hub for discovering and installing agent skills from registered
sources. Sources define where skills come from; targets define where skills are
installed. The Go/Cobra command owns the user-facing interface; the POSIX shell
scripts remain the portable backend used by the CLI and TUI.

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
skillhub installed uninstall rules-selector
skillhub installed uninstall rules-selector --target directory --dir /tmp/skills
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
j/k       move
space     select/unselect
enter     open highlighted skill details
/         search
a         select all visible skills
c         clear selection
d         open recommended source presets
n         add custom source path or git URL
t         choose install targets
i         choose targets for selected skills
s         sync sources
r         reload catalog
q         quit
```

The TUI opens as a task-oriented selector: a concise summary header, a readable
category tree, and an action bar. Skill rows show name, description, source, and
triggers without a wide table; selected skills use a green `[✓]` marker. Press
`enter` to open details for the highlighted skill. Press `i` after selecting
skills to choose one or more supported assistants, then press `enter` to install
to all selected targets. When a preset source screen is open, use `enter` to add
the highlighted source or `esc` to return to skills. Use `n` to add a custom
path or git URL from the TUI; custom source flags such as `--name` and `--ref`
remain CLI-only.

Update `skillhub` itself without reinstalling skills:

```sh
skillhub update
skillhub version
```

`skillhub update` refreshes the source checkout and rebuilds the installed
binary. Installed skills are not modified.

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

Inspect installed skills without modifying them:

```sh
skillhub installed list
skillhub installed list --target claude --scope project --project /path/to/project
skillhub installed list --target directory --dir /tmp/skills --tsv
```

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
