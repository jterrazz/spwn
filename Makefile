.PHONY: build test test-unit test-integration test-clean vet

build:
	go build ./cmd/universe

vet:
	go vet ./...

test: vet
	go test ./...

test-unit: test

test-integration: build vet
	go test -tags=integration -v -count=1 -timeout 5m ./internal/backend/
	go test -tags=integration -v -count=1 -timeout 5m ./test/e2e/

test-clean:
	@echo "Removing orphan universe test containers..."
	@docker ps -a --filter label=universe.id --filter ancestor=alpine:3.19 -q | xargs -r docker rm -f 2>/dev/null || true
	@echo "Done."
