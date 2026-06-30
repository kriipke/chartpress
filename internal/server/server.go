// internal/server/server.go
package server

import (
	"log"
	"net/http"
	"os"
)

// Server is the chartpress backend HTTP layer. Its dependencies are interfaces
// so handlers can be tested with in-memory fakes (no apiserver, no OpenAI).
type Server struct {
	Applier   Applier
	Lister    ChartLister
	Drafter   Drafter
	Namespace string
}

// Handler builds the HTTP mux. Routes are registered by their owning task:
// /generate (this task), /charts + /charts/ (Task 5), /text-to-config (Task 6).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/generate", s.cors(s.handleGenerate))
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
		Namespace: resolveNamespace(),
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
