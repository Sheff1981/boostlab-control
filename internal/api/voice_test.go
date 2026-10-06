package api

import "testing"

func TestParseVoiceIceConfigDefaultsToStun(t *testing.T) {
	config, err := ParseVoiceIceConfig("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(config.IceServers) != 1 || len(config.IceServers[0].URLs) < 1 {
		t.Fatalf("unexpected ICE config: %#v", config)
	}
}

func TestParseVoiceIceConfigAddsTurn(t *testing.T) {
	config, err := ParseVoiceIceConfig(
		"turn:turn.example.com:3478,turns:turn.example.com:5349",
		"boostlab",
		"secret",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.IceServers) != 2 {
		t.Fatalf("expected STUN + TURN, got %#v", config)
	}
	if config.IceServers[1].Username != "boostlab" || config.IceServers[1].Credential != "secret" {
		t.Fatalf("unexpected TURN credentials: %#v", config.IceServers[1])
	}
}

func TestParseVoiceIceConfigRejectsMissingCredentials(t *testing.T) {
	if _, err := ParseVoiceIceConfig("turn:turn.example.com:3478", "", ""); err == nil {
		t.Fatal("expected missing TURN credentials error")
	}
}
