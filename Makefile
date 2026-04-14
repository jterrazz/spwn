.PHONY: help build install uninstall clean \
        lint go-vet \
        test go-test test-pkg \
        test-e2e test-e2e-imagebuilder \
        test-ts test-web test-web-headed \
        build-test-image \
        web-build web-dev \
        docs

# Go modules are the single source of truth: every entry in go.work
# Gets linted and tested. Adding a new module to go.work is the only
# Thing needed to bring it under CI coverage — no Makefile edits.
GO_MODS := $(shell go work edit -json 2>/dev/null | jq -r '.Use[].DiskPath')

help:
	@echo "Common targets:"
	@echo "  make build               Build bin/spwn"
	@echo "  make install             Build and install to ~/.local/bin"
	@echo "  make lint                go vet all modules + pnpm -r lint"
	@echo "  make test                Go unit tests across the workspace"
	@echo "  make test-pkg PKG=mind   Verbose go test for one package"
	@echo "  make test-e2e            Go E2E (Docker required)"
	@echo "  make test-ts             TypeScript E2E (Docker + Node 22)"
	@echo "  make test-web            Playwright web E2E (Docker + browser)"
	@echo "  make web-dev             Run the Next.js dev server"
	@echo "  make docs                Regenerate apps/cli docs"
	@echo "  make clean               rm -rf bin/"

# ── Build ─────────────────────────────────────────────────────────

build:
	cd apps/cli && go build -o ../../bin/spwn ./cmd/spwn

install: build
	@scripts/install.sh

uninstall:
	@rm -f $${INSTALL_DIR:-$$HOME/.local/bin}/spwn
	@echo "  ✓ spwn removed"

clean:
	rm -rf bin/

build-test-image:
	docker build -t spwn-test:latest -f tests/fixtures/Dockerfile.test ./tests/fixtures/mock-claude

# ── Lint ──────────────────────────────────────────────────────────

lint: go-vet
	pnpm -r lint

go-vet:
	@for mod in $(GO_MODS); do \
		echo "==> go vet $$mod"; \
		(cd $$mod && go vet ./...) || exit 1; \
	done

# ── Test ──────────────────────────────────────────────────────────

test: go-test

go-test:
	@for mod in $(GO_MODS); do \
		echo "==> go test $$mod"; \
		(cd $$mod && go test ./...) || exit 1; \
	done

# Run verbose tests for a single package: `make test-pkg PKG=mind` or
# `make test-pkg PKG=apps/cli`. Path is resolved relative to repo root.
test-pkg:
	@if [ -z "$(PKG)" ]; then \
		echo "usage: make test-pkg PKG=<module-path-or-name>" >&2; \
		exit 1; \
	fi
	@if [ -d "packages/$(PKG)" ]; then cd packages/$(PKG) && go test -v ./...; \
	elif [ -d "$(PKG)" ]; then cd $(PKG) && go test -v ./...; \
	else echo "no such package: $(PKG)" >&2; exit 1; fi

# ── E2E ───────────────────────────────────────────────────────────

test-e2e: build-test-image
	cd packages/world && go test -v -tags=e2e -timeout=5m ./tests/e2e/...

test-e2e-imagebuilder:
	cd packages/image && go test -v -tags=e2e -timeout=10m ./e2e/...

# TypeScript E2E against the compiled spwn CLI (vitest + real Docker).
test-ts: build build-test-image
	pnpm -C tests test

# Playwright against the Next.js web UI.
test-web: build
	pnpm -C tests test:web

test-web-headed: build
	pnpm -C tests test:web:headed

# ── Web (apps/web) ────────────────────────────────────────────────

web-build:
	pnpm -C apps/web build

web-dev:
	pnpm -C apps/web dev

# ── Docs ──────────────────────────────────────────────────────────

docs:
	cd apps/cli && go run ./cmd/gen-docs ../../docs/cli
