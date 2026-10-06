package api

import (
	"fmt"
	"testing"
)

func TestSocialHubEventsAreOrderedAndIncremental(t *testing.T) {
	hub := NewSocialHub()
	first := hub.Append("SQ-1", "BL-A", "chat", "hello", "")
	second := hub.Append("SQ-1", "BL-B", "chat", "world", "")

	if second.ID <= first.ID {
		t.Fatalf("expected increasing IDs: first=%d second=%d", first.ID, second.ID)
	}

	items := hub.List("SQ-1", first.ID)
	if len(items) != 1 || items[0].ID != second.ID {
		t.Fatalf("expected only second event, got %#v", items)
	}
}

func TestSocialHubBoundsRoomHistory(t *testing.T) {
	hub := NewSocialHub()
	for i := 0; i < maxRoomEvents+25; i++ {
		hub.Append("SQ-1", "BL-A", "chat", fmt.Sprintf("%d", i), "")
	}
	items := hub.List("SQ-1", 0)
	if len(items) != maxRoomEvents {
		t.Fatalf("expected %d retained events, got %d", maxRoomEvents, len(items))
	}
}

func TestSocialHubPresence(t *testing.T) {
	hub := NewSocialHub()
	hub.Touch("SQ-1", "BL-B")
	hub.Touch("SQ-1", "BL-A")
	items := hub.Presence("SQ-1")
	if len(items) != 2 || items[0].UserID != "BL-A" || items[1].UserID != "BL-B" {
		t.Fatalf("unexpected presence: %#v", items)
	}
}
