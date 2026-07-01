// internal/server/auth.go — optional GitHub OAuth sign-in (identity only).
//
// Sign-in is NON-GATING: the existing endpoints stay open. Signing in just
// establishes an identity the UI can show (nav avatar, profile). The session is
// a stateless, HMAC-signed cookie carrying the user's public GitHub profile — no
// server-side session store, no database.
//
// Configure via env (rendered from the Helm chart's github Secret):
//
//	GITHUB_OAUTH_CLIENT_ID, GITHUB_OAUTH_CLIENT_SECRET,
//	GITHUB_OAUTH_REDIRECT_URL (…/auth/github/callback), SESSION_SECRET
//
// When unset, GitHubAuth.configured() is false: /auth/me reports {configured:
// false} and the login/callback endpoints return 503, so the UI simply hides
// sign-in and the app remains fully usable.
package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	sessionCookie = "cp_session"
	stateCookie   = "cp_oauth_state"
	sessionTTL    = 7 * 24 * time.Hour
	githubMaxBody = 1 << 20 // 1 MiB cap on GitHub API responses
)

// GitHubAuth holds the OAuth app configuration and the session signing key. A
// zero value (or missing env) is "not configured" and degrades gracefully.
type GitHubAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	secret       []byte
	http         *http.Client
}

// NewGitHubAuth reads the OAuth configuration from the environment.
func NewGitHubAuth() *GitHubAuth {
	return &GitHubAuth{
		ClientID:     os.Getenv("GITHUB_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_OAUTH_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GITHUB_OAUTH_REDIRECT_URL"),
		secret:       []byte(os.Getenv("SESSION_SECRET")),
		http:         &http.Client{Timeout: 10 * time.Second},
	}
}

// configured reports whether sign-in can run (nil-safe).
func (a *GitHubAuth) configured() bool {
	return a != nil && a.ClientID != "" && a.ClientSecret != "" && len(a.secret) > 0
}

// sessionUser is the identity carried in the signed session cookie and returned
// by /auth/me. All fields are public GitHub profile data.
type sessionUser struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Registry  string `json:"registry,omitempty"`
}

// handleAuthMe returns whether sign-in is configured and, if a valid session
// cookie is present, the current user.
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	resp := struct {
		Configured    bool         `json:"configured"`
		Authenticated bool         `json:"authenticated"`
		User          *sessionUser `json:"user,omitempty"`
	}{Configured: s.Auth.configured()}
	if u := s.currentUser(r); u != nil {
		resp.Authenticated = true
		resp.User = u
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// currentUser returns the signed-in user from the session cookie, or nil.
func (s *Server) currentUser(r *http.Request) *sessionUser {
	if !s.Auth.configured() {
		return nil
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	payload, ok := s.Auth.verify(c.Value)
	if !ok {
		return nil
	}
	var u sessionUser
	if json.Unmarshal(payload, &u) != nil {
		return nil
	}
	return &u
}

// handleAuthLogin starts the OAuth dance: set a CSRF state cookie and redirect
// to GitHub's authorize screen.
func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if !s.Auth.configured() {
		http.Error(w, "GitHub sign-in is not configured on this server", http.StatusServiceUnavailable)
		return
	}
	state, err := randToken()
	if err != nil {
		http.Error(w, "failed to start sign-in", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: stateCookie, Value: state, Path: "/", HttpOnly: true,
		Secure: isSecure(r), SameSite: http.SameSiteLaxMode, MaxAge: 600,
	})
	q := url.Values{}
	q.Set("client_id", s.Auth.ClientID)
	if s.Auth.RedirectURL != "" {
		q.Set("redirect_uri", s.Auth.RedirectURL)
	}
	q.Set("scope", "read:user user:email")
	q.Set("state", state)
	http.Redirect(w, r, "https://github.com/login/oauth/authorize?"+q.Encode(), http.StatusFound)
}

// handleAuthCallback validates state, exchanges the code, loads the profile, and
// sets the session cookie before redirecting back to the app.
func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	if !s.Auth.configured() {
		http.Error(w, "GitHub sign-in is not configured on this server", http.StatusServiceUnavailable)
		return
	}
	st, err := r.Cookie(stateCookie)
	if err != nil || st.Value == "" || st.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid or missing OAuth state", http.StatusBadRequest)
		return
	}
	// Consume the state cookie.
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Value: "", Path: "/", MaxAge: -1})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}
	token, err := s.Auth.exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "token exchange failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	u, err := s.Auth.fetchUser(r.Context(), token)
	if err != nil {
		http.Error(w, "failed to load GitHub profile: "+err.Error(), http.StatusBadGateway)
		return
	}
	payload, _ := json.Marshal(u)
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: s.Auth.sign(payload), Path: "/", HttpOnly: true,
		Secure: isSecure(r), SameSite: http.SameSiteLaxMode, MaxAge: int(sessionTTL.Seconds()),
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleAuthLogout clears the session cookie.
func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", HttpOnly: true,
		Secure: isSecure(r), SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
	if r.Method == http.MethodGet {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// sign returns "<base64url(payload)>.<base64url(HMAC-SHA256(payload))>".
func (a *GitHubAuth) sign(payload []byte) string {
	mac := hmac.New(sha256.New, a.secret)
	mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// verify checks the signature and returns the payload if valid.
func (a *GitHubAuth) verify(token string) ([]byte, bool) {
	dot := strings.IndexByte(token, '.')
	if dot < 0 {
		return nil, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(token[:dot])
	if err != nil {
		return nil, false
	}
	sig, err := base64.RawURLEncoding.DecodeString(token[dot+1:])
	if err != nil {
		return nil, false
	}
	mac := hmac.New(sha256.New, a.secret)
	mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, false
	}
	return payload, true
}

// exchange trades an authorization code for an access token.
func (a *GitHubAuth) exchange(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", a.ClientID)
	form.Set("client_secret", a.ClientSecret)
	form.Set("code", code)
	if a.RedirectURL != "" {
		form.Set("redirect_uri", a.RedirectURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := a.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, githubMaxBody))
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github token endpoint returned %d", res.StatusCode)
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}
	if tr.AccessToken == "" {
		if tr.Error != "" {
			return "", fmt.Errorf("%s", tr.Error)
		}
		return "", fmt.Errorf("no access token returned")
	}
	return tr.AccessToken, nil
}

// fetchUser loads the authenticated user's public profile (and primary email
// when the profile email is private).
func (a *GitHubAuth) fetchUser(ctx context.Context, token string) (*sessionUser, error) {
	body, err := a.githubGet(ctx, token, "https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	var gh struct {
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &gh); err != nil {
		return nil, err
	}
	u := &sessionUser{Login: gh.Login, Name: gh.Name, Email: gh.Email, AvatarURL: gh.AvatarURL}
	if u.Name == "" {
		u.Name = gh.Login
	}
	if u.Login != "" {
		u.Registry = "oci://ghcr.io/" + u.Login
	}
	if u.Email == "" {
		if eb, err := a.githubGet(ctx, token, "https://api.github.com/user/emails"); err == nil {
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			if json.Unmarshal(eb, &emails) == nil {
				for _, e := range emails {
					if e.Primary {
						u.Email = e.Email
						break
					}
				}
			}
		}
	}
	return u, nil
}

func (a *GitHubAuth) githubGet(ctx context.Context, token, uri string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "chartpress")
	res, err := a.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, githubMaxBody))
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github %s returned %d", uri, res.StatusCode)
	}
	return body, nil
}

func randToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// isSecure reports whether the original request used HTTPS, honoring the
// X-Forwarded-Proto header set by the ingress/nginx in front of the backend.
func isSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
