# Variables
OUTPUT_DIR := ./output
FRONTEND_DIR := ./web
DOCKER_IMAGE := chartpress-server:0.1

.PHONY: all clean build-api build-web chart wire-do

# Default target
all: clean build-api

CHART_DIR := ./chart
TESTS_DIR := ./tests
TF_DIR := ./infra/terraform
NAMESPACE ?= chartpress


# Clean target: removes output directories and npm artifacts
clean:
	@echo "Cleaning output directories and npm artifacts..."
	@rm -rf ./output
	@mkdir -p ./output
	@touch ./output/.gitkeep
	@rm -rf $(FRONTEND_DIR)/node_modules
	@rm -rf $(FRONTEND_DIR)/build $(FRONTEND_DIR)/dist
	@find $(FRONTEND_DIR) -name "*.log" -type f -delete
	@echo "Clean complete."

# Build target: builds the Docker image
build-api:
	@echo "Building Docker image..."
	@docker build -t chartpress-api:0.1 .
	@echo "Docker image built: chartpress-api:0.1"

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

chart-reinstall:
	@helm uninstall -n chartpress-test chartpress
	@helm install -n chartpress-test chartpress  -f chart/values.yaml chart &&  sleep 3
	@kubectl port-forward -n chartpress-test svc/chartpress-frontend 8080:80

tests:
	@echo "Running Makefile in $(TESTS_DIR)..."
	@$(MAKE) -C $(TESTS_DIR)
	@echo "Makefile in $(TESTS_DIR) executed successfully."

# wire-do: bridge the terraform outputs (infra/terraform) into the cluster by
# creating/updating the chartpress-s3 Secret the chart's s3.existingSecret
# consumes. Idempotent (apply-from-dry-run). Override the target namespace with
# NAMESPACE=<ns>. Deploy afterward with: -f chart/values-do.yaml
wire-do:
	@command -v terraform >/dev/null || { echo "terraform not found"; exit 1; }
	@command -v kubectl   >/dev/null || { echo "kubectl not found"; exit 1; }
	@echo "Reading Spaces credentials from $(TF_DIR) outputs..."
	@ACCESS_KEY=$$(terraform -chdir=$(TF_DIR) output -raw access_key) ; \
	 SECRET_KEY=$$(terraform -chdir=$(TF_DIR) output -raw secret_key) ; \
	 BUCKET=$$(terraform -chdir=$(TF_DIR) output -raw bucket_name) ; \
	 echo "Creating Secret chartpress-s3 in namespace $(NAMESPACE) (bucket: $$BUCKET)" ; \
	 kubectl create secret generic chartpress-s3 \
	   --namespace $(NAMESPACE) \
	   --from-literal=access-key=$$ACCESS_KEY \
	   --from-literal=secret-key=$$SECRET_KEY \
	   --dry-run=client -o yaml | kubectl apply -f -
	@echo "Done. Deploy: helm upgrade --install chartpress ./chart -n $(NAMESPACE) -f chart/values-do.yaml"
