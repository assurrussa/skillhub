# Skillhub Agent Notes

## Scope

This repository owns the `skillhub` CLI, TUI, shell backend, installer, source
presets, and target registry. It must stay neutral: no active skill source is
required at runtime, and user-added sources live outside the checkout.

## Source Of Truth

- User request and this `AGENTS.md` override general rules.
- Shell scripts are the portable backend. Keep source, install, target, and
  metadata behavior in `scripts/*.sh` and `scripts/lib.sh` first.
- The Go/Cobra CLI and Bubble Tea TUI should delegate to the shell backend
  rather than reimplementing persistence or install semantics.
- `README.md` documents the public CLI contract. Update it when commands,
  flags, targets, config files, or install behavior changes.

## Architecture Boundaries

- `internal/cli` owns user-facing Cobra command wiring, flag parsing, and repo
  discovery.
- `internal/tui` owns interactive selection and display state only. It should
  call the same scripts as the CLI for source, target, and install operations.
- `scripts/lib.sh` owns shared POSIX shell helpers and validation.
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
- Use Context7 for current Cobra, Bubble Tea, Lip Gloss, or other library docs
  when changing APIs or library-specific behavior.

## Common Checks

Prefer Makefile targets for local work:

```sh
make verify
make smoke-temp
make tui-temp
```

Equivalent raw checks:

```sh
sh scripts/check.sh
go vet ./...
git diff --check
```
