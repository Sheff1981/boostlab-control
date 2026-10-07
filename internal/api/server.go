package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sort"
	"sync"
	"time"
)

type Node struct {
	ID                 string    `json:"id"`
	Region             string    `json:"region"`
	CountryCode        string    `json:"country_code,omitempty"`
	City               string    `json:"city,omitempty"`
	DisplayName        string    `json:"display_name,omitempty"`
	Host               string    `json:"host"`
	UDPPort            int       `json:"udp_port"`
	RouteAPIURL        string    `json:"route_api_url,omitempty"`
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
	Registry     *Registry
	Social       *SocialHub
	Games        []GameCatalogEntry
	RouteTargets []GameRouteTarget
	VoiceIce     VoiceIceProvider
	Auth         *DeviceAuthHub
}

func NewServer(registry *Registry) Server {
	return NewServerWithFullDependencies(registry, nil, nil, NewSocialHub(), VoiceIceProvider{})
}

func NewServerWithGames(registry *Registry, games []GameCatalogEntry) Server {
	return NewServerWithFullDependencies(registry, games, nil, NewSocialHub(), VoiceIceProvider{})
}

func NewServerWithDependencies(
	registry *Registry,
	games []GameCatalogEntry,
	social *SocialHub,
	voiceIce VoiceIceProvider,
) Server {
	return NewServerWithFullDependencies(registry, games, nil, social, voiceIce)
}

func NewServerWithFullDependencies(
	registry *Registry,
	games []GameCatalogEntry,
	routeTargets []GameRouteTarget,
	social *SocialHub,
	voiceIce VoiceIceProvider,
) Server {
	if social == nil {
		social = NewSocialHub()
	}
	return Server{
		Registry:     registry,
		Social:       social,
		Games:        append([]GameCatalogEntry(nil), games...),
		RouteTargets: append([]GameRouteTarget(nil), routeTargets...),
		VoiceIce:     voiceIce,
		Auth:         NewDeviceAuthHub(),
	}
}

