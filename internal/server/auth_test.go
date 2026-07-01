package server

import (
	"encoding/json"
	"testing"
)

func TestGitHubAuthConfigured(t *testing.T) {
	var nilAuth *GitHubAuth
	if nilAuth.configured() {
		t.Error("nil *GitHubAuth should not be configured")
	}
	cases := []struct {
		name string
		a    *GitHubAuth
		want bool
	}{
		{"empty", &GitHubAuth{}, false},
		{"no-secret", &GitHubAuth{ClientID: "id", ClientSecret: "s"}, false},
		{"complete", &GitHubAuth{ClientID: "id", ClientSecret: "s", secret: []byte("k")}, true},
	}
	for _, c := range cases {
		if got := c.a.configured(); got != c.want {
			t.Errorf("%s: configured()=%v, want %v", c.name, got, c.want)
		}
	}
}

func TestSignVerifyRoundtrip(t *testing.T) {
	a := &GitHubAuth{secret: []byte("test-signing-secret")}
	u := sessionUser{Login: "ada", Name: "Ada Lovelace", Registry: "oci://ghcr.io/ada"}
	payload, _ := json.Marshal(u)

	token := a.sign(payload)
	got, ok := a.verify(token)
	if !ok {
		t.Fatal("verify rejected a freshly signed token")
	}
	var back sessionUser
	if err := json.Unmarshal(got, &back); err != nil {
		t.Fatalf("unmarshal roundtrip: %v", err)
	}
	if back != u {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", back, u)
	}

	// Tampered payload must fail.
	if _, ok := a.verify("x" + token); ok {
		t.Error("verify accepted a tampered token")
	}
	// A different key must not verify.
	other := &GitHubAuth{secret: []byte("different-secret")}
	if _, ok := other.verify(token); ok {
		t.Error("verify accepted a token signed with a different key")
	}
	// Malformed tokens must fail, not panic.
	for _, bad := range []string{"", "nodot", "a.b.c"} {
		if _, ok := a.verify(bad); ok {
			t.Errorf("verify accepted malformed token %q", bad)
		}
	}
}
