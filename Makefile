.DEFAULT_GOAL := help

# outside: Turborepo reads go.work and pnpm-workspace.yaml into one task graph, caches every result under
# .artifacts/turbo, and names the workspace as a whole `go-workspace`, a scope that owns no directory.
GOLANGCI_VERSION := v2.13.2
GOLANGCI         := $(shell go env GOPATH)/bin/golangci-lint
TURBO            := GOLANGCI_VERSION=$(GOLANGCI_VERSION) PATH="$(shell go env GOPATH)/bin:$$PATH" pnpm exec turbo
GO_PACKAGES      := 'go-workspace...'

.PHONY: help install build fix lint test mutation check
.PHONY: generate uninstall clean docs test-pkg test-image test-go-e2e test-compile-e2e test-smoke web-dev

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\n"} \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

install: ## Install both toolchains' dependencies
	go work sync
	pnpm install --frozen-lockfile

build: ## Build the spwn binary into .artifacts/go/spwn
	@$(TURBO) run build --filter=cli

generate: ## Run every //go:generate directive (refreshes the embedded catalog)
	@$(TURBO) run generate --filter=dependency

fix: ## Repair what a fixer can: every fixer of the TypeScript toolchain
	pnpm run lint:fix

lint: ## golangci-lint across go.work, then every gate of the TypeScript toolchain
	@$(GOLANGCI) --version 2>/dev/null | grep -q "$(GOLANGCI_VERSION:v%=%)" || \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	@$(TURBO) run lint --filter='!go-workspace'

test: build test-image ## Go unit tests, the web and gate module tests, the cli and website specs
	@$(TURBO) run test --filter='!go-workspace'

mutation: ## The mutation verdict over the change, member by member
	pnpm exec typescript mutation

check: lint test mutation ## Everything CI runs

test-pkg: generate ## Verbose go test for one package, e.g. make test-pkg PKG=agent
	@if [ -z "$(PKG)" ]; then echo "usage: make test-pkg PKG=<module-path-or-name>" >&2; exit 1; fi
	@if [ -d "packages/$(PKG)" ]; then cd packages/$(PKG) && go test -v ./...; \
	elif [ -d "$(PKG)" ]; then cd $(PKG) && go test -v ./...; \
	else echo "no such package: $(PKG)" >&2; exit 1; fi

test-image: ## Build spwn-test:latest, the base image holding the mock Claude and Codex runtimes
	docker build --quiet -t spwn-test:latest -f docker/spwn-test/Dockerfile docker/spwn-test

test-go-e2e: generate test-image ## Go world E2E (//go:build e2e), Docker required
	cd packages/world && go test -v -tags=e2e -timeout=30m ./tests/e2e/...
	cd packages/container && go test -v -tags=e2e -timeout=5m ./backend/...

test-compile-e2e: generate ## Go image-build E2E (compile and Dockerfile rendering), Docker required
	cd packages/compile && go test -v -tags=e2e -timeout=15m ./e2e/...

test-smoke: build ## Real image builds: spwn init, up, tool probe (~10 minutes cold)
	SPWN_SMOKE=1 pnpm -C specs exec vitest --run --project cli

install-binary: build ## Install the built binary into ~/.local/bin
	@scripts/install.sh

uninstall: ## Remove the installed binary
	@rm -f $${INSTALL_DIR:-$$HOME/.local/bin}/spwn

clean: ## Remove .artifacts/, the turbo cache included
	rm -rf .artifacts/

docs: generate ## Regenerate docs/reference/commands.md from the Cobra tree
	cd apps/cli && go run ./cmd/gen-docs ../../docs/reference/commands.md

web-dev: ## Next.js dev server
	@pnpm -C apps/web dev
