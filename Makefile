.PHONY: build run test lint fmt install clean generate check-generate snapshot

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

# Generate typed API client from OpenAPI spec
generate:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config internal/client/oapi_codegen_config.yml api/openapi.yaml

# CI check: generated code matches committed code
check-generate: generate
	@if [ -n "$$(git diff --name-only)" ]; then \
		echo "Generated code is out of date. Run 'make generate' and commit."; \
		git diff; \
		exit 1; \
	fi

# Snapshot release (test goreleaser locally without publishing)
snapshot:
	goreleaser release --snapshot --clean
