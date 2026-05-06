SHELL := /bin/sh

GO ?= go
SKILLHUB ?= sh bin/skillhub
AGENT_RULES_PATH ?= ../agent-rules

.PHONY: help check test vet diff-check verify smoke-temp tui-temp

help:
	@printf '%s\n' \
		'Targets:' \
		'  make check       Run repository validation script' \
		'  make test        Run Go tests' \
		'  make vet         Run go vet' \
		'  make diff-check  Run git diff --check' \
		'  make verify      Run check, vet, and diff-check' \
		'  make smoke-temp  Test default source and temp install without user config' \
		'  make tui-temp    Open TUI with temp config and temp install dir' \
		'' \
		'Variables:' \
		'  AGENT_RULES_PATH=../agent-rules' \
		'  SKILLHUB="sh bin/skillhub"'

check:
	sh scripts/check.sh

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

diff-check:
	git diff --check

verify: check vet diff-check

smoke-temp:
	@set -e; \
	tmp=$$(mktemp -d); \
	project="$$tmp/project"; \
	project_b="$$tmp/project-b"; \
	mkdir -p "$$project/docs" "$$project/pkg" "$$project/cmd/service" "$$project_b"; \
	printf 'module example.com/project\n\ngo 1.26.0\n' > "$$project/go.mod"; \
	printf 'go 1.26.0\n\nuse .\n' > "$$project/go.work"; \
	printf '# Temp Go service\n\nOpenAPI contracts and architecture docs.\n' > "$$project/README.md"; \
	printf 'openapi: 3.0.0\ninfo:\n  title: API\n  version: v1\n' > "$$project/docs/openapi.yaml"; \
	printf 'package pkg\n' > "$$project/pkg/public.go"; \
	printf 'package main\nfunc main() {}\n' > "$$project/cmd/service/main.go"; \
	printf 'Using temp dir: %s\n' "$$tmp"; \
	SKILLHUB_CONFIG_DIR="$$tmp/config" \
	SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" \
	$(SKILLHUB) sources defaults add agent-rules; \
	SKILLHUB_CONFIG_DIR="$$tmp/config" \
	SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" \
	$(SKILLHUB) search go; \
		SKILLHUB_CONFIG_DIR="$$tmp/config" \
		SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" \
		AGENT_SKILLS_DIR="$$tmp/skills" \
		$(SKILLHUB) install rules-selector; \
		SKILLHUB_CONFIG_DIR="$$tmp/config" \
		$(SKILLHUB) installed usage rules-selector --tsv | grep 'rules-selector'; \
		test -f "$$tmp/skills/rules-selector/SKILL.md"; \
		test -f "$$tmp/skills/rules-selector/.skillhub.json"; \
		SKILLHUB_CONFIG_DIR="$$tmp/config" \
		SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" \
		$(SKILLHUB) install rules-selector --target codex --scope project --project "$$project"; \
		SKILLHUB_CONFIG_DIR="$$tmp/config" \
		SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" \
		$(SKILLHUB) install rules-selector --target claude --scope project --project "$$project_b"; \
		SKILLHUB_CONFIG_DIR="$$tmp/config" \
		$(SKILLHUB) installed usage rules-selector --tsv | grep "$$project"; \
		SKILLHUB_CONFIG_DIR="$$tmp/config" \
		$(SKILLHUB) installed usage rules-selector --tsv | grep "$$project_b"; \
		usage_update_output=$$(SKILLHUB_CONFIG_DIR="$$tmp/config" SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" $(SKILLHUB) installed usage update --projects rules-selector -v); \
		printf '%s\n' "$$usage_update_output"; \
		printf '%s\n' "$$usage_update_output" | grep 'Checking codex/project rules-selector'; \
		printf '%s\n' "$$usage_update_output" | grep 'Checking claude/project rules-selector'; \
		targeted_usage_output=$$(SKILLHUB_CONFIG_DIR="$$tmp/config" SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" $(SKILLHUB) installed usage update --projects --target codex --project "$$project" rules-selector -v); \
		printf '%s\n' "$$targeted_usage_output"; \
		printf '%s\n' "$$targeted_usage_output" | grep 'Checking codex/project rules-selector'; \
		! printf '%s\n' "$$targeted_usage_output" | grep 'Checking claude/project rules-selector'; \
		SKILLHUB_CONFIG_DIR="$$tmp/config" \
		SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" \
		$(SKILLHUB) recommend --project "$$project"; \
		recommend_output=$$(SKILLHUB_CONFIG_DIR="$$tmp/config" SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" $(SKILLHUB) recommend --project "$$project" --tsv); \
		printf '%s\n' "$$recommend_output"; \
		printf '%s\n' "$$recommend_output" | grep '^agent-rules	go-project-rules	'; \
		printf '%s\n' "$$recommend_output" | grep '^agent-rules	docs-project-rules	'; \
		SKILLHUB_CONFIG_DIR="$$tmp/config" \
		$(SKILLHUB) installed list --target directory --dir "$$tmp/skills"; \
	SKILLHUB_CONFIG_DIR="$$tmp/config" \
	$(SKILLHUB) installed update --target directory --dir "$$tmp/skills"; \
	SKILLHUB_CONFIG_DIR="$$tmp/config" \
	$(SKILLHUB) installed uninstall rules-selector --target directory --dir "$$tmp/skills"; \
	test ! -e "$$tmp/skills/rules-selector"; \
	SKILLHUB_CONFIG_DIR="$$tmp/config" \
	$(SKILLHUB) targets detect --project "$$project"

tui-temp:
	@tmp=$$(mktemp -d); \
	printf 'Using temp dir: %s\n' "$$tmp"; \
	printf 'Config: %s/config\n' "$$tmp"; \
	printf 'Install dir: %s/skills\n' "$$tmp"; \
	SKILLHUB_CONFIG_DIR="$$tmp/config" \
	SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" \
	$(SKILLHUB) sources defaults add agent-rules; \
	SKILLHUB_CONFIG_DIR="$$tmp/config" \
	SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" \
	AGENT_SKILLS_DIR="$$tmp/skills" \
	$(SKILLHUB) tui
