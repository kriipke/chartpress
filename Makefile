# SPDX-License-Identifier: Apache-2.0
# Canonical developer and CI entry points.
OUTPUT_DIR := ./output
FRONTEND_DIR := ./web
DOCKER_IMAGE := chartpress-server:0.1
GO ?= go
NPM ?= npm
HELM ?= helm
GO_TEST_COUNT ?= 1

.PHONY: all clean build-api build-web chart wire-do tests test-go vet test-web \
	build-web-app lint-chart test-templates smoke verify verify-offline \
	check-clean verify-fixtures update-chart-fixture license-check

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

test-go:
	@echo "==> Go tests"
	@GOTOOLCHAIN=local $(GO) test -count=$(GO_TEST_COUNT) ./...

vet:
	@echo "==> Go vet"
	@GOTOOLCHAIN=local $(GO) vet ./...

test-web:
	@echo "==> Web tests"
	@cd $(FRONTEND_DIR) && CI=1 $(NPM) test

build-web-app:
	@echo "==> Web production build"
	@cd $(FRONTEND_DIR) && $(NPM) run build

lint-chart:
	@echo "==> Helm lint"
	@$(HELM) lint ./chart --set backend.openai.apiKeySecret.name=verify-placeholder

test-templates:
	@echo "==> Generated template validation"
	@./templates/umbrella/tests/validate-templates.sh

license-check:
	@echo "==> Dependency licenses"
	@GOTOOLCHAIN=local $(GO) run ./tools/licensecheck

verify-fixtures:
	@unzip -t tests/chart.zip >/dev/null
	@test "$$(find . -type f -name '*.zip' -not -path './.git/*' -print | wc -l | tr -d ' ')" = "1" || { echo "unexpected checked-in ZIP archive" >&2; exit 1; }

update-chart-fixture:
	@CHARTPRESS_UPDATE_GOLDEN=1 GOTOOLCHAIN=local $(GO) test ./internal/operator -run TestGoldenChartFixture -count=1

smoke:
	@$(MAKE) -C $(TESTS_DIR) test-curl

verify: vet test-go test-web build-web-app lint-chart test-templates verify-fixtures license-check
	@GOTOOLCHAIN=local $(GO) mod verify
	@echo "All required verification checks passed."

verify-offline:
	@GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off npm_config_offline=true \
		npm_config_audit=false npm_config_fund=false npm_config_update_notifier=false \
		$(MAKE) verify

check-clean:
	@git diff --exit-code -- .
	@git diff --cached --exit-code -- .
	@test -z "$$(git ls-files --others --exclude-standard)" || { \
		echo "verification left untracked files:" >&2; \
		git ls-files --others --exclude-standard >&2; exit 1; }

# Chart target: runs the Makefile in the ./chart directory
chart:
	@echo "Running Makefile in $(CHART_DIR)..."
	@$(MAKE) -C $(CHART_DIR)
	@echo "Makefile in $(CHART_DIR) executed successfully."

chart-reinstall:
	@helm uninstall -n chartpress-test chartpress
	@helm install -n chartpress-test chartpress  -f chart/values.yaml chart &&  sleep 3
	@kubectl port-forward -n chartpress-test svc/chartpress-frontend 8080:80

tests: test-go
	@echo "'make tests' is retained as an alias for unit/regression tests; use 'make smoke' for the live server check."

smoke-legacy:
	@echo "Running Makefile in $(TESTS_DIR)..."
	@$(MAKE) -C $(TESTS_DIR)
	@echo "Makefile in $(TESTS_DIR) executed successfully."

# wire-do: bridge the terraform outputs (infra/terraform) into the cluster by
# creating/updating the chartpress-s3 Secret the chart's s3.existingSecret
# consumes. Idempotent (apply-from-dry-run). Override the target namespace with
# NAMESPACE=<ns>. Deploy afterward with: -f chart/values-do.yaml -f
# chart/values-do.generated.yaml (the latter is written by this target).
wire-do:
	@command -v terraform >/dev/null || { echo "terraform not found"; exit 1; }
	@command -v kubectl   >/dev/null || { echo "kubectl not found"; exit 1; }
	@echo "Reading Spaces credentials + bucket/region/endpoint from $(TF_DIR) outputs..."
	@ACCESS_KEY=$$(terraform -chdir=$(TF_DIR) output -raw access_key 2>/dev/null) || true ; \
	 SECRET_KEY=$$(terraform -chdir=$(TF_DIR) output -raw secret_key 2>/dev/null) || true ; \
	 BUCKET=$$(terraform -chdir=$(TF_DIR) output -raw bucket_name 2>/dev/null) || true ; \
	 REGION=$$(terraform -chdir=$(TF_DIR) output -raw region 2>/dev/null) || true ; \
	 ENDPOINT=$$(terraform -chdir=$(TF_DIR) output -raw endpoint 2>/dev/null) || true ; \
	 if [ -z "$$ACCESS_KEY" ] || [ -z "$$SECRET_KEY" ] || [ -z "$$BUCKET" ] || [ -z "$$REGION" ] || [ -z "$$ENDPOINT" ]; then \
	   echo "ERROR: terraform outputs missing/empty (need access_key + secret_key + bucket_name + region + endpoint)." ; \
	   echo "       Run 'terraform apply' in $(TF_DIR) first — the Spaces bucket AND scoped key must exist." ; \
	   echo "       (bucket_name/region/endpoint can resolve from a FAILED apply; the key outputs cannot.)" ; \
	   exit 1 ; \
	 fi ; \
	 echo "Creating Secret chartpress-s3 in namespace $(NAMESPACE) (bucket: $$BUCKET)" ; \
	 kubectl create secret generic chartpress-s3 \
	   --namespace $(NAMESPACE) \
	   --from-literal=access-key=$$ACCESS_KEY \
	   --from-literal=secret-key=$$SECRET_KEY \
	   --dry-run=client -o yaml | kubectl apply -f - ; \
	 { echo "# Generated by 'make wire-do' from infra/terraform outputs — do not edit." ; \
	   echo "# Layer AFTER chart/values-do.yaml so the terraform-provisioned bucket/region/" ; \
	   echo "# endpoint always win, for any env (prod, staging, pr-*)." ; \
	   echo "s3:" ; \
	   echo "  bucket: $$BUCKET" ; \
	   echo "  region: $$REGION" ; \
	   echo "  endpoint: $$ENDPOINT" ; \
	 } > chart/values-do.generated.yaml ; \
	 echo "Wrote chart/values-do.generated.yaml (bucket=$$BUCKET region=$$REGION endpoint=$$ENDPOINT)"
	@echo "Done. Deploy: helm upgrade --install chartpress ./chart -n $(NAMESPACE) -f chart/values-do.yaml -f chart/values-do.generated.yaml"
