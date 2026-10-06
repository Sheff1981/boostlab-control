package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxRoomEvents = 200
	presenceTTL   = 45 * time.Second
)

type SquadEvent struct {
	ID        uint64    `json:"id"`
	Sender    string    `json:"sender"`
	Type      string    `json:"type"`
	Text      string    `json:"text,omitempty"`
	Payload   string    `json:"payload,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type SquadPresence struct {
	UserID string    `json:"user_id"`
	SeenAt time.Time `json:"seen_at"`
}

type socialSnapshot struct {
	NextID uint64                   `json:"next_id"`
	Events map[string][]SquadEvent `json:"events"`
}

type SocialHub struct {
	mu          sync.RWMutex
	nextID      uint64
	events      map[string][]SquadEvent
	presence    map[string]map[string]time.Time
	storagePath string
}

func NewSocialHub() *SocialHub {
	return &SocialHub{
		events:   make(map[string][]SquadEvent),
		presence: make(map[string]map[string]time.Time),
	}
}

func NewPersistentSocialHub(path string) (*SocialHub, error) {
	hub := NewSocialHub()
	path = strings.TrimSpace(path)
	if path == "" {
		return hub, nil
	}
	hub.storagePath = path

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return hub, nil
		}
		return nil, fmt.Errorf("read social data: %w", err)
	}
	if len(raw) == 0 {
		return hub, nil
	}

	var snapshot socialSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("parse social data: %w", err)
	}

	hub.nextID = snapshot.NextID
	for room, items := range snapshot.Events {
		cleanRoom, ok := normalizeRoomCode(room)
		if !ok {
			continue
		}
		filtered := make([]SquadEvent, 0, len(items))
		for _, event := range items {
			if event.Type != "chat" || event.Text == "" || !validUserID(event.Sender) {
				continue
			}
			filtered = append(filtered, event)
			if event.ID > hub.nextID {
				hub.nextID = event.ID
			}
		}
		if len(filtered) > maxRoomEvents {
			filtered = filtered[len(filtered)-maxRoomEvents:]
		}
		if len(filtered) > 0 {
			hub.events[cleanRoom] = append([]SquadEvent(nil), filtered...)
		}
	}

	return hub, nil
}

func (h *SocialHub) Append(room, sender, eventType, text, payload string) (SquadEvent, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	event := SquadEvent{
		ID:        h.nextID,
		Sender:    sender,
		Type:      eventType,
		Text:      text,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}

	items := append(h.events[room], event)
	if len(items) > maxRoomEvents {
		items = append([]SquadEvent(nil), items[len(items)-maxRoomEvents:]...)
	}
	h.events[room] = items

	if eventType == "chat" {
		if err := h.persistLocked(); err != nil {
			return event, err
		}
	}
	return event, nil
}

func (h *SocialHub) List(room string, after uint64) []SquadEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()

	items := h.events[room]
	out := make([]SquadEvent, 0, len(items))
	for _, event := range items {
		if event.ID > after {
			out = append(out, event)
		}
	}
	return out
}

func (h *SocialHub) Touch(room, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.presence[room] == nil {
		h.presence[room] = make(map[string]time.Time)
	}
	h.presence[room][userID] = time.Now().UTC()
}

func (h *SocialHub) Presence(room string) []SquadPresence {
	h.mu.Lock()
	defer h.mu.Unlock()

	cutoff := time.Now().UTC().Add(-presenceTTL)
	users := h.presence[room]
	out := make([]SquadPresence, 0, len(users))
	for userID, seenAt := range users {
		if seenAt.Before(cutoff) {
			delete(users, userID)
			continue
		}
		out = append(out, SquadPresence{UserID: userID, SeenAt: seenAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UserID < out[j].UserID })
	return out
}

func (h *SocialHub) persistLocked() error {
	if h.storagePath == "" {
		return nil
	}

	snapshot := socialSnapshot{
		NextID: h.nextID,
		Events: make(map[string][]SquadEvent),
	}
	for room, items := range h.events {
		chats := make([]SquadEvent, 0, len(items))
		for _, event := range items {
			if event.Type == "chat" {
				chats = append(chats, event)
			}
		}
		if len(chats) > 0 {
			snapshot.Events[room] = chats
		}
	}

	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode social data: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(h.storagePath), 0o755); err != nil {
		return fmt.Errorf("create social data directory: %w", err)
	}

	tmp := h.storagePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write social data: %w", err)
	}

	// Windows cannot replace an existing file with os.Rename.
	_ = os.Remove(h.storagePath)
	if err := os.Rename(tmp, h.storagePath); err != nil {
		return fmt.Errorf("commit social data: %w", err)
	}
	return nil
}
