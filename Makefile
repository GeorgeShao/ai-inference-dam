.PHONY: build run test clean dev generate

# Build the server
build: generate
	go build -o bin/server ./cmd/server

# Run the server
run: build
	./bin/server

# Run in development mode
dev:
	go run ./cmd/server

# Generate sqlc code
generate:
	sqlc generate

# Install development tools
tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

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
