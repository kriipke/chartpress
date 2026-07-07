// internal/server/charts_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kriipke/chartpress/internal/apis"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// anonOwner is the owner of a request that carries no session cookie and no
// client-id header — the default for the httptest requests below.
var anonOwner = ownerRef{kind: apis.OwnerKindAnon, id: ""}

func userOwner(login string) ownerRef { return ownerRef{kind: apis.OwnerKindUser, id: login} }

// owned stamps the owner metadata (name prefix, labels, display annotation) onto
// a CR, mirroring what applyOwnership does at generate time, so scoped handlers
// can find and match it.
func owned(cr unstructured.Unstructured, o ownerRef, chartName string) unstructured.Unstructured {
	applyOwnership(&cr, o, chartName, time.Unix(0, 0))
	return cr
}

// testAuth returns a configured GitHubAuth so requests can carry a signed
// session; authCookie forges the cookie for a given login.
func testAuth() *GitHubAuth {
	return &GitHubAuth{ClientID: "id", ClientSecret: "secret", secret: []byte("test-key")}
}

func authCookie(a *GitHubAuth, login string) *http.Cookie {
	payload, _ := json.Marshal(sessionUser{Login: login})
	return &http.Cookie{Name: sessionCookie, Value: a.sign(payload)}
}

// fakeLister serves canned CRs.
type fakeLister struct {
	items []unstructured.Unstructured
}

func (f *fakeLister) List(ctx context.Context, namespace string) ([]unstructured.Unstructured, error) {
	return f.items, nil
}
func (f *fakeLister) Get(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	for i := range f.items {
		if f.items[i].GetName() == name {
			return &f.items[i], nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: apiGroup, Resource: "chartpressconfigs"}, name)
}

func crWith(name, phase, message string, subcharts int, lastGen string) unstructured.Unstructured {
	subs := make([]interface{}, subcharts)
	for i := range subs {
		subs[i] = map[string]interface{}{"name": "s", "workload": "deployment"}
	}
	status := map[string]interface{}{}
	if phase != "" {
		status["phase"] = phase
	}
	if message != "" {
		status["message"] = message
	}
	if lastGen != "" {
		status["lastGenerated"] = lastGen
	}
	return unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersionV1alpha1,
		"kind":       kindChartpressConfig,
		"metadata":   map[string]interface{}{"name": name},
		"spec":       map[string]interface{}{"umbrellaChartName": name, "subcharts": subs},
		"status":     status,
	}}
}

func TestChartsListMapsFields(t *testing.T) {
	srv := &Server{Lister: &fakeLister{items: []unstructured.Unstructured{
		owned(crWith("ready-one", "Ready", "", 3, "2026-06-30T03:10:00Z"), anonOwner, "ready-one"),
		owned(crWith("fresh-one", "", "", 1, ""), anonOwner, "fresh-one"), // empty status → Pending
		owned(crWith("bad-one", "Failed", "render blew up", 2, ""), anonOwner, "bad-one"),
	}}, Namespace: "ns"}

	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got []chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	byName := map[string]chartSummary{}
	for _, c := range got {
		byName[c.Name] = c
	}
	if c := byName["ready-one"]; c.Phase != "Ready" || c.SubchartCount != 3 || c.LastGenerated != "2026-06-30T03:10:00Z" || c.DownloadURL != "" {
		t.Fatalf("ready-one = %+v (downloadUrl must be empty when the Server has no Presigner)", c)
	}
	if c := byName["fresh-one"]; c.Phase != "Pending" {
		t.Fatalf("empty status should map to Pending, got %q", c.Phase)
	}
	if c := byName["bad-one"]; c.Phase != "Failed" || c.Message != "render blew up" {
		t.Fatalf("bad-one = %+v", c)
	}
}

func TestChartByNameFound(t *testing.T) {
	srv := &Server{Lister: &fakeLister{items: []unstructured.Unstructured{
		owned(crWith("demo", "Generating", "", 2, ""), anonOwner, "demo"),
	}}}
	req := httptest.NewRequest(http.MethodGet, "/charts/demo", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "demo" || got.Phase != "Generating" || got.SubchartCount != 2 {
		t.Fatalf("got = %+v", got)
	}
}

func TestChartByNameNotFound(t *testing.T) {
	srv := &Server{Lister: &fakeLister{}}
	req := httptest.NewRequest(http.MethodGet, "/charts/missing", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

type fakePresigner struct {
	url       string
	err       error
	lastKey   string
	callCount int
}

func (p *fakePresigner) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	p.callCount++
	p.lastKey = key
	return p.url, p.err
}

// crReady builds a Ready CR whose status reflects the current generation — the
// state the operator leaves after a successful reconcile (observedGeneration ==
// metadata.generation), so /charts should mint a download URL.
func crReady(name, artifactKey string) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersionV1alpha1,
		"kind":       kindChartpressConfig,
		"metadata":   map[string]interface{}{"name": name, "generation": int64(1)},
		"spec": map[string]interface{}{
			"umbrellaChartName": name,
			"subcharts":         []interface{}{map[string]interface{}{"name": "api", "workload": "deployment"}},
		},
		"status": map[string]interface{}{"phase": "Ready", "artifactKey": artifactKey, "observedGeneration": int64(1)},
	}}
}

