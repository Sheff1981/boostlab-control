package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type DeviceChallenge struct {
	ID                string
	Nonce             string
	Message           string
	PublicKey         *ecdsa.PublicKey
	PublicKeyBase64   string
	DeviceID          string
	EnrollmentAllowed bool
	ExpiresAt         time.Time
}

type DeviceSession struct {
	Token     string
	DeviceID  string
	ExpiresAt time.Time
}

type DeviceAuthHub struct {
	mu             sync.Mutex
	challenges     map[string]DeviceChallenge
	sessions       map[string]DeviceSession
	authorizedKeys map[string]string
	dataFile       string
	enrollmentCode string
	openEnrollment bool
}

func NewDeviceAuthHub() *DeviceAuthHub {
	return &DeviceAuthHub{
		challenges:     make(map[string]DeviceChallenge),
		sessions:       make(map[string]DeviceSession),
		authorizedKeys: make(map[string]string),
		openEnrollment: true,
	}
}

func NewPersistentDeviceAuthHub(dataFile, enrollmentCode string) (*DeviceAuthHub, error) {
	enrollmentCode = strings.TrimSpace(enrollmentCode)
	if enrollmentCode != "" && len(enrollmentCode) < 12 {
		return nil, fmt.Errorf("BOOSTLAB_ENROLLMENT_CODE must be at least 12 characters")
	}

	hub := &DeviceAuthHub{
		challenges:     make(map[string]DeviceChallenge),
		sessions:       make(map[string]DeviceSession),
		authorizedKeys: make(map[string]string),
		dataFile:       strings.TrimSpace(dataFile),
		enrollmentCode: enrollmentCode,
		openEnrollment: false,
	}
	if err := hub.loadAuthorized(); err != nil {
		return nil, err
	}
	return hub, nil
}

func (h *DeviceAuthHub) NewChallenge(publicKeyBase64 string, now time.Time) (DeviceChallenge, error) {
	return h.NewChallengeWithEnrollment(publicKeyBase64, "", now)
}

func (h *DeviceAuthHub) NewChallengeWithEnrollment(
	publicKeyBase64 string,
	enrollmentCode string,
	now time.Time,
) (DeviceChallenge, error) {
	publicKeyBase64 = strings.TrimSpace(publicKeyBase64)
	publicKey, err := parseDevicePublicKey(publicKeyBase64)
	if err != nil {
		return DeviceChallenge{}, err
	}
	id := deviceID(publicKey)

	challengeID, err := randomToken(18)
	if err != nil {
		return DeviceChallenge{}, err
	}
	nonce, err := randomToken(32)
	if err != nil {
		return DeviceChallenge{}, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanupLocked(now)

	authorized := h.authorizedKeys[id] == publicKeyBase64
	enrollmentAllowed := authorized || h.openEnrollment || constantTimeEqual(
		strings.TrimSpace(enrollmentCode),
		h.enrollmentCode,
	)

	message := "BOOSTLAB-AUTH-V1\n" + challengeID + "\n" + nonce
	challenge := DeviceChallenge{
		ID:                challengeID,
		Nonce:             nonce,
		Message:           message,
		PublicKey:         publicKey,
		PublicKeyBase64:   publicKeyBase64,
		DeviceID:          id,
		EnrollmentAllowed: enrollmentAllowed,
		ExpiresAt:         now.Add(2 * time.Minute),
	}
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

	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanupLocked(now)

	authorized := h.authorizedKeys[challenge.DeviceID] == challenge.PublicKeyBase64
	if !authorized {
		if !challenge.EnrollmentAllowed {
			return DeviceSession{}, fmt.Errorf("device is not enrolled")
		}
		h.authorizedKeys[challenge.DeviceID] = challenge.PublicKeyBase64
		if err := h.saveAuthorizedLocked(); err != nil {
			delete(h.authorizedKeys, challenge.DeviceID)
			return DeviceSession{}, err
		}
	}

	token, err := randomToken(32)
	if err != nil {
		return DeviceSession{}, err
	}
	session := DeviceSession{
		Token:     token,
		DeviceID:  challenge.DeviceID,
		ExpiresAt: now.Add(15 * time.Minute),
	}
	h.sessions[token] = session
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

func (h *DeviceAuthHub) loadAuthorized() error {
	if h.dataFile == "" {
		return nil
	}
	raw, err := os.ReadFile(h.dataFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read device auth registry: %w", err)
	}

	var stored map[string]string
	if err := json.Unmarshal(raw, &stored); err != nil {
		return fmt.Errorf("parse device auth registry: %w", err)
	}
	for id, publicKey := range stored {
		id = strings.TrimSpace(id)
		publicKey = strings.TrimSpace(publicKey)
		if id == "" || publicKey == "" {
			continue
		}
		h.authorizedKeys[id] = publicKey
	}
	return nil
}

func (h *DeviceAuthHub) saveAuthorizedLocked() error {
	if h.dataFile == "" {
		return nil
	}
	raw, err := json.MarshalIndent(h.authorizedKeys, "", "  ")
	if err != nil {
		return fmt.Errorf("encode device auth registry: %w", err)
	}
	raw = append(raw, '\n')

	if err := os.MkdirAll(filepath.Dir(h.dataFile), 0o750); err != nil {
		return fmt.Errorf("create device auth directory: %w", err)
	}
	tmp := h.dataFile + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write device auth registry: %w", err)
	}
	if err := os.Rename(tmp, h.dataFile); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace device auth registry: %w", err)
	}
	return nil
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

func constantTimeEqual(value, expected string) bool {
	if expected == "" || len(value) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("secure random failed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
