.PHONY: build build-image test lint clean

build:
	go build -o bin/universe ./cmd/universe

build-image:
	docker build -t universe-base:latest ./container

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
