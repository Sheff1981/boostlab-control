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
