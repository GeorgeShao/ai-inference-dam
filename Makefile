.PHONY: build run test clean dev api web generate

# Build the server
build: generate
	go build -o bin/server ./cmd/server

# Run the server
run: build
	./bin/server

# Run both API and web in development mode
dev:
	make -j2 api web

# Run API server only
api:
	go run ./cmd/server

# Run web frontend only
web:
	cd web && pnpm run dev

# Generate sqlc code
generate:
	sqlc generate

# Generate TypeScript types from Go types
generate-types:
	go run github.com/gzuidhof/tygo@latest generate

# Install development tools
tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/gzuidhof/tygo@latest

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf data/

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Tidy dependencies
tidy:
	go mod tidy
