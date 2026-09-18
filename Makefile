# Build the Historic CLI

.PHONY: build test

build:
	go build -o historic .

test:
	go test ./...
