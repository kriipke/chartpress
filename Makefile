# Variables
OUTPUT_DIR := ./output
FRONTEND_DIR := ./web
DOCKER_IMAGE := chartpress-server:0.1

.PHONY: all clean build-api build-web chart

# Default target
all: clean build-api

CHART_DIR := ./chart
TESTS_DIR := ./tests


# Clean target: removes output directories and npm artifacts
clean:
	@echo "Cleaning output directories and npm artifacts..."
	@rm -rf ./output
	@echo > ./output/.gitkeep
	@rm -rf ./frontend/node_modules
	@rm -rf ./frontend/build
	@find ./frontend -name "*.log" -type f -delete
	@echo "Clean complete."

# Build target: builds the Docker image
build-api:
	@echo "Building Docker image..."
	@docker build -t chartpress-api:0.1 .
	@echo "Docker image built: chartpress-server:0.1"

# Build target: builds the Docker image
build-web:
	@echo "Building Docker image..."
	@docker build -t chartpress-server:0.1 ./web/
	@echo "Docker image built: chartpress-server:0.1"

# Chart target: runs the Makefile in the ./chart directory
chart:
	@echo "Running Makefile in $(CHART_DIR)..."
	@$(MAKE) -C $(CHART_DIR)
	@echo "Makefile in $(CHART_DIR) executed successfully."

tests:
	@echo "Running Makefile in $(TESTS_DIR)..."
	@$(MAKE) -C $(TESTS_DIR)
	@echo "Makefile in $(TESTS_DIR) executed successfully."
	@curl -X POST http://localhost:9090/generate \
  		-H "Content-Type: application/json" \
  		--data-binary ./tests/@chartpress.json
