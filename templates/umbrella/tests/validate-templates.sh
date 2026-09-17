#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Render the chart and validate every manifest against the Kubernetes schemas.
# Requires: helm, kubeconform (https://github.com/yannh/kubeconform)
set -euo pipefail

cd "$(dirname "$0")/.."

command -v helm >/dev/null 2>&1 || { echo "helm is required but not installed" >&2; exit 1; }
command -v kubeconform >/dev/null 2>&1 || { echo "kubeconform is required but not installed" >&2; exit 1; }

rendered="$(mktemp "${TMPDIR:-/tmp}/chartpress-manifests.XXXXXX.yaml")"
trap 'rm -f "$rendered"' EXIT

echo "Rendering chart with 'helm template'..."
helm template chartpress-test . > "$rendered"

echo "Validating rendered manifests with kubeconform..."
if [[ -n "${KUBECONFORM_SCHEMA_LOCATION:-}" ]]; then
  kubeconform -kubernetes-version "${KUBERNETES_SCHEMA_VERSION:-1.32.0}" \
    -schema-location "$KUBECONFORM_SCHEMA_LOCATION" \
    -strict -summary -ignore-missing-schemas "$rendered"
else
  kubeconform -kubernetes-version "${KUBERNETES_SCHEMA_VERSION:-1.32.0}" \
    -strict -summary -ignore-missing-schemas "$rendered"
fi

echo "Done."
