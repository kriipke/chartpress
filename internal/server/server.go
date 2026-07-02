// internal/server/server.go
package server

import (
	"log"
	"net/http"
	"os"

	"github.com/kriipke/chartpress/internal/objectstore"
)

// Server is the chartpress backend HTTP layer. Its dependencies are interfaces
// so handlers can be tested with in-memory fakes (no apiserver, no OpenAI).
type Server struct {
	Applier    Applier
	Lister     ChartLister
	Drafter    Drafter
	Presigner  Presigner
	Downloader objectstore.Downloader
	Auth       *GitHubAuth
	Namespace  string
}

// Handler builds the HTTP mux. Routes are registered by their owning task:
// /generate (this task), /charts + /charts/ (Task 5), /text-to-config (Task 6).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/generate", s.cors(s.handleGenerate))
	mux.HandleFunc("/charts", s.cors(s.handleCharts))
	// /charts/ covers both /charts/{name} and /charts/{name}/files; the handler
	// dispatches on the sub-path.
	mux.HandleFunc("/charts/", s.cors(s.handleChartByName))
	mux.HandleFunc("/text-to-config", s.cors(s.handleTextToConfig))
	mux.HandleFunc("/compose-to-config", s.cors(s.handleComposeToConfig))
	// Optional GitHub sign-in (identity only; non-gating). Disabled endpoints
	// return 503 when unconfigured — see auth.go.
	mux.HandleFunc("/auth/github/login", s.cors(s.handleAuthLogin))
	mux.HandleFunc("/auth/github/callback", s.cors(s.handleAuthCallback))
	mux.HandleFunc("/auth/logout", s.cors(s.handleAuthLogout))
	mux.HandleFunc("/auth/me", s.cors(s.handleAuthMe))
	return mux
}

// cors sets permissive CORS headers and short-circuits preflight requests.
func (s *Server) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// Start wires production dependencies and serves. The dynamic client backs both
// apply and list; the namespace comes from the downward API.
func Start() {
	log.Println("[INFO] starting chartpress backend")
	client, err := newDynamicClient()
	if err != nil {
		log.Fatalf("[FATAL] kube client: %v", err)
	}
	srv := &Server{
		Applier:   &dynamicApplier{client: client},
		Lister:    &dynamicLister{client: client},
		Drafter:   newOpenAIDrafter(),
		Auth:      NewGitHubAuth(),
		Namespace: resolveNamespace(),
	}
	if srv.Auth.configured() {
		log.Println("[INFO] GitHub sign-in enabled (identity only)")
	}
	if store, err := objectstore.New(objectstore.ConfigFromEnv()); err != nil {
		log.Printf("[WARN] object storage not configured, downloads and file browsing disabled: %v", err)
	} else {
		srv.Presigner = store
		srv.Downloader = store
	}
	port := getPort()
	log.Printf("[INFO] listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv.Handler()))
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}
