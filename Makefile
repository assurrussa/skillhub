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
	@tmp=$$(mktemp -d); \
	project="$$tmp/project"; \
	mkdir -p "$$project"; \
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
	test -f "$$tmp/skills/rules-selector/SKILL.md"; \
	test -f "$$tmp/skills/rules-selector/.skillhub.json"; \
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
