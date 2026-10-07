package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"testing"
	"time"
)

func makeDeviceKey(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return key, base64.StdEncoding.EncodeToString(der)
}

func signChallenge(t *testing.T, key *ecdsa.PrivateKey, message string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(message))
	signature, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(signature)
}

func TestDeviceAuthChallengeExchange(t *testing.T) {
	hub := NewDeviceAuthHub()
	key, publicKey := makeDeviceKey(t)
	now := time.Unix(1_800_000_000, 0)

	challenge, err := hub.NewChallenge(publicKey, now)
	if err != nil {
		t.Fatal(err)
	}
	session, err := hub.Exchange(
		challenge.ID,
		signChallenge(t, key, challenge.Message),
		now.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	if session.DeviceID == "" || session.Token == "" {
		t.Fatalf("unexpected session: %#v", session)
	}
	if _, ok := hub.AuthenticateBearer("Bearer "+session.Token, now.Add(2*time.Second)); !ok {
		t.Fatal("expected bearer session to authenticate")
	}
}

func TestDeviceAuthChallengeIsOneTime(t *testing.T) {
	hub := NewDeviceAuthHub()
	key, publicKey := makeDeviceKey(t)
	now := time.Unix(1_800_000_000, 0)

	challenge, err := hub.NewChallenge(publicKey, now)
	if err != nil {
		t.Fatal(err)
	}
	signature := signChallenge(t, key, challenge.Message)

	if _, err := hub.Exchange(challenge.ID, signature, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := hub.Exchange(challenge.ID, signature, now.Add(2*time.Second)); err == nil {
		t.Fatal("expected replayed challenge to fail")
	}
}

func TestDeviceAuthRejectsWrongSignatureAndExpiredChallenge(t *testing.T) {
	hub := NewDeviceAuthHub()
	key, publicKey := makeDeviceKey(t)
	other, _ := makeDeviceKey(t)
	now := time.Unix(1_800_000_000, 0)

	challenge, err := hub.NewChallenge(publicKey, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hub.Exchange(
		challenge.ID,
		signChallenge(t, other, challenge.Message),
		now.Add(time.Second),
	); err == nil {
		t.Fatal("expected wrong signature to fail")
	}

	challenge, err = hub.NewChallenge(publicKey, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hub.Exchange(
		challenge.ID,
		signChallenge(t, key, challenge.Message),
		now.Add(3*time.Minute),
	); err == nil {
		t.Fatal("expected expired challenge to fail")
	}
}


func TestPersistentDeviceEnrollment(t *testing.T) {
	dataFile := t.TempDir() + "/devices.json"
	const enrollmentCode = "boostlab-join-2026"
	now := time.Unix(1_800_000_000, 0)
	key, publicKey := makeDeviceKey(t)

	hub, err := NewPersistentDeviceAuthHub(dataFile, enrollmentCode)
	if err != nil {
		t.Fatal(err)
	}

	challenge, err := hub.NewChallengeWithEnrollment(publicKey, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hub.Exchange(
		challenge.ID,
		signChallenge(t, key, challenge.Message),
		now.Add(time.Second),
	); err == nil {
		t.Fatal("expected unenrolled device without code to fail")
	}

	challenge, err = hub.NewChallengeWithEnrollment(publicKey, enrollmentCode, now)
	if err != nil {
		t.Fatal(err)
	}
	session, err := hub.Exchange(
		challenge.ID,
		signChallenge(t, key, challenge.Message),
		now.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	if session.DeviceID == "" {
		t.Fatal("expected enrolled device id")
	}

	restarted, err := NewPersistentDeviceAuthHub(dataFile, "")
	if err != nil {
		t.Fatal(err)
	}
	challenge, err = restarted.NewChallengeWithEnrollment(
		publicKey,
		"",
		now.Add(2*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Exchange(
		challenge.ID,
		signChallenge(t, key, challenge.Message),
		now.Add(3*time.Second),
	); err != nil {
		t.Fatalf("persisted device should authenticate without code: %v", err)
	}
}
