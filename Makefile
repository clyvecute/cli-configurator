# Build Logic
BINARY_NAME=sentinel.exe
BUILD_DIR=bin

.PHONY: all clean test build run-server run-cli docker-build

all: clean test build

clean:
	@echo "Cleaning..."
	@if exist $(BUILD_DIR) rmdir /s /q $(BUILD_DIR)
	@if exist server-bin del server-bin

test:
	@echo "Running tests..."
	@go test -v ./...

build:
	@echo "Building binaries..."
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/server.exe ./cmd/server
	@go build -o $(BUILD_DIR)/cli.exe ./cmd/cli

run-server: build
	@echo "Starting server..."
	@set CONFIG_LINTER_API_KEY=dev-key && $(BUILD_DIR)/server.exe

run-cli: build
	@echo "Running CLI example..."
	@$(BUILD_DIR)/cli.exe -help

docker-build:
	@echo "Building Docker image..."
	@docker build -t sentinel-app .
