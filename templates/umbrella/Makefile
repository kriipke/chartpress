.PHONY: template lint test

template: ## Render all manifests to stdout
	helm template chartpress-test .

lint: ## Run helm lint
	helm lint .

test: ## Render and validate manifests with kubeconform
	./tests/validate-templates.sh