func (s Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /v1/auth/challenge", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		defer r.Body.Close()

		var input struct {
			PublicKey string `json:"public_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		challenge, err := s.Auth.NewChallenge(strings.TrimSpace(input.PublicKey), time.Now())
		if err != nil {
			http.Error(w, "invalid device public key", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(struct {
			ChallengeID string    `json:"challenge_id"`
			Message     string    `json:"message"`
			ExpiresAt   time.Time `json:"expires_at"`
		}{
			ChallengeID: challenge.ID,
			Message:     challenge.Message,
			ExpiresAt:   challenge.ExpiresAt,
		})
	})

	mux.HandleFunc("POST /v1/auth/session", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		defer r.Body.Close()

		var input struct {
			ChallengeID string `json:"challenge_id"`
			Signature   string `json:"signature"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		session, err := s.Auth.Exchange(
			strings.TrimSpace(input.ChallengeID),
			strings.TrimSpace(input.Signature),
			time.Now(),
		)
		if err != nil {
			http.Error(w, "device authentication failed", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(struct {
			DeviceID    string    `json:"device_id"`
			AccessToken string    `json:"access_token"`
			ExpiresAt   time.Time `json:"expires_at"`
		}{
			DeviceID:    session.DeviceID,
			AccessToken: session.Token,
			ExpiresAt:   session.ExpiresAt,
		})
	})

	mux.HandleFunc("GET /v1/nodes", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(s.Registry.List())
	})

	mux.HandleFunc("GET /v1/games", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(s.Games)
	})

	mux.HandleFunc("GET /v1/route-targets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(
			filterRouteTargets(s.RouteTargets, r.URL.Query().Get("package_name")),
		)
	})

	mux.HandleFunc("GET /v1/voice/ice", func(w http.ResponseWriter, r *http.Request) {
		config, err := s.VoiceIce.ConfigFor(
			strings.TrimSpace(r.URL.Query().Get("user_id")),
			time.Now(),
		)
		if err != nil {
			http.Error(w, "invalid voice credentials request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(config)
	})

	mux.HandleFunc("GET /v1/direct/{peer}/events", func(w http.ResponseWriter, r *http.Request) {
		self := strings.TrimSpace(r.URL.Query().Get("self"))
		peer := strings.TrimSpace(r.PathValue("peer"))
		room, ok := directRoom(self, peer)
		if !ok {
			http.Error(w, "invalid direct chat participants", http.StatusBadRequest)
			return
		}

		var after uint64
		if raw := strings.TrimSpace(r.URL.Query().Get("after")); raw != "" {
			value, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				http.Error(w, "invalid after cursor", http.StatusBadRequest)
				return
			}
			after = value
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(s.Social.List(room, after))
	})

	mux.HandleFunc("POST /v1/direct/{peer}/events", func(w http.ResponseWriter, r *http.Request) {
		peer := strings.TrimSpace(r.PathValue("peer"))
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		defer r.Body.Close()

		var input struct {
			Sender string `json:"sender"`
			Text   string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		input.Sender = strings.TrimSpace(input.Sender)
		input.Text = strings.TrimSpace(input.Text)
		room, ok := directRoom(input.Sender, peer)
		if !ok {
			http.Error(w, "invalid direct chat participants", http.StatusBadRequest)
			return
		}
		if input.Text == "" {
			http.Error(w, "message text is required", http.StatusBadRequest)
			return
		}
		if len(input.Text) > 1000 {
			http.Error(w, "message too large", http.StatusRequestEntityTooLarge)
			return
		}

		event, err := s.Social.Append(room, input.Sender, "chat", input.Text, "")
		if err != nil {
			http.Error(w, "failed to persist direct message", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(event)
	})

	mux.HandleFunc("GET /v1/squads/{code}/events", func(w http.ResponseWriter, r *http.Request) {
		code, ok := normalizeRoomCode(r.PathValue("code"))
		if !ok {
			http.Error(w, "invalid squad code", http.StatusBadRequest)
			return
		}

		var after uint64
		if raw := strings.TrimSpace(r.URL.Query().Get("after")); raw != "" {
			value, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				http.Error(w, "invalid after cursor", http.StatusBadRequest)
				return
			}
			after = value
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(s.Social.List(code, after))
	})

	mux.HandleFunc("POST /v1/squads/{code}/events", func(w http.ResponseWriter, r *http.Request) {
		code, ok := normalizeRoomCode(r.PathValue("code"))
		if !ok {
			http.Error(w, "invalid squad code", http.StatusBadRequest)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
		defer r.Body.Close()

		var input struct {
			Sender  string `json:"sender"`
			Type    string `json:"type"`
			Text    string `json:"text"`
			Payload string `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		input.Sender = strings.TrimSpace(input.Sender)
		input.Type = strings.TrimSpace(input.Type)
		input.Text = strings.TrimSpace(input.Text)
		input.Payload = strings.TrimSpace(input.Payload)

		if !validUserID(input.Sender) {
			http.Error(w, "invalid sender", http.StatusBadRequest)
			return
		}
		if !validSquadEventType(input.Type) {
			http.Error(w, "invalid event type", http.StatusBadRequest)
			return
		}
		if len(input.Text) > 1000 || len(input.Payload) > 16<<10 {
			http.Error(w, "event too large", http.StatusRequestEntityTooLarge)
			return
		}
		if input.Type == "chat" && input.Text == "" {
			http.Error(w, "chat text is required", http.StatusBadRequest)
			return
		}
		if input.Type != "chat" && input.Payload == "" {
			http.Error(w, "signal payload is required", http.StatusBadRequest)
			return
		}

		event, err := s.Social.Append(code, input.Sender, input.Type, input.Text, input.Payload)
		if err != nil {
			http.Error(w, "failed to persist event", http.StatusInternalServerError)
			return
		}
		s.Social.Touch(code, input.Sender)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(event)
	})

	mux.HandleFunc("GET /v1/squads/{code}/presence", func(w http.ResponseWriter, r *http.Request) {
		code, ok := normalizeRoomCode(r.PathValue("code"))
		if !ok {
			http.Error(w, "invalid squad code", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(s.Social.Presence(code))
	})

	mux.HandleFunc("POST /v1/squads/{code}/presence", func(w http.ResponseWriter, r *http.Request) {
		code, ok := normalizeRoomCode(r.PathValue("code"))
		if !ok {
			http.Error(w, "invalid squad code", http.StatusBadRequest)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		defer r.Body.Close()

		var input struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		input.UserID = strings.TrimSpace(input.UserID)
		if !validUserID(input.UserID) {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}
		s.Social.Touch(code, input.UserID)
		w.WriteHeader(http.StatusNoContent)
	})

	return mux
}

func normalizeRoomCode(value string) (string, bool) {
	code := strings.ToUpper(strings.TrimSpace(value))
	if len(code) < 4 || len(code) > 64 {
		return "", false
	}
	for _, r := range code {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return "", false
		}
	}
	return code, true
}

func validUserID(value string) bool {
	if len(value) < 3 || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func validSquadEventType(value string) bool {
	switch value {
	case "chat", "voice_offer", "voice_answer", "voice_ice", "voice_hangup":
		return true
	default:
		return false
	}
}
