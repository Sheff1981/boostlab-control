package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ProvisioningClaims struct {
	NodeID        string `json:"node_id"`
	DeviceID      string `json:"device_id"`
	WireGuardKey  string `json:"wireguard_public_key"`
	ExpiresAtUnix int64  `json:"exp"`
	Nonce         string `json:"nonce"`
}

func issueProvisioningTicket(
	secret []byte,
	nodeID string,
	deviceID string,
	wireGuardPublicKey string,
	now time.Time,
) (string, time.Time, error) {
	if len(secret) < 32 {
		return "", time.Time{}, fmt.Errorf("provisioning secret must be at least 32 bytes")
	}
	nodeID = strings.TrimSpace(nodeID)
	deviceID = strings.TrimSpace(deviceID)
	wireGuardPublicKey = strings.TrimSpace(wireGuardPublicKey)
	if nodeID == "" || deviceID == "" || wireGuardPublicKey == "" {
		return "", time.Time{}, fmt.Errorf("ticket claims are incomplete")
	}

	nonce, err := randomToken(18)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := now.Add(90 * time.Second)
	claims := ProvisioningClaims{
		NodeID:        nodeID,
		DeviceID:      deviceID,
		WireGuardKey:  wireGuardPublicKey,
		ExpiresAtUnix: expiresAt.Unix(),
		Nonce:         nonce,
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}

	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payloadEncoded))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payloadEncoded + "." + signature, expiresAt, nil
}
