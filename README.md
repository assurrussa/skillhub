# Skillhub

CLI and TUI hub for discovering and installing Codex skills from registered
sources. The Go/Cobra command owns the user-facing interface; the POSIX shell
scripts remain the portable backend used by the CLI and TUI.

## Install

Install the `skillhub` command. Go is required because the installed command is
a compiled Cobra/Bubble Tea binary:

```sh
sh install.sh
```

By default it writes `~/.local/bin/skillhub`. Override the command location:

```sh
sh install.sh --bin-dir /tmp/bin
SKILLHUB_BIN_DIR=/tmp/bin sh install.sh
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

Sources are listed in `sources/sources.tsv`.

Current source:

- `agent-rules` from `git@github.com:assurrussa/agent-rules.git`

For local development, point the source at a checkout instead of cloning:

```sh
SKILLHUB_AGENT_RULES_PATH=../agent-rules skillhub skills list
```

## Commands

```sh
skillhub sources list
skillhub sources sync agent-rules
skillhub skills list
skillhub skills search go
skillhub skills install rules-selector
skillhub skills install --all
skillhub tui
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
i         install selected skills
s         sync sources
r         reload catalog
q         quit
```

Installed skills go to:

```text
~/.agents/skills/<skill-name>
```

Override the install target:

```sh
AGENT_SKILLS_DIR=/tmp/skills skillhub skills install rules-selector
```

## Validation

```sh
sh scripts/check.sh
SKILLHUB_AGENT_RULES_PATH=../agent-rules sh bin/skillhub skills list
```
