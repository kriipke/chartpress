// internal/engine/istio_test.go
package engine

import (
	"strings"
	"testing"
)

func TestIngressIstioGatewayAndVirtualService(t *testing.T) {
	man := renderWithHost(t, "istio")
	if !strings.Contains(man, "kind: Gateway") {
		t.Fatalf("expected istio Gateway, got:\n%s", man)
	}
	if !strings.Contains(man, "kind: VirtualService") {
		t.Fatalf("expected istio VirtualService, got:\n%s", man)
	}
	if strings.Contains(man, "kind: Ingress") {
		t.Fatalf("istio should not emit a plain Ingress")
	}
}
