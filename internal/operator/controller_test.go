package operator

import (
	"testing"
	"time"

	"github.com/kriipke/chartpress/internal/apis"
)

func TestNewControllerEnqueuesEvents(t *testing.T) {
	obj := crObj("demo", 1)
	dyn := newFakeDynamic(obj)
	rec := &Reconciler{
		Client:    newDynamicCRClient(dyn),
		Renderer:  fakeRenderer{zip: []byte("x")},
		Uploader:  &fakeUploader{},
		Namespace: "chartpress-system",
		Now:       fixedClock(),
	}

	c, err := NewController(dyn, rec, "chartpress-system", 30*time.Second)
	if err != nil {
		t.Fatalf("NewController: %v", err)
	}
	// The seeded object should produce an enqueueable key.
	c.enqueue(obj)
	if c.queue.Len() == 0 {
		t.Fatal("expected the seeded CR to be enqueued")
	}
	key, _ := c.queue.Get()
	if key != "chartpress-system/demo" {
		t.Fatalf("queue key = %q, want chartpress-system/demo", key)
	}
	_ = apis.GVR // keep apis imported for clarity
}

func TestNamespaceAndTemplatesDirDefaults(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "")
	if namespaceFromEnv() != "default" {
		t.Fatalf("namespace default = %q", namespaceFromEnv())
	}
	t.Setenv("POD_NAMESPACE", "chartpress-system")
	if namespaceFromEnv() != "chartpress-system" {
		t.Fatalf("namespace = %q", namespaceFromEnv())
	}
	t.Setenv("CHARTPRESS_TEMPLATES_DIR", "")
	if templatesDir() != "templates" {
		t.Fatalf("templatesDir default = %q", templatesDir())
	}
	t.Setenv("CHARTPRESS_TEMPLATES_DIR", "/app/templates")
	if templatesDir() != "/app/templates" {
		t.Fatalf("templatesDir = %q", templatesDir())
	}
}
