# spwn — top-level orchestration.
#
# CI calls these targets directly; the workflow file at
# .github/workflows/validate.yaml is the canonical aggregate. There is
# no "test-pr" or "test-release" meta-target by design — if you want
# to know what CI runs, read validate.yaml.
#
# Help is auto-generated from `## comment` markers after the colon, so
# `make` (or `make help`) never goes stale. Sections are introduced
# by `##@ Section Name` lines below.

.DEFAULT_GOAL := help

# Go modules are the single source of truth: every entry in go.work
# gets linted and tested. Adding a new module to go.work is the only
# thing needed to bring it under CI coverage — Turborepo reads go.work
# itself and turns each module into a package, so no Makefile edits.
#
# One task graph spans both toolchains: `turbo` runs the Go modules and
# the pnpm workspace together, caches every result under .artifacts/turbo,
# and skips what has not changed. `turbo.json` holds the wiring.
# `go-workspace` is Turborepo's synthetic scope for the workspace as a
# whole; it owns no directory, so a command-carrying task must exclude it.
GOLANGCI_VERSION := v2.13.2
GOLANGCI         := $(shell go env GOPATH)/bin/golangci-lint
TURBO            := GOLANGCI_VERSION=$(GOLANGCI_VERSION) PATH="$(shell go env GOPATH)/bin:$$PATH" pnpm exec turbo
GO_PACKAGES      := 'go-workspace...'

.PHONY: help
help:  ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next } \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ Build

.PHONY: build install uninstall clean generate docs

generate:  ## Run every //go:generate directive (refreshes the embedded catalog)
	@$(TURBO) run generate --filter=dependency

build:  ## Build .artifacts/go/spwn
	@$(TURBO) run build --filter=cli

install: build  ## Build and install to ~/.local/bin
	@scripts/install.sh

uninstall:  ## Remove the installed spwn
	@rm -f $${INSTALL_DIR:-$$HOME/.local/bin}/spwn
	@echo "  ✓ spwn removed"

clean:  ## rm -rf .artifacts/ (the turbo cache lives there too)
	rm -rf .artifacts/

docs: generate  ## Regenerate docs/reference from Cobra
	cd apps/cli && go run ./cmd/gen-docs ../../docs/reference

##@ Lint

.PHONY: lint docs-layout
lint: docs-layout  ## golangci-lint across go.work + the web/specs quality gates + docs layout
	@$(GOLANGCI) --version 2>/dev/null | grep -q "$(GOLANGCI_VERSION:v%=%)" || \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	@$(TURBO) run lint --filter='!go-workspace'

# pnpm, not npx: npm's ephemeral install dies on `edgesOut` of null when it
# resolves this package on the CI runner, with or without --package=. The lint
# job already sets pnpm up for the workspace half, so there is nothing to add.
docs-layout:  ## Check docs/ against the estate's manual spine
	@echo "==> docs layout"
	@pnpm --package=@jterrazz/typescript@10.1.6 dlx typescript docs-layout .

##@ Test — fast (no Docker)

.PHONY: test test-pkg test-contracts test-web-unit test-gate-node

test:  ## Go unit tests across the workspace (~5s)
	@$(TURBO) run test --filter=$(GO_PACKAGES)

test-pkg: generate  ## Verbose go test for one package — usage: make test-pkg PKG=agent
	@if [ -z "$(PKG)" ]; then \
		echo "usage: make test-pkg PKG=<module-path-or-name>" >&2; \
		exit 1; \
	fi
	@if [ -d "packages/$(PKG)" ]; then cd packages/$(PKG) && go test -v ./...; \
	elif [ -d "$(PKG)" ]; then cd $(PKG) && go test -v ./...; \
	else echo "no such package: $(PKG)" >&2; exit 1; fi

test-contracts:  ## Static checks that every surface declared its tests
	@node specs/_contracts/assert-contracts.mjs

test-web-unit:  ## apps/web vitest (MSW-mocked network, ~1s)
	@$(TURBO) run test --filter=web

test-gate-node:  ## apps/gate vitest (sidecar + SDK, ~1s)
	@$(TURBO) run test --filter=spwn-gate

##@ Test — Docker required

.PHONY: test-image test-go-e2e test-compile-e2e test-cli test-smoke test-web test-web-headed

test-image:  ## Build spwn-test:latest (mock Claude/Codex runtimes)
	docker build -t spwn-test:latest -f specs/_simulators/Dockerfile.test ./specs/_simulators

test-go-e2e: generate test-image  ## Go world E2E (//go:build e2e) — Architect/world/container
	cd packages/world && go test -v -tags=e2e -timeout=30m ./tests/e2e/...

test-compile-e2e: generate  ## Go image-build E2E (compile + Dockerfile rendering)
	cd packages/compile && go test -v -tags=e2e -timeout=15m ./e2e/...

test-cli: build test-image  ## TypeScript CLI E2E against the compiled binary (vitest)
	pnpm -C specs test

test-smoke: build  ## Real-build smoke: spwn init → up → tool probe (~10min cold)
	pnpm -C specs test:smoke

test-web: build test-image  ## Playwright web E2E (real Next.js + Go API + Chromium)
	pnpm -C specs test:web

test-web-headed: build  ## Playwright in headed mode (visual debugging)
	pnpm -C specs test:web:headed

##@ Web (apps/web)

.PHONY: web-build web-dev

web-build:  ## Production Next.js build
	@$(TURBO) run build --filter=web

web-dev:  ## Next.js dev server
	@pnpm -C apps/web dev
