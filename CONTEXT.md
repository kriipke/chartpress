# Chartpress

Chartpress generates an umbrella Helm chart and its subcharts from a single
declarative description, usable as a CLI, an HTTP server, or a web wizard. This
glossary fixes the vocabulary shared across those surfaces, the CRD, and the docs.

## Language

**ChartpressConfig**:
The declarative description of one umbrella chart and the subcharts to generate
from it. The single input to every chartpress surface — a CLI YAML file, an HTTP
JSON body, or a Kubernetes CRD resource.
_Avoid_: Config, config file, manifest, spec

**Umbrella Chart**:
The top-level Helm chart chartpress generates; it aggregates every subchart as a
dependency and is named by `umbrellaChartName`.
_Avoid_: parent chart, root chart

**Subchart**:
A child Helm chart generated under the umbrella's `charts/` directory. Each
subchart targets exactly one Workload.
_Avoid_: child chart, component, service

**Workload**:
The single Kubernetes workload kind a subchart is generated for. The canonical
set is exactly three: `deployment`, `statefulset`, `daemonset`. No other value
(e.g. `job`, `cronjob`) is a Workload in this context.
_Avoid_: workload type, resource kind

**Rules**:
The set of generation controls carried on a ChartpressConfig that determine how
the umbrella and subcharts are templated — ingress options, shared configuration,
and documentation generation. Rules are part of the desired state and are honored
by the generator (not inert annotations).
_Avoid_: options, flags, settings

**Ingress**:
An ingress controller a generated chart is templated to support, listed under a
ChartpressConfig's `possible_ingresses`. Drawn from a fixed set: `alb`, `nginx`,
`traefik`, `istio`, `gce`.
_Avoid_: ingress type, ingress class

**Chart Template**:
The source Helm chart skeleton chartpress copies and prunes to produce output —
the `umbrella` and `subchart` template directories. Distinct from Helm's own
in-chart `templates/` files, which carry the same word but a different meaning.
_Avoid_: scaffold, skeleton, boilerplate

**Preset**:
A ready-made ChartpressConfig for a common platform shape (e.g. microservices,
IoT, ML, big-data, event-driven, CI/CD), kept under `presets/`.
_Avoid_: example, sample, profile
