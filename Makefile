.PHONY: build build-image build-test-image build-gate \
        test test-universe test-agent test-gate test-foundation test-messenger \
        test-e2e test-e2e-universe test-e2e-agent \
        lint clean docs

# Build
build:
	cd apps/cli && go build -o ../../bin/spwn ./cmd/spwn

build-image:
	docker build -t spwn-base:latest ./platform/images

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
