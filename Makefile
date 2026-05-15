SHELL := /bin/sh

.DEFAULT_GOAL := verify
GO ?= go
SKILLHUB ?= $(GO) run ./cmd/skillhub
SKILLHUB_DEV ?= $(GO) run ./cmd/skillhub-dev
AGENT_RULES_PATH ?= ../agent-rules

.PHONY: help check check-all verify smoke-temp tui-temp test vet diff-check tidy generate fmt lint test-race bench-all cover-html maintainer-check

help:
	@printf '%s\n' \
		'Targets:' \
		'  make check       Run Go dev verification' \
		'  make check-all   Alias for make verify' \
		'  make test        Run Go tests' \
		'  make vet         Run go vet' \
		'  make diff-check  Run git diff --check' \
		'  make verify      Run Go dev verification' \
		'  make smoke-temp  Test default source and temp install without user config' \
		'  make tui-temp    Open TUI with temp config and temp install dir' \
		'  make maintainer-check  Run local Unix maintenance checks' \
		'' \
		'Variables:' \
		'  AGENT_RULES_PATH=../agent-rules' \
		'  SKILLHUB="$(GO) run ./cmd/skillhub"' \
		'  SKILLHUB_DEV="$(GO) run ./cmd/skillhub-dev"'


check check-all verify:
	SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" $(SKILLHUB_DEV) verify

maintainer-check: tidy generate fmt vet lint test test-race cover-html

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

test-race:
	$(GO) test -race -count=5 ./...

vet:
	$(GO) vet ./...

diff-check:
	git diff --check

generate:
	$(GO) generate ./...

fmt:
	$(GO) fmt ./...
	gofumpt -l -w .
	gci write -s standard -s default -s "prefix($$($(GO) list -m))" .

lint:
	golangci-lint run -v --fix --timeout=5m ./...

bench-all:
	$(GO) test -bench=. -benchmem ./...

cover-html:
	@$(GO) test -coverprofile=./coverage.text -covermode=atomic $(shell go list ./...)
	@$(GO) tool cover -html=./coverage.text -o ./cover.html && rm ./coverage.text

smoke-temp:
	SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" $(SKILLHUB_DEV) smoke-temp

tui-temp:
	SKILLHUB_AGENT_RULES_PATH="$(AGENT_RULES_PATH)" $(SKILLHUB_DEV) tui-temp
