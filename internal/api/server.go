package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Node struct {
	ID                 string    `json:"id"`
	Region             string    `json:"region"`
	Host               string    `json:"host"`
	UDPPort            int       `json:"udp_port"`
	WireGuardPublicKey string    `json:"wireguard_public_key,omitempty"`
	WireGuardPort      int       `json:"wireguard_port,omitempty"`
	Healthy            bool      `json:"healthy"`
	Updated            time.Time `json:"updated_at"`
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

	sort.Slice(out, func(i, j int) bool {
		if out[i].Region == out[j].Region {
			return out[i].ID < out[j].ID
		}
		return out[i].Region < out[j].Region
	})

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
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(s.Registry.List())
	})

	return mux
}
