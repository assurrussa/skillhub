# Skillhub

Shell-first hub for discovering and installing Codex skills from registered
sources. TUI support can be added later on top of the same source registry.

## Sources

Sources are listed in `sources/sources.tsv`.

Current source:

- `agent-rules` from `git@github.com:assurrussa/agent-rules.git`

For local development, point the source at a checkout instead of cloning:

```sh
SKILLHUB_AGENT_RULES_PATH=/Users/amir/dev/projects/my/agent-rules \
  sh scripts/skills.sh list
```

## Commands

```sh
sh scripts/sources.sh list
sh scripts/sources.sh sync agent-rules
sh scripts/skills.sh list
sh scripts/skills.sh search go
sh scripts/skills.sh install rules-selector
sh scripts/skills.sh install --all
```

Installed skills go to:

```text
~/.agents/skills/<skill-name>
```

Override the install target:

```sh
AGENT_SKILLS_DIR=/tmp/skills sh scripts/skills.sh install rules-selector
```

## Validation

```sh
sh scripts/check.sh
SKILLHUB_AGENT_RULES_PATH=/Users/amir/dev/projects/my/agent-rules \
  sh scripts/skills.sh list
```
