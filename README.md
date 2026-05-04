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
`~/.local/share/skillhub` for curl installs and writes the command to
`~/.local/bin/skillhub`. Override locations:

```sh
sh install.sh --bin-dir /tmp/bin
SKILLHUB_BIN_DIR=/tmp/bin sh install.sh
SKILLHUB_HOME=/tmp/skillhub-home sh install.sh
```

If the install directory is not in `PATH`, the installer prints a note.

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
skillhub skills list
skillhub skills search go
skillhub skills install rules-selector
skillhub skills install rules-selector --target codex --scope project
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
/         search
a         select all visible skills
c         clear selection
d         open recommended source presets
n         add custom source path or git URL
t         toggle Codex global/project target
i         install selected skills
s         sync sources
r         reload catalog
q         quit
```

The skills list shows `source`, `skill`, `category`, and description columns.
When a preset source screen is open, use `enter` to add the highlighted source
or `esc` to return to skills. Use `n` to add a custom path or git URL from the
TUI; custom source flags such as `--name` and `--ref` remain CLI-only.

Update `skillhub` itself without reinstalling skills:

```sh
skillhub update
skillhub version
```

`skillhub update` refreshes the source checkout and rebuilds the installed
binary. Installed skills under `~/.agents/skills` are not modified.

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
```

Supported v1 targets:

- `codex --scope global`: installs to `~/.agents/skills`.
- `codex --scope project`: installs to `<project>/.agents/skills`.
- `directory --dir <path>`: installs directly to an explicit skills directory.

Other assistants are listed as planned targets so the CLI and future TUI can
show the roadmap without writing into unverified formats.

Project-scope installs use the caller working directory unless `--project` is
provided:

```sh
skillhub skills install rules-selector --target codex --scope project
skillhub skills install rules-selector --target codex --scope project --project /path/to/project
skillhub skills install rules-selector --target directory --dir /tmp/skills
```

If two sources publish the same skill name, install one explicitly as
`<source>/<skill-name>`:

```sh
skillhub skills install agent-rules/go-project-rules
```

## Validation

```sh
sh scripts/check.sh
SKILLHUB_CONFIG_DIR=/tmp/skillhub-config sh bin/skillhub sources defaults add agent-rules
SKILLHUB_CONFIG_DIR=/tmp/skillhub-config SKILLHUB_AGENT_RULES_PATH=../agent-rules sh bin/skillhub skills list
```
