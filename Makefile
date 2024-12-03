BUILD_DIR := ./bin
BUILD_FILE := $(BUILD_DIR)/app

# Default target
all: build run

# Build for Linux x86_64
build:
	@echo "Building the application for Linux x86_64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_FILE) ./cmd/app
	@echo "Build complete. Binary saved to $(BUILD_FILE)"

# Run the application
run: build
	@echo "Running the application..."
	@$(BUILD_FILE)

# Clean build files
clean:
	@echo "Cleaning build files..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete."

.PHONY: build run clean all
