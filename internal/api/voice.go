package api

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type IceServerConfig struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

type VoiceIceConfig struct {
	IceServers []IceServerConfig `json:"ice_servers"`
}

type VoiceIceProvider struct {
	TurnURLs []string
	Secret   string
	TTL      time.Duration
}

func ParseVoiceIceProvider(turnURLsRaw, secret, ttlSecondsRaw string) (VoiceIceProvider, error) {
	turnURLsRaw = strings.TrimSpace(turnURLsRaw)
	secret = strings.TrimSpace(secret)

	ttl := 30 * time.Minute
	if raw := strings.TrimSpace(ttlSecondsRaw); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds < 60 || seconds > 86400 {
			return VoiceIceProvider{}, fmt.Errorf("TURN TTL must be 60..86400 seconds")
		}
		ttl = time.Duration(seconds) * time.Second
	}

	if turnURLsRaw == "" {
		return VoiceIceProvider{TTL: ttl}, nil
	}
	if secret == "" {
		return VoiceIceProvider{}, fmt.Errorf("TURN secret is required when TURN URLs are configured")
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
			return VoiceIceProvider{}, fmt.Errorf("invalid TURN URL %q", value)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		urls = append(urls, value)
	}
	if len(urls) == 0 {
		return VoiceIceProvider{}, fmt.Errorf("no valid TURN URLs configured")
	}

	return VoiceIceProvider{
		TurnURLs: urls,
		Secret:   secret,
		TTL:      ttl,
	}, nil
}

func (p VoiceIceProvider) ConfigFor(userID string, now time.Time) (VoiceIceConfig, error) {
	config := VoiceIceConfig{
		IceServers: []IceServerConfig{
			{URLs: []string{"stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"}},
		},
	}
	if len(p.TurnURLs) == 0 {
		return config, nil
	}
	if !validUserID(userID) {
		return VoiceIceConfig{}, fmt.Errorf("valid user id is required for TURN credentials")
	}

	expires := now.UTC().Add(p.TTL).Unix()
	username := fmt.Sprintf("%d:%s", expires, userID)
	mac := hmac.New(sha1.New, []byte(p.Secret))
	_, _ = mac.Write([]byte(username))
	credential := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	config.IceServers = append(config.IceServers, IceServerConfig{
		URLs:       append([]string(nil), p.TurnURLs...),
		Username:   username,
		Credential: credential,
	})
	return config, nil
}
