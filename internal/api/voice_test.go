package api

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestVoiceIceProviderDefaultsToStun(t *testing.T) {
	provider, err := ParseVoiceIceProvider("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	config, err := provider.ConfigFor("", time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(config.IceServers) != 1 || len(config.IceServers[0].URLs) < 1 {
		t.Fatalf("unexpected ICE config: %#v", config)
	}
}

func TestVoiceIceProviderIssuesShortLivedTurnCredential(t *testing.T) {
	provider, err := ParseVoiceIceProvider(
		"turn:turn.example.com:3478,turns:turn.example.com:5349",
		"shared-secret",
		"600",
	)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0)
	config, err := provider.ConfigFor("BL-ABC123", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.IceServers) != 2 {
		t.Fatalf("expected STUN + TURN, got %#v", config)
	}
	turn := config.IceServers[1]
	if len(turn.URLs) != 2 {
		t.Fatalf("expected two TURN URLs, got %#v", turn.URLs)
	}
	if !strings.HasPrefix(turn.Username, "1700000600:BL-ABC123") {
		t.Fatalf("unexpected TURN username: %q", turn.Username)
	}
	if _, err := base64.StdEncoding.DecodeString(turn.Credential); err != nil {
		t.Fatalf("credential is not base64: %v", err)
	}
}

func TestVoiceIceProviderRejectsMissingSecret(t *testing.T) {
	if _, err := ParseVoiceIceProvider("turn:turn.example.com:3478", "", ""); err == nil {
		t.Fatal("expected missing TURN secret error")
	}
}

func TestVoiceIceProviderRejectsInvalidTTL(t *testing.T) {
	if _, err := ParseVoiceIceProvider("", "", "10"); err == nil {
		t.Fatal("expected invalid TURN TTL error")
	}
}
