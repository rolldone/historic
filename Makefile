# Build the Historic CLI

.PHONY: build test vet quality benchmark

build:
	go build -o historic .

test:
	go test ./...

vet:
	go vet ./...

quality: test vet build

benchmark:
	go test ./internal/search -run '^$$' -bench BenchmarkFind1000Files -benchtime=1x