func TestChartsMintsDownloadURLWhenReady(t *testing.T) {
	pre := &fakePresigner{url: "https://s3.example.com/charts/demo.zip?sig=abc"}
	srv := &Server{
		Lister:    &fakeLister{items: []unstructured.Unstructured{owned(crReady("demo", "charts/demo.zip"), anonOwner, "demo")}},
		Presigner: pre,
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got []chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].DownloadURL != "https://s3.example.com/charts/demo.zip?sig=abc" {
		t.Fatalf("downloadUrl = %q (got %+v)", got[0].DownloadURL, got)
	}
	if pre.lastKey != "charts/demo.zip" {
		t.Fatalf("presigned key = %q, want charts/demo.zip", pre.lastKey)
	}
}

// A user reapplied a changed spec: metadata.generation is now 2, but the operator
// has only reconciled generation 1, so the lingering Ready status describes the
// superseded artifact. /charts must NOT presign it (the URL would point at a stale
// chart, indefinitely if the operator is down).
func TestChartsNoDownloadURLWhenGenerationStale(t *testing.T) {
	cr := owned(crReady("demo", "charts/demo.zip"), anonOwner, "demo") // observedGeneration=1
	_ = unstructured.SetNestedField(cr.Object, int64(2), "metadata", "generation")

	pre := &fakePresigner{url: "https://should-not-be-used"}
	srv := &Server{
		Lister:    &fakeLister{items: []unstructured.Unstructured{cr}},
		Presigner: pre,
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got []chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Phase != "Ready" {
		t.Fatalf("want a single Ready row, got %+v", got)
	}
	if got[0].DownloadURL != "" {
		t.Fatalf("downloadUrl must be empty for a stale Ready status; got %q", got[0].DownloadURL)
	}
	if pre.callCount != 0 {
		t.Fatalf("presigner must not be called when observedGeneration != generation; calls=%d", pre.callCount)
	}
}

func TestChartsNoDownloadURLWhenNotReady(t *testing.T) {
	pre := &fakePresigner{url: "https://should-not-be-used"}
	srv := &Server{
		Lister: &fakeLister{items: []unstructured.Unstructured{
			owned(crWith("pending-one", "Generating", "", 1, ""), anonOwner, "pending-one"), // no artifactKey
		}},
		Presigner: pre,
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got []chartSummary
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got) != 1 || got[0].DownloadURL != "" {
		t.Fatalf("downloadUrl must be empty when not Ready; got %+v", got)
	}
	if pre.callCount != 0 {
		t.Fatalf("presigner must not be called for non-Ready charts; calls=%d", pre.callCount)
	}
}

// A signed-in user's /charts returns only their own charts — not another user's,
// and not anonymous charts.
func TestChartsListScopedToOwner(t *testing.T) {
	auth := testAuth()
	srv := &Server{
		Auth: auth,
		Lister: &fakeLister{items: []unstructured.Unstructured{
			owned(crWith("mine", "Ready", "", 1, ""), userOwner("ada"), "mine"),
			owned(crWith("theirs", "Ready", "", 1, ""), userOwner("grace"), "theirs"),
			owned(crWith("anon", "Ready", "", 1, ""), anonOwner, "anon"),
		}},
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	req.AddCookie(authCookie(auth, "ada"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got []chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Name != "mine" {
		t.Fatalf("ada should see only [mine], got %+v", got)
	}
}

// Fetching another owner's chart by name is a 404 (the name resolves under the
// caller's owner-space, which has no such chart).
func TestChartByNameDeniesOtherOwner(t *testing.T) {
	auth := testAuth()
	srv := &Server{
		Auth: auth,
		Lister: &fakeLister{items: []unstructured.Unstructured{
			owned(crWith("secret", "Ready", "", 1, ""), userOwner("grace"), "secret"),
		}},
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts/secret", nil)
	req.AddCookie(authCookie(auth, "ada")) // ada, not grace
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (cross-owner access denied)", rec.Code)
	}
}

// Two distinct owners can hold charts with the same user-facing name; each sees
// their own, addressed by that shared name.
func TestChartByNameSameNameDistinctOwners(t *testing.T) {
	auth := testAuth()
	srv := &Server{
		Auth: auth,
		Lister: &fakeLister{items: []unstructured.Unstructured{
			owned(crWith("app", "Ready", "ada's", 1, ""), userOwner("ada"), "app"),
			owned(crWith("app", "Failed", "grace's", 1, ""), userOwner("grace"), "app"),
		}},
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts/app", nil)
	req.AddCookie(authCookie(auth, "grace"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "app" || got.Phase != "Failed" || got.Message != "grace's" {
		t.Fatalf("grace should see her own 'app', got %+v", got)
	}
}

// An anonymous browser is scoped by its client-id header: it sees its own charts
// and not those of another anonymous client.
func TestChartsListScopedToAnonClient(t *testing.T) {
	srv := &Server{
		Lister: &fakeLister{items: []unstructured.Unstructured{
			owned(crWith("c1", "Ready", "", 1, ""), ownerRef{apis.OwnerKindAnon, "client-1"}, "c1"),
			owned(crWith("c2", "Ready", "", 1, ""), ownerRef{apis.OwnerKindAnon, "client-2"}, "c2"),
		}},
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	req.Header.Set(clientIDHeader, "client-1")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got []chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Name != "c1" {
		t.Fatalf("client-1 should see only [c1], got %+v", got)
	}
}
