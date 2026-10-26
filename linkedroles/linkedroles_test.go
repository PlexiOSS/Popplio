// Copyright (C) 2026 NodeByte LTD

package linkedroles

import (
	"bytes"
	"encoding/base64"
	"testing"

	"popplio/config"
	"popplio/state"
)

func withKey(t *testing.T, key string) {
	t.Helper()
	prev := state.Config
	state.Config = &config.Config{Meta: config.Meta{LinkedRolesKey: key}}
	t.Cleanup(func() { state.Config = prev })
}

func TestEncryptRoundTrip(t *testing.T) {
	withKey(t, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))

	sealed, err := encrypt("refresh-token-value")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("refresh-token-value")) {
		t.Fatal("sealed token contains the plaintext")
	}

	again, _ := encrypt("refresh-token-value")
	if bytes.Equal(sealed, again) {
		t.Fatal("two encryptions of the same token should differ (random nonce)")
	}

	plain, err := decrypt(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "refresh-token-value" {
		t.Fatalf("got %q", plain)
	}
}

func TestDecryptRejectsTampering(t *testing.T) {
	withKey(t, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32)))

	sealed, _ := encrypt("token")
	sealed[len(sealed)-1] ^= 0xff

	if _, err := decrypt(sealed); err == nil {
		t.Fatal("expected tampered ciphertext to fail")
	}
}

func TestBadKeys(t *testing.T) {
	withKey(t, base64.StdEncoding.EncodeToString([]byte("too short")))
	if _, err := encrypt("x"); err == nil {
		t.Fatal("expected a short key to be rejected")
	}

	withKey(t, "not base64!!")
	if _, err := encrypt("x"); err == nil {
		t.Fatal("expected invalid base64 to be rejected")
	}

	withKey(t, "")
	if Enabled() {
		t.Fatal("an empty key should disable linked roles")
	}
}
