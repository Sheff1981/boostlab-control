package api

import "testing"

func TestDirectRoomIsSymmetric(t *testing.T) {
	left, ok := directRoom("BL-A123", "BL-B456")
	if !ok {
		t.Fatal("expected valid room")
	}
	right, ok := directRoom("BL-B456", "BL-A123")
	if !ok || left != right {
		t.Fatalf("direct room must be symmetric: %q vs %q", left, right)
	}
}

func TestDirectRoomRejectsSelf(t *testing.T) {
	if _, ok := directRoom("BL-A123", "BL-A123"); ok {
		t.Fatal("expected self room rejection")
	}
}
