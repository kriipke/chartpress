# Variables
HELM_RELEASE := chartpress
HELM_CHART_DIR := .
NAMESPACE := chartpress-test
KUBECONFIG := ~/.kube/config

.PHONY: all install upgrade uninstall clean

# Default target
all: install

# Install the Helm chart
install:
	@echo "Installing Helm chart..."
	@kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@helm install $(HELM_RELEASE) $(HELM_CHART_DIR) --namespace $(NAMESPACE)
	@echo "Helm chart installed successfully."

# Upgrade the Helm chart
upgrade:
	@echo "Upgrading Helm chart..."
	@helm upgrade $(HELM_RELEASE) $(HELM_CHART_DIR) --namespace $(NAMESPACE)
	@echo "Helm chart upgraded successfully."

# Uninstall the Helm chart
uninstall:
	@echo "Uninstalling Helm chart..."
	@helm uninstall $(HELM_RELEASE) --namespace $(NAMESPACE)
	@kubectl delete namespace $(NAMESPACE) --wait
	@echo "Helm chart uninstalled successfully."

# Clean up resources
clean: uninstall
	@echo "Cleaning up resources..."
	@kubectl delete all --all --namespace $(NAMESPACE) --ignore-not-found
	@echo "Resources cleaned up."
