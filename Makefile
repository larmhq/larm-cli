.PHONY: build run test lint fmt install clean snapshot

# Build the binary
build:
	go build -o bin/larm .

# Build and run with args: make run ARGS="monitors list"
run: build
	./bin/larm $(ARGS)

# Run all tests
test:
	go test ./...

# Lint (golangci-lint)
lint:
	golangci-lint run

# Install to GOPATH/bin
install:
	go install .

# Format
fmt:
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	rm -rf bin/ dist/

# Snapshot release (test goreleaser locally without publishing)
snapshot:
	goreleaser release --snapshot --clean
