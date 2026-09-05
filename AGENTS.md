# Skillhub Agent Notes

## Scope

This repository owns the `skillhub` CLI, TUI, Go backend, installer, source
presets, and target registry. It must stay neutral: no active skill source is
required at runtime, and user-added sources live outside the checkout.

## Source Of Truth

- User request and this `AGENTS.md` override general rules.
- `internal/core` is the source of truth for source, catalog, install, target,
  metadata, lockfile, restore, usage, and recommendation behavior.
- The Go/Cobra CLI and Bubble Tea TUI should call `internal/core` directly and
  must not shell out for runtime behavior.
- `cmd/skillhub-dev` owns cross-platform development validation and temp smoke
  workflows. Makefile targets are Unix convenience wrappers around it.
- `README.md` documents the public CLI contract. Update it when commands,
  flags, targets, config files, or install behavior changes.

## Architecture Boundaries

- `internal/cli` owns user-facing Cobra command wiring, flag parsing, and repo
  discovery.
- `internal/tui` owns interactive selection and display state only. It should
  call typed `internal/core` services for source, target, install, and registry
  operations instead of parsing CLI stdout.
- `internal/core` owns shared backend helpers and validation.
- `defaults/sources.tsv` contains recommended presets only. It must not become
  an active registry.
- Active sources are user config only:
  `${XDG_CONFIG_HOME:-$HOME/.config}/skillhub/sources.tsv`, or
  `SKILLHUB_CONFIG_DIR/sources.tsv` in tests.
- Managed install usage is user config only:
  `${XDG_CONFIG_HOME:-$HOME/.config}/skillhub/installed.tsv`, or
  `SKILLHUB_CONFIG_DIR/installed.tsv` in tests. It tracks only
  Skillhub-managed installs created or updated by Skillhub.
- `targets/targets.tsv` is the registry for supported and planned install
  targets. Do not write to an assistant-specific format until its current path
  and file contract are verified.

## Development Rules

- Keep runtime tests isolated from the real user config. Use
  `SKILLHUB_CONFIG_DIR`, temp directories, and local source overrides.
- Use `SKILLHUB_AGENT_RULES_PATH=../agent-rules` when testing the bundled
  `agent-rules` preset against a sibling checkout.
- Preserve legacy Codex install behavior: `skillhub install <skill>` and
  `AGENT_SKILLS_DIR=/tmp/skills skillhub install <skill>` must keep working.
- Project-scope installs must record the resolved project path in metadata and
  `installed.tsv`; global and custom directory installs use `project_path=-`.
- Recommendation flow is read-only. `skillhub recommend` analyzes active source
  catalogs and project signals, then prints install commands without running
  them.
- For TUI changes, cover cursor movement, selection state, and rendered output
  with `internal/tui` tests when practical.
- Use `$find-docs` for current Cobra, Bubble Tea, Lip Gloss, or other library docs
  when changing APIs or library-specific behavior.

## Project Initialization

- Treat this repository as initialized when `AGENTS.md` exists, relevant
  project-scope skills are installed under `.agents/skills`, and
  `skills.lock.toml` records those project installs.
- To initialize or refresh recommendations, run `skillhub recommend --project .`
  first and install only relevant project skills. Do not blindly install low
  confidence recommendations from unrelated sources.
- Project-scope installs are allowed to update `.agents/skills`,
  `skills.lock.toml`, and Skillhub-managed registry rows. They must not depend
  on `.skillhub.json` sidecars inside the project.

## Common Checks

Prefer Makefile targets for local work:

```sh
make verify
make smoke-temp
make tui-temp
```

Equivalent raw checks:

```sh
go run ./cmd/skillhub-dev verify
go run ./cmd/skillhub-dev smoke-temp
go run ./cmd/skillhub-dev tui-temp
```

## Shared Agent Context

Use `$project-context-router` for cross-project context after local grounding.
Resolve the shared root through `AGENT_CONTEXT_ROOT` or the skill resolver.
Local verified docs and code remain the source of truth.

When shared context is needed, follow `streams/AGENTS.md` and its query route.
Reuse already loaded root rules, PII policy and glossary. Open the known hub
and only the topic relevant to the task:

- `streams/wiki/platforms/skillhub.md`

For integration work, open only the affected neighbour hub:

- `streams/wiki/platforms/agent-rules.md`

Use `streams/wiki/index.md` only to locate an unknown area or answer an overview
question. This is a task router, not a mandatory list of wiki pages.


If the wiki disagrees with local evidence, report the drift. Update the shared
page only when documentation upkeep is in scope, after verification.
