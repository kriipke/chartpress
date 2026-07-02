.PHONY: deps template lint test

deps: ## Fetch infrastructure dependencies declared in Chart.yaml (helm dependency build)
	helm dependency build

template: ## Render all manifests to stdout
	helm template chartpress-test .

lint: ## Run helm lint
	helm lint .

test: ## Render and validate manifests with kubeconform
	./tests/validate-templates.sh

# If this chart declares infrastructure dependencies (see Chart.yaml
# `dependencies:` and docs/best-practices.adoc), resolve any TODO(chartpress)
# stub repositories/versions first, then run `make deps` before `make template`
# / `make test` — Helm will not render a chart whose dependencies aren't fetched.
