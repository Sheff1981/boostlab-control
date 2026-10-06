package api

import (
	"sort"
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
	UserID   string    `json:"user_id"`
	SeenAt   time.Time `json:"seen_at"`
}

type SocialHub struct {
	mu       sync.RWMutex
	nextID   uint64
	events   map[string][]SquadEvent
	presence map[string]map[string]time.Time
}

func NewSocialHub() *SocialHub {
	return &SocialHub{
		events:   make(map[string][]SquadEvent),
		presence: make(map[string]map[string]time.Time),
	}
}

func (h *SocialHub) Append(room, sender, eventType, text, payload string) SquadEvent {
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
	return event
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
