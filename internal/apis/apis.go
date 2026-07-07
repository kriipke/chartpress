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

	// Ownership metadata. Every chart is scoped to an owner — a signed-in user
	// (GitHub login) or an anonymous browser (a client-generated id). The owner is
	// hashed into the CR name so distinct owners can reuse a chart name without
	// colliding, and mirrored onto a label so the backend can filter /charts to
	// the caller.
	LabelOwner     = "chartpress.dev/owner"      // owner hash; label-selectable
	LabelOwnerKind = "chartpress.dev/owner-kind" // OwnerKindUser | OwnerKindAnon

	// AnnotationChartName preserves the user-facing umbrella chart name for
	// display, since metadata.name carries the owner-hash prefix.
	AnnotationChartName = "chartpress.dev/chart-name"
	// AnnotationExpiresAt (RFC3339) is set only on anonymous charts; the operator
	// deletes a chart once it is past this instant, and the artifact-cleanup
	// finalizer then removes the stored archive.
	AnnotationExpiresAt = "chartpress.dev/expires-at"

	OwnerKindUser = "user"
	OwnerKindAnon = "anon"
)

// GVR is the GroupVersionResource for ChartpressConfig CRs.
var GVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: Resource}
