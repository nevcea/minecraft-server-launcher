.PHONY: build test clean run install

build:
	go build -ldflags="-s -w" -o paper-launcher.exe .

build-all:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o paper-launcher-windows-amd64.exe .
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o paper-launcher-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o paper-launcher-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o paper-launcher-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o paper-launcher-darwin-arm64 .

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f paper-launcher paper-launcher.exe paper-launcher-*
	go clean -cache

run:
	go run .

install:
	go install .

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

deps:
	go mod download
	go mod tidy

help:
	@echo "Available targets:"
	@echo "  build        - Build the project (fast!)"
	@echo "  build-all    - Build for all platforms"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  clean        - Clean build artifacts"
	@echo "  run          - Run the project"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run linter"
	@echo "  deps         - Download and tidy dependencies"

