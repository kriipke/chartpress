#!/usr/bin/env bash
#
# Render the chart and validate every manifest against the Kubernetes schemas.
# Requires: helm, kubeconform (https://github.com/yannh/kubeconform)
set -euo pipefail

cd "$(dirname "$0")/.."

command -v helm >/dev/null 2>&1 || { echo "helm is required but not installed" >&2; exit 1; }
command -v kubeconform >/dev/null 2>&1 || { echo "kubeconform is required but not installed" >&2; exit 1; }

mkdir -p tests/generated

echo "Rendering chart with 'helm template'..."
helm template chartpress-test . > tests/generated/manifests.yaml

echo "Validating rendered manifests with kubeconform..."
kubeconform -strict -summary -ignore-missing-schemas tests/generated/manifests.yaml

echo "Done."
