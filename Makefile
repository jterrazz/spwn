.PHONY: build install uninstall \
        build-image build-architect-image build-test-image build-gate \
        test test-universe test-agent test-gate test-foundation test-messenger \
        test-e2e test-e2e-universe test-e2e-agent \
        lint clean docs

INSTALL_DIR ?= $(HOME)/.local/bin
PATH_EXPORT := export PATH="$$HOME/.local/bin:$$PATH"

# Build
build:
	cd apps/cli && go build -o ../../bin/spwn ./cmd/spwn

# Install — builds from source and installs to ~/.local/bin (same as install.sh)
install: build
	@mkdir -p $(INSTALL_DIR)
	@cp bin/spwn $(INSTALL_DIR)/spwn
	@chmod +x $(INSTALL_DIR)/spwn
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
	@echo "    spwn agent init neo"
	@echo "    spwn world --agent neo -w ."
	@echo ""

# Uninstall
uninstall:
	@rm -f $(INSTALL_DIR)/spwn
	@echo "  ✓ spwn removed from $(INSTALL_DIR)"

build-image:
	docker build -t spwn-base:latest ./platform/images

build-architect-image: build
	docker build -t spwn-architect:latest -f platform/images/Dockerfile.architect .

build-test-image:
	docker build -t spwn-test:latest -f platform/images/Dockerfile.test ./platform/fixtures/mock-claude

build-gate:
	cd platform/gate-runtime && cargo build --release

# Unit tests (per domain)
test:
	cd core/foundation && go test ./...
	cd core/agent && go test ./...
	cd core/gate && go test ./...
	cd core/messenger && go test ./...
	cd core/universe && go test ./...
	cd apps/cli && go test ./...

test-foundation:
	cd core/foundation && go test -v ./...

test-agent:
	cd core/agent && go test -v ./...

test-gate:
	cd core/gate && go test -v ./...

test-messenger:
	cd core/messenger && go test -v ./...

test-universe:
	cd core/universe && go test -v ./...

test-cli:
	cd apps/cli && go test -v ./...

# E2E tests (Docker required)
test-e2e: build-test-image
	cd core/universe && go test -v -tags=e2e -timeout=5m ./tests/e2e/...

test-e2e-universe: build-test-image
	cd core/universe && go test -v -tags=e2e -timeout=5m ./tests/e2e/...

test-e2e-agent:
	cd core/agent && go test -v -tags=e2e -timeout=3m ./tests/e2e/...

# Lint
lint:
	cd core/foundation && go vet ./...
	cd core/agent && go vet ./...
	cd core/gate && go vet ./...
	cd core/messenger && go vet ./...
	cd core/universe && go vet ./...
	cd apps/cli && go vet ./...

# Docs
docs:
	cd apps/cli && go run ./cmd/gen-docs ../../docs/cli

# Clean
clean:
	rm -rf bin/
	cd platform/gate-runtime && cargo clean 2>/dev/null || true
