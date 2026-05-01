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
# thing needed to bring it under CI coverage — no Makefile edits.
GO_MODS := $(shell go work edit -json 2>/dev/null | jq -r '.Use[].DiskPath')

.PHONY: help
help:  ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next } \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ Build

.PHONY: build install uninstall clean generate docs

generate:  ## Run every //go:generate directive (refreshes the embedded catalog)
	@cd packages/dependency && go generate ./...

build: generate  ## Build bin/spwn
	cd apps/cli && go build -o ../../bin/spwn ./cmd/spwn

install: build  ## Build and install to ~/.local/bin
	@scripts/install.sh

uninstall:  ## Remove the installed bin/spwn
	@rm -f $${INSTALL_DIR:-$$HOME/.local/bin}/spwn
	@echo "  ✓ spwn removed"

clean:  ## rm -rf bin/
	rm -rf bin/

docs:  ## Regenerate docs/cli from Cobra
	cd apps/cli && go run ./cmd/gen-docs ../../docs/cli

##@ Lint

.PHONY: lint
lint: generate  ## go vet across go.work + pnpm -r lint (oxlint + oxfmt + knip)
	@for mod in $(GO_MODS); do \
		echo "==> go vet $$mod"; \
		(cd $$mod && go vet ./...) || exit 1; \
	done
	@pnpm -r lint

##@ Test — fast (no Docker)

.PHONY: test test-pkg test-contracts test-web-unit test-gate-node

test: generate  ## Go unit tests across the workspace (~5s)
	@for mod in $(GO_MODS); do \
		echo "==> go test $$mod"; \
		(cd $$mod && go test ./...) || exit 1; \
	done

test-pkg: generate  ## Verbose go test for one package — usage: make test-pkg PKG=agent
	@if [ -z "$(PKG)" ]; then \
		echo "usage: make test-pkg PKG=<module-path-or-name>" >&2; \
		exit 1; \
	fi
	@if [ -d "packages/$(PKG)" ]; then cd packages/$(PKG) && go test -v ./...; \
	elif [ -d "$(PKG)" ]; then cd $(PKG) && go test -v ./...; \
	else echo "no such package: $(PKG)" >&2; exit 1; fi

test-contracts:  ## Static checks that every surface declared its tests
	@node tests/_contracts/assert-contracts.mjs

test-web-unit:  ## apps/web vitest (MSW-mocked network, ~1s)
	@pnpm -C apps/web test

test-gate-node:  ## apps/gate vitest (sidecar + SDK, ~1s)
	@pnpm -C apps/gate test

##@ Test — Docker required

.PHONY: test-image test-go-e2e test-compile-e2e test-cli test-smoke test-web test-web-headed

test-image:  ## Build spwn-test:latest (mock Claude/Codex runtimes)
	docker build -t spwn-test:latest -f tests/_simulators/Dockerfile.test ./tests/_simulators

test-go-e2e: generate test-image  ## Go world E2E (//go:build e2e) — Architect/world/container
	cd packages/world && go test -v -tags=e2e -timeout=30m ./tests/e2e/...

test-compile-e2e: generate  ## Go image-build E2E (compile + Dockerfile rendering)
	cd packages/compile && go test -v -tags=e2e -timeout=10m ./e2e/...

test-cli: build test-image  ## TypeScript CLI E2E against compiled bin/spwn (vitest)
	pnpm -C tests test

test-smoke: build  ## Real-build smoke: spwn init → up → tool probe (~10min cold)
	pnpm -C tests test:smoke

test-web: build test-image  ## Playwright web E2E (real Next.js + Go API + Chromium)
	pnpm -C tests test:web

test-web-headed: build  ## Playwright in headed mode (visual debugging)
	pnpm -C tests test:web:headed

##@ Web (apps/web)

.PHONY: web-build web-dev

web-build:  ## Production Next.js build
	@pnpm -C apps/web build

web-dev:  ## Next.js dev server
	@pnpm -C apps/web dev
