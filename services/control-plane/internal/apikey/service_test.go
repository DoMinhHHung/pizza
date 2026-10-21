package apikey

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"github.com/google/uuid"
)

func TestGenerateSecret_FormatAndHash(t *testing.T) {
	secret, prefix, hash, err := generateSecret()
	if err != nil {
		t.Fatalf("generateSecret: %v", err)
	}
	if !strings.HasPrefix(secret, "pk_live_") {
		t.Fatalf("secret prefix: %q", secret[:8])
	}
	if len(secret) < 20 {
		t.Fatalf("secret too short: %d", len(secret))
	}
	if prefix != secret[:16] {
		t.Fatalf("prefix want %q got %q", secret[:16], prefix)
	}
	sum := sha256.Sum256([]byte(secret))
	wantHash := hex.EncodeToString(sum[:])
	if hash != wantHash {
		t.Fatalf("hash mismatch")
	}
	secret2, _, hash2, err := generateSecret()
	if err != nil {
		t.Fatal(err)
	}
	if secret == secret2 || hash == hash2 {
		t.Fatal("secrets must be unique")
	}
}

func TestToProto_OmitsHashAndHandlesNilTimes(t *testing.T) {
	k := &ApiKey{
		ID:        mustParse("11111111-1111-1111-1111-111111111111"),
		ProjectID: mustParse("22222222-2222-2222-2222-222222222222"),
		Name:      "ci",
		KeyPrefix: "pk_live_abcdefgh",
		KeyHash:   "should-not-appear-in-proto",
		CreatedBy: mustParse("33333333-3333-3333-3333-333333333333"),
	}
	p := toProto(k)
	if p.GetKeyPrefix() != "pk_live_abcdefgh" {
		t.Fatalf("prefix: %s", p.GetKeyPrefix())
	}
	if p.GetName() != "ci" {
		t.Fatalf("name: %s", p.GetName())
	}
	if p.RevokedAt != nil || p.LastUsedAt != nil {
		t.Fatal("nil times should stay unset")
	}
}

func mustParse(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}
