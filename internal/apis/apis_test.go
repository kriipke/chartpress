// internal/apis/apis_test.go
package apis

import "testing"

func TestGVRAndIdentity(t *testing.T) {
	if Group != "chartpress.dev" || Version != "v1alpha1" {
		t.Fatalf("group/version = %s/%s", Group, Version)
	}
	if GroupVersion != "chartpress.dev/v1alpha1" {
		t.Fatalf("groupVersion = %q", GroupVersion)
	}
	if Kind != "ChartpressConfig" || Resource != "chartpressconfigs" {
		t.Fatalf("kind/resource = %s/%s", Kind, Resource)
	}
	if GVR.Group != Group || GVR.Version != Version || GVR.Resource != Resource {
		t.Fatalf("GVR = %+v", GVR)
	}
}

func TestFinalizerAndFieldManagers(t *testing.T) {
	if FinalizerArtifactCleanup != "chartpress.dev/artifact-cleanup" {
		t.Fatalf("finalizer = %q", FinalizerArtifactCleanup)
	}
	if FieldManagerBackend != "chartpress-backend" || FieldManagerOperator != "chartpress-operator" {
		t.Fatalf("field managers = %s / %s", FieldManagerBackend, FieldManagerOperator)
	}
}
