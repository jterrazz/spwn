.PHONY: build build-image build-test-image build-gate test test-e2e lint clean

build:
	go build -o bin/universe ./cmd/universe

build-image:
	docker build -t universe-base:latest ./container

build-test-image:
	docker build -t universe-test:latest -f container/Dockerfile.test ./__tests__/mock

build-gate:
	cd container/gate && cargo build --release

test:
	go test ./...

test-e2e: build-test-image
	go test -v -tags=e2e -timeout=5m ./__tests__/e2e/...

lint:
	go vet ./...

clean:
	rm -rf bin/
	cd container/gate && cargo clean 2>/dev/null || true
