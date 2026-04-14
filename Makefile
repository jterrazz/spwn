.PHONY: build install uninstall \
        build-test-image \
        test test-world test-agent test-foundation test-messenger test-imagebuilder \
        test-e2e test-e2e-world test-e2e-imagebuilder \
        test-ts test-ui test-ui-headed \
        web-build web-dev web-lint \
        lint clean docs

INSTALL_DIR ?= $(HOME)/.local/bin
PATH_EXPORT := export PATH="$$HOME/.local/bin:$$PATH"

# Build
build:
	cd apps/cli && go build -o ../../bin/spwn ./cmd/spwn

# Install - builds from source and installs to ~/.local/bin (same as install.sh)
install: build
	@mkdir -p $(INSTALL_DIR)
	@cp bin/spwn $(INSTALL_DIR)/spwn
	@chmod +x $(INSTALL_DIR)/spwn
	@codesign -s - $(INSTALL_DIR)/spwn 2>/dev/null || true
	@# Ensure ~/.local/bin is in PATH
	@case ":$$PATH:" in \
		*":$(INSTALL_DIR):"*) ;; \
		*) \
			ADDED=false; \
			for rc in "$$HOME/.zshrc" "$$HOME/.bashrc" "$$HOME/.bash_profile" "$$HOME/.profile"; do \
				if [ -f "$$rc" ]; then \
					if ! grep -q '.local/bin' "$$rc" 2>/dev/null; then \
						echo "" >> "$$rc"; \
						echo "# Added by spwn (make install)" >> "$$rc"; \
						echo '$(PATH_EXPORT)' >> "$$rc"; \
						echo "  Added ~/.local/bin to PATH in $$(basename $$rc)"; \
					fi; \
					ADDED=true; \
					break; \
				fi; \
			done; \
			if [ "$$ADDED" = false ]; then \
				echo "" >> "$$HOME/.profile"; \
				echo "# Added by spwn (make install)" >> "$$HOME/.profile"; \
				echo '$(PATH_EXPORT)' >> "$$HOME/.profile"; \
				echo "  Added ~/.local/bin to PATH in .profile"; \
			fi; \
			export PATH="$(INSTALL_DIR):$$PATH"; \
		;; \
	esac
	@echo ""
	@echo "  ✓ spwn installed to $(INSTALL_DIR)/spwn"
	@echo ""
	@echo "  Get started:"
	@echo "    spwn init"
	@echo "    spwn agent new neo"
	@echo "    spwn up --agent neo -w ."
	@echo ""

# Uninstall
uninstall:
	@rm -f $(INSTALL_DIR)/spwn
	@echo "  ✓ spwn removed from $(INSTALL_DIR)"

build-test-image:
	docker build -t spwn-test:latest -f tests/fixtures/Dockerfile.test ./tests/fixtures/mock-claude

# Unit tests (per domain)
test:
	cd packages/foundation && go test ./...
	cd packages/imagebuilder && go test ./...
	cd packages/mind && go test ./...
	cd packages/messenger && go test ./...
	cd packages/world && go test ./...
	cd apps/cli && go test ./...

test-foundation:
	cd packages/foundation && go test -v ./...

test-agent:
	cd packages/mind && go test -v ./...

test-messenger:
	cd packages/messenger && go test -v ./...

test-world:
	cd packages/world && go test -v ./...

test-cli:
	cd apps/cli && go test -v ./...

test-imagebuilder:
	cd packages/imagebuilder && go test -v ./...

# E2E tests (Docker required)
test-e2e: build-test-image
	cd packages/world && go test -v -tags=e2e -timeout=5m ./tests/e2e/...

test-e2e-world: build-test-image
	cd packages/world && go test -v -tags=e2e -timeout=5m ./tests/e2e/...

test-e2e-imagebuilder:
	cd packages/imagebuilder && go test -v -tags=e2e -timeout=10m ./e2e/...

# TS E2E (vitest, Docker required)
test-ts: build build-test-image
	pnpm -C tests test

# UI E2E tests (Docker + browser required)
test-ui: build
	pnpm -C tests test:ui

test-ui-headed: build
	pnpm -C tests test:ui:headed

# Web UI (Next.js + Tauri)
web-build:
	pnpm -C apps/web build

web-dev:
	pnpm -C apps/web dev

web-lint:
	pnpm -C apps/web lint

# Lint
lint: web-lint
	cd packages/foundation && go vet ./...
	cd packages/imagebuilder && go vet ./...
	cd packages/mind && go vet ./...
	cd packages/messenger && go vet ./...
	cd packages/world && go vet ./...
	cd apps/cli && go vet ./...

# Docs
docs:
	cd apps/cli && go run ./cmd/gen-docs ../../docs/cli

# Clean
clean:
	rm -rf bin/
