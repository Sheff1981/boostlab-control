package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Node struct {
	ID       string
	Region   string
	Host     string
	UDPPort  int
	Healthy  bool
	Updated  time.Time
}

type Registry struct {
	mu    sync.RWMutex
	nodes map[string]Node
}

func NewRegistry() *Registry {
	return &Registry{nodes: make(map[string]Node)}
}

func (r *Registry) Upsert(node Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	node.Updated = time.Now().UTC()
	r.nodes[node.ID] = node
}

func (r *Registry) List() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Node, 0, len(r.nodes))
	for _, node := range r.nodes {
		out = append(out, node)
	}
	return out
}

type Server struct {
	Registry *Registry
}

func NewServer(registry *Registry) Server {
	return Server{Registry: registry}
}

func (s Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /v1/nodes", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.Registry.List())
	})

	return mux
}
