package api

import (
	"fmt"
	"strings"
)

type IceServerConfig struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

type VoiceIceConfig struct {
	IceServers []IceServerConfig `json:"ice_servers"`
}

func ParseVoiceIceConfig(turnURLsRaw, username, credential string) (VoiceIceConfig, error) {
	config := VoiceIceConfig{
		IceServers: []IceServerConfig{
			{URLs: []string{"stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"}},
		},
	}

	turnURLsRaw = strings.TrimSpace(turnURLsRaw)
	username = strings.TrimSpace(username)
	credential = strings.TrimSpace(credential)
	if turnURLsRaw == "" {
		return config, nil
	}
	if username == "" || credential == "" {
		return VoiceIceConfig{}, fmt.Errorf("TURN username and credential are required when TURN URLs are configured")
	}

	parts := strings.Split(turnURLsRaw, ",")
	urls := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, item := range parts {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, "turn:") && !strings.HasPrefix(value, "turns:") {
			return VoiceIceConfig{}, fmt.Errorf("invalid TURN URL %q", value)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		urls = append(urls, value)
	}
	if len(urls) == 0 {
		return VoiceIceConfig{}, fmt.Errorf("no valid TURN URLs configured")
	}

	config.IceServers = append(config.IceServers, IceServerConfig{
		URLs:       urls,
		Username:   username,
		Credential: credential,
	})
	return config, nil
}
