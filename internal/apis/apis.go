// internal/apis/apis.go
// Package apis is the single source of truth for the ChartpressConfig API
// identity, shared by the backend (apply/list) and the operator (watch/reconcile).
package apis

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	Group        = "chartpress.dev"
	Version      = "v1alpha1"
	GroupVersion = Group + "/" + Version
	Kind         = "ChartpressConfig"
	Resource     = "chartpressconfigs"

	FinalizerArtifactCleanup = "chartpress.dev/artifact-cleanup"
	FieldManagerBackend      = "chartpress-backend"
	FieldManagerOperator     = "chartpress-operator"
)

// GVR is the GroupVersionResource for ChartpressConfig CRs.
var GVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: Resource}
