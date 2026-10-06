package api

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

func directRoom(userA, userB string) (string, bool) {
	if !validUserID(userA) || !validUserID(userB) || userA == userB {
		return "", false
	}
	users := []string{userA, userB}
	sort.Strings(users)
	sum := sha256.Sum256([]byte(users[0] + "\x00" + users[1]))
	return "DM-" + hex.EncodeToString(sum[:16]), true
}
