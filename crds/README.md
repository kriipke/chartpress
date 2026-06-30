# ChartpressConfig Custom Resource Definition (CRD)

The `ChartpressConfig` Custom Resource Definition (CRD) enables you to declaratively define and manage umbrella Helm charts and their subcharts within your Kubernetes clusters. This CRD is designed to work with Chartpress or similar tools to streamline the templating, validation, and publishing of Helm charts for your cloud-native applications.

## Overview

The `ChartpressConfig` resource describes the desired state of a Helm umbrella chart, its subcharts, and a set of rules that control chart templating, documentation, and configuration sharing.

**Key Features:**
- Define umbrella and subcharts with their workload types (Deployment, StatefulSet, DaemonSet).
- Specify a single platform-wide ingress controller, chart naming conventions, shared configuration, and documentation generation.
- Supports flexible chart architectures for SaaS platforms, data pipelines, ML stacks, IoT, and more.

## CRD Example

```yaml
apiVersion: chartpress.dev/v1alpha1
kind: ChartpressConfig
metadata:
  name: saas-platform
spec:
  umbrellaChartName: saas-platform
  description: "A multi-tenant SaaS platform."
  subcharts:
    - name: api
      workload: deployment
      description: "REST API service."
    - name: cache
      workload: deployment
      description: "In-memory cache layer."
    - name: database
      workload: statefulset
      description: "Primary relational database."
  rules:
    ingress: alb
    common_annotations: true
    linked_templates: true
    resource_names_match_chart_name: true
    shared_secrets_config: true
    shared_newrelic_config: true
    generate_umbrella_readme: true
    generate_subchart_readme: true
    include_docs: true
```

## Field Reference

### `spec.umbrellaChartName`
- **Type:** string
- **Description:** Name of the umbrella Helm chart (kebab-case).

### `spec.description`
- **Type:** string
- **Description:** Human-readable description for the umbrella chart.

### `spec.subcharts`
- **Type:** array
- **Description:** List of subcharts.
- **Fields:**
  - `name`: Name of the subchart (kebab-case).
  - `workload`: Type of workload (`deployment`, `statefulset`, or `daemonset`).
  - `description`: Human-readable description for this subchart.

### `spec.rules`
- **Type:** object
- **Description:** Rules for chart templating and configuration.
- **Fields:**
  - `ingress`: Single platform-wide ingress controller (one of `alb`, `nginx`, `traefik`, `istio`, `gce`, `none`).
  - `common_annotations`: Boolean, whether to enable common annotations.
  - `linked_templates`: Boolean, whether to link templates.
  - `resource_names_match_chart_name`: Boolean, if resource names should match chart name.
  - `shared_secrets_config`: Boolean, enable a shared umbrella Secret wired into every subchart.
  - `shared_newrelic_config`: Boolean, enable shared New Relic config + license wired into every subchart.
  - `generate_umbrella_readme`: Boolean, generate a README for the umbrella chart.
  - `generate_subchart_readme`: Boolean, generate READMEs for subcharts.
  - `include_docs`: Boolean, include the docs/ directory in the output.

### `status`
Operator-owned status written by the chartpress operator (arrives in a later phase):
- `phase`: Lifecycle phase of the generated chart (`Pending`, `Generating`, `Ready`, `Failed`).
- `observedGeneration`: `metadata.generation` last reconciled by the operator.
- `artifactKey`: Object-storage key of the generated chart archive.
- `lastGenerated`: RFC3339 timestamp of the last successful generation.
- `message`: Human-readable status detail (error text when `Failed`).

## Usage

1. **Apply the CRD**  
   Install the CRD definition using `kubectl`:
   ```sh
   kubectl apply -f crd-helmchart.yaml
   ```

2. **Create ChartpressConfig Resources**  
   Define your `ChartpressConfig` resources as YAML manifests and apply them:
   ```sh
   kubectl apply -f helmchart-iot.yaml
   ```

3. **Integrate with Chartpress/Controller**  
   Ensure your cluster has a controller or operator that understands the `ChartpressConfig` CRD for automated chart templating and deployment. (No such controller ships in this repository yet — the operator arrives in a later phase.)

## Manifests

- [crd-helmchart.yaml](./crd-helmchart.yaml) — ChartpressConfig CRD definition
- [helmchart-iot.yaml](./helmchart-iot.yaml) — Example resource: IoT platform
- [helmchart-ml.yaml](./helmchart-ml.yaml) — Example resource: ML platform

## License

This project is proprietary — **All Rights Reserved**. See the [LICENSE](../LICENSE) for details.
