package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

type DeviceChallenge struct {
	ID        string
	Nonce     string
	Message   string
	PublicKey *ecdsa.PublicKey
	ExpiresAt time.Time
}

type DeviceSession struct {
	Token     string
	DeviceID  string
	ExpiresAt time.Time
}

type DeviceAuthHub struct {
	mu         sync.Mutex
	challenges map[string]DeviceChallenge
	sessions   map[string]DeviceSession
}

func NewDeviceAuthHub() *DeviceAuthHub {
	return &DeviceAuthHub{
		challenges: make(map[string]DeviceChallenge),
		sessions:   make(map[string]DeviceSession),
	}
}

func (h *DeviceAuthHub) NewChallenge(publicKeyBase64 string, now time.Time) (DeviceChallenge, error) {
	publicKey, err := parseDevicePublicKey(publicKeyBase64)
	if err != nil {
		return DeviceChallenge{}, err
	}

	challengeID, err := randomToken(18)
	if err != nil {
		return DeviceChallenge{}, err
	}
	nonce, err := randomToken(32)
	if err != nil {
		return DeviceChallenge{}, err
	}

	message := "BOOSTLAB-AUTH-V1\n" + challengeID + "\n" + nonce
	challenge := DeviceChallenge{
		ID:        challengeID,
		Nonce:     nonce,
		Message:   message,
		PublicKey: publicKey,
		ExpiresAt: now.Add(2 * time.Minute),
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanupLocked(now)
	h.challenges[challenge.ID] = challenge
	return challenge, nil
}

func (h *DeviceAuthHub) Exchange(challengeID, signatureBase64 string, now time.Time) (DeviceSession, error) {
	challengeID = strings.TrimSpace(challengeID)
	signatureBase64 = strings.TrimSpace(signatureBase64)
	if challengeID == "" || signatureBase64 == "" {
		return DeviceSession{}, fmt.Errorf("challenge_id and signature are required")
	}

	h.mu.Lock()
	challenge, ok := h.challenges[challengeID]
	if ok {
		delete(h.challenges, challengeID)
	}
	h.cleanupLocked(now)
	h.mu.Unlock()

	if !ok || !challenge.ExpiresAt.After(now) {
		return DeviceSession{}, fmt.Errorf("challenge expired or unknown")
	}

	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil || len(signature) == 0 {
		return DeviceSession{}, fmt.Errorf("invalid signature encoding")
	}

	digest := sha256.Sum256([]byte(challenge.Message))
	if !ecdsa.VerifyASN1(challenge.PublicKey, digest[:], signature) {
		return DeviceSession{}, fmt.Errorf("invalid device signature")
	}

	deviceID := deviceID(challenge.PublicKey)
	token, err := randomToken(32)
	if err != nil {
		return DeviceSession{}, err
	}

	session := DeviceSession{
		Token:     token,
		DeviceID:  deviceID,
		ExpiresAt: now.Add(15 * time.Minute),
	}

	h.mu.Lock()
	h.cleanupLocked(now)
	h.sessions[token] = session
	h.mu.Unlock()
	return session, nil
}

func (h *DeviceAuthHub) AuthenticateBearer(value string, now time.Time) (DeviceSession, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return DeviceSession{}, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(value, prefix))
	if token == "" {
		return DeviceSession{}, false
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanupLocked(now)
	session, ok := h.sessions[token]
	if !ok || !session.ExpiresAt.After(now) {
		return DeviceSession{}, false
	}
	return session, true
}

func (h *DeviceAuthHub) cleanupLocked(now time.Time) {
	for id, challenge := range h.challenges {
		if !challenge.ExpiresAt.After(now) {
			delete(h.challenges, id)
		}
	}
	for token, session := range h.sessions {
		if !session.ExpiresAt.After(now) {
			delete(h.sessions, token)
		}
	}
}

func parseDevicePublicKey(raw string) (*ecdsa.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid public key encoding")
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("invalid public key")
	}
	publicKey, ok := parsed.(*ecdsa.PublicKey)
	if !ok || publicKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("device key must be ECDSA P-256")
	}
	return publicKey, nil
}

func deviceID(publicKey *ecdsa.PublicKey) string {
	der, _ := x509.MarshalPKIXPublicKey(publicKey)
	sum := sha256.Sum256(der)
	return "BLDEV-" + strings.ToUpper(hex.EncodeToString(sum[:10]))
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("secure random failed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
