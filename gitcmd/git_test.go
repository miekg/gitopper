package gitcmd

import (
	"encoding/hex"
	"testing"
)

func TestHash(t *testing.T) {
	g := New("", ".", nil)

	hash, err := g.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if len(hash) != 40 {
		t.Fatalf("Hash length should be 40, got %d", len(hash))
	}

	if _, err := hex.DecodeString(hash); err != nil {
		t.Fatalf("Failed to decode hash: %s", err)
	}
}
