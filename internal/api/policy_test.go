package api

import "testing"

func TestDefaultClientPolicy(t *testing.T) {
	policy := DefaultClientPolicy()

	if policy.DefaultTier != "free" {
		t.Fatalf("expected free default tier, got %q", policy.DefaultTier)
	}
	if !policy.Free.AdsEnabled {
		t.Fatal("expected ads enabled for free tier")
	}
	if policy.Premium.AdsEnabled {
		t.Fatal("expected ads disabled for premium tier")
	}
	if policy.Free.MaxAutoCandidates < 1 {
		t.Fatal("free tier must allow at least one route candidate")
	}
}
