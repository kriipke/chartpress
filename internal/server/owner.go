// internal/server/owner.go — chart ownership.
//
// Every generated chart is scoped to an owner so that (a) a signed-in user has a
// private, persistent library and (b) anonymous browsers don't see or clobber
// each other's charts. The owner is:
//
//   - a signed-in user, keyed by GitHub login (from the session cookie), or
//   - an anonymous browser, keyed by a random client id the web app generates and
//     sends in the X-Chartpress-Client header (persisted in the browser).
//
// The owner is hashed into the CR name (<hash>-<chartName>) so two owners can
// reuse the same chart name without colliding on the cluster-unique resource
// name, and mirrored onto the LabelOwner label so /charts can filter to the
// caller. The user-facing chart name is preserved in AnnotationChartName and in
// spec.umbrellaChartName; the mangled metadata.name never surfaces in the API.
package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/kriipke/chartpress/internal/apis"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// clientIDHeader carries the anonymous browser's stable client id.
const clientIDHeader = "X-Chartpress-Client"

// anonChartTTL bounds how long an anonymous chart's server-side artifact lives.
// The browser keeps its own durable record (spec + name) in localStorage, so a
// reaped chart can be regenerated; this only caps cluster/object-store growth.
const anonChartTTL = 24 * time.Hour

// k8s resource names are DNS subdomains capped at 253 chars.
const maxResourceName = 253

// ownerRef identifies who a chart belongs to. id is the GitHub login for a user
// or the browser client id for an anonymous session ("" when an anonymous caller
// sent no client id — a degenerate shared bucket, used only by direct API hits).
type ownerRef struct {
	kind string // apis.OwnerKindUser | apis.OwnerKindAnon
	id   string
}

// requestOwner derives the owner of a request: the signed-in user if the session
// cookie is valid, else the anonymous browser identified by the client-id header.
func (s *Server) requestOwner(r *http.Request) ownerRef {
	if u := s.currentUser(r); u != nil && u.Login != "" {
		return ownerRef{kind: apis.OwnerKindUser, id: u.Login}
	}
	return ownerRef{kind: apis.OwnerKindAnon, id: clientID(r)}
}

var clientIDInvalid = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// clientID reads and sanitizes the anonymous client id from the request header,
// keeping it to an opaque, bounded, filesystem-safe token.
func clientID(r *http.Request) string {
	id := clientIDInvalid.ReplaceAllString(strings.TrimSpace(r.Header.Get(clientIDHeader)), "")
	if len(id) > 64 {
		id = id[:64]
	}
	return id
}

// hash is a short, stable, DNS-safe token derived from the owner identity. It is
// used both as the CR-name prefix and the LabelOwner value, so an owner's charts
// share a name-space and are selectable by label.
func (o ownerRef) hash() string {
	sum := sha256.Sum256([]byte(o.kind + ":" + o.id))
	return hex.EncodeToString(sum[:])[:12]
}

// storedName maps a user-facing chart name to the owner-scoped metadata.name.
// The 12-hex prefix keeps the result a valid DNS-1123 subdomain (umbrella names
// are already lowercase alphanumeric/hyphen); it is truncated to the k8s cap.
func (o ownerRef) storedName(chartName string) string {
	name := o.hash() + "-" + chartName
	if len(name) > maxResourceName {
		name = strings.TrimRight(name[:maxResourceName], "-.")
	}
	return name
}

// applyOwnership rewrites obj into an owner-scoped CR: owner-prefixed name, owner
// labels, the display-name annotation, and (for anonymous owners) a TTL that lets
// the operator reap the chart later.
func applyOwnership(obj *unstructured.Unstructured, o ownerRef, chartName string, now time.Time) {
	obj.SetName(o.storedName(chartName))

	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[apis.LabelOwner] = o.hash()
	labels[apis.LabelOwnerKind] = o.kind
	obj.SetLabels(labels)

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[apis.AnnotationChartName] = chartName
	if o.kind == apis.OwnerKindAnon {
		annotations[apis.AnnotationExpiresAt] = now.Add(anonChartTTL).UTC().Format(time.RFC3339)
	}
	obj.SetAnnotations(annotations)
}

// ownsChart reports whether obj belongs to owner o (its LabelOwner matches).
func (o ownerRef) ownsChart(obj unstructured.Unstructured) bool {
	return obj.GetLabels()[apis.LabelOwner] == o.hash()
}

// chartDisplayName is the user-facing name for a CR: the display annotation, else
// spec.umbrellaChartName, else the owner-prefix-stripped metadata.name.
func chartDisplayName(obj unstructured.Unstructured) string {
	if n := obj.GetAnnotations()[apis.AnnotationChartName]; n != "" {
		return n
	}
	if n, _, _ := unstructured.NestedString(obj.Object, "spec", "umbrellaChartName"); n != "" {
		return n
	}
	name := obj.GetName()
	if i := strings.IndexByte(name, '-'); i == 12 { // "<12hex>-"
		return name[i+1:]
	}
	return name
}
