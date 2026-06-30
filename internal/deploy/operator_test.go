package deploy

import (
	"strings"
	"testing"
)

func TestOperatorDeploymentRendered(t *testing.T) {
	man := renderChart(t)
	for _, want := range []string{
		"name: chartpress-operator",
		"command: [\"/app/operator\"]",
		"image: \"ghcr.io/kriipke/chartpress/api:",
		"serviceAccountName: chartpress-operator",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("operator deployment missing %q", want)
		}
	}
}

func TestOperatorRBACGrantsStatusAndFinalizers(t *testing.T) {
	man := renderChart(t)
	for _, want := range []string{
		"name: chartpress-operator",
		"chartpressconfigs/status",
		"chartpressconfigs/finalizers",
		`verbs: ["get", "list", "watch", "update", "patch"]`,
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("operator RBAC missing %q", want)
		}
	}
}

func TestS3EnvOnOperatorAndBackend(t *testing.T) {
	man := renderChart(t)
	for _, want := range []string{
		"name: S3_ENDPOINT",
		"name: S3_BUCKET",
		"name: S3_ACCESS_KEY",
		"name: S3_SECRET_KEY",
	} {
		if strings.Count(man, want) < 2 {
			t.Fatalf("expected %q on BOTH operator and backend (>=2 occurrences)", want)
		}
	}
	// The S3 Secret is rendered by default (s3.create defaults true).
	if !strings.Contains(man, "name: chartpress-s3") {
		t.Fatal("chartpress-s3 Secret not rendered")
	}
}
