BINARY := universe
BUILD_DIR := bin
IMAGE := universe-base:latest

.PHONY: build build-image test lint clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/universe

build-image:
	docker build -t $(IMAGE) ./container

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
