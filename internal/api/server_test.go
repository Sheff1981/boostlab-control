package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	h := NewServer(NewRegistry()).Routes()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
}


func TestSquadChatAndPresenceAPI(t *testing.T) {
	h := NewServer(NewRegistry()).Routes()

	body, _ := json.Marshal(map[string]string{
		"sender": "BL-ABC123",
		"type":   "chat",
		"text":   "hello",
	})
	post := httptest.NewRequest(http.MethodPost, "/v1/squads/SQ-1234/events", bytes.NewReader(body))
	post.Header.Set("Content-Type", "application/json")
	postRes := httptest.NewRecorder()
	h.ServeHTTP(postRes, post)
	if postRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", postRes.Code, postRes.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, "/v1/squads/SQ-1234/events?after=0", nil)
	getRes := httptest.NewRecorder()
	h.ServeHTTP(getRes, get)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRes.Code)
	}
	var events []SquadEvent
	if err := json.NewDecoder(getRes.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Text != "hello" || events[0].Sender != "BL-ABC123" {
		t.Fatalf("unexpected events: %#v", events)
	}

	presenceReq := httptest.NewRequest(http.MethodGet, "/v1/squads/SQ-1234/presence", nil)
	presenceRes := httptest.NewRecorder()
	h.ServeHTTP(presenceRes, presenceReq)
	if presenceRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", presenceRes.Code)
	}
	var presence []SquadPresence
	if err := json.NewDecoder(presenceRes.Body).Decode(&presence); err != nil {
		t.Fatal(err)
	}
	if len(presence) != 1 || presence[0].UserID != "BL-ABC123" {
		t.Fatalf("unexpected presence: %#v", presence)
	}
}

func TestVoiceSignalRequiresPayload(t *testing.T) {
	h := NewServer(NewRegistry()).Routes()
	body, _ := json.Marshal(map[string]string{
		"sender": "BL-ABC123",
		"type":   "voice_offer",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/squads/SQ-1234/events", bytes.NewReader(body))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}


func TestVoiceIceEndpoint(t *testing.T) {
	voice, err := ParseVoiceIceProvider("turn:turn.example.com:3478", "shared-secret", "600")
	if err != nil {
		t.Fatal(err)
	}
	h := NewServerWithDependencies(NewRegistry(), nil, NewSocialHub(), voice).Routes()
	req := httptest.NewRequest(http.MethodGet, "/v1/voice/ice?user_id=BL-ABC123", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var payload VoiceIceConfig
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.IceServers) != 2 || payload.IceServers[1].Credential == "" {
		t.Fatalf("expected STUN + temporary TURN, got %#v", payload)
	}
}


func TestDirectChatAPI(t *testing.T) {
	h := NewServer(NewRegistry()).Routes()

	body, _ := json.Marshal(map[string]string{
		"sender": "BL-A123",
		"text":   "private hello",
	})
	post := httptest.NewRequest(
		http.MethodPost,
		"/v1/direct/BL-B456/events",
		bytes.NewReader(body),
	)
	postRes := httptest.NewRecorder()
	h.ServeHTTP(postRes, post)
	if postRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", postRes.Code, postRes.Body.String())
	}

	get := httptest.NewRequest(
		http.MethodGet,
		"/v1/direct/BL-A123/events?self=BL-B456&after=0",
		nil,
	)
	getRes := httptest.NewRecorder()
	h.ServeHTTP(getRes, get)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRes.Code)
	}
	var events []SquadEvent
	if err := json.NewDecoder(getRes.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Sender != "BL-A123" || events[0].Text != "private hello" {
		t.Fatalf("unexpected direct events: %#v", events)
	}
}


func TestRouteTargetsEndpointFiltersByPackage(t *testing.T) {
	targets := []GameRouteTarget{
		{ID: "a", PackageNames: []string{"game.a"}, Host: "a.example.com", TCPPort: 443},
		{ID: "b", PackageNames: []string{"game.b"}, Host: "b.example.com", TCPPort: 443},
	}
	h := NewServerWithFullDependencies(
		NewRegistry(),
		nil,
		targets,
		NewSocialHub(),
		VoiceIceProvider{},
	).Routes()

	req := httptest.NewRequest(http.MethodGet, "/v1/route-targets?package_name=game.b", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var payload []GameRouteTarget
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 1 || payload[0].ID != "b" {
		t.Fatalf("unexpected route targets: %#v", payload)
	}
}
