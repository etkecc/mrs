package matrix

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/matrix-org/gomatrixserverlib"

	"github.com/etkecc/mrs/internal/model"
)

const keysTestServerName = "example.org"

func signedServerKeys(t *testing.T) ([]byte, *model.ServerKeys) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	keys := model.ServerKeys{
		ServerName:   keysTestServerName,
		ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
		VerifyKeys: map[string]map[string]string{
			"ed25519:test": {"key": base64.RawStdEncoding.EncodeToString(publicKey)},
		},
	}
	raw, err := json.Marshal(keys)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	signed, err := gomatrixserverlib.SignJSON(keysTestServerName, gomatrixserverlib.KeyID("ed25519:test"), privateKey, raw)
	if err != nil {
		t.Fatalf("SignJSON: %v", err)
	}
	var parsed model.ServerKeys
	if err := json.Unmarshal(signed, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return signed, &parsed
}

func TestVerifyServerKeys(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		if err := verifyServerKeys(t.Context(), signed, parsed); err != nil {
			t.Fatalf("valid response rejected: %v", err)
		}
	})

	t.Run("renamed claim", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		parsed.ServerName = "other.example"
		if err := verifyServerKeys(t.Context(), signed, parsed); err == nil {
			t.Fatal("renamed claim accepted")
		}
	})

	t.Run("empty server name", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		parsed.ServerName = ""
		if err := verifyServerKeys(t.Context(), signed, parsed); err == nil {
			t.Fatal("empty server_name accepted")
		}
	})

	t.Run("unsigned", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		parsed.Signatures = nil
		if err := verifyServerKeys(t.Context(), signed, parsed); err == nil {
			t.Fatal("unsigned response accepted")
		}
	})

	t.Run("tampered payload", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		tampered := bytes.Replace(signed, []byte(`"valid_until_ts":`), []byte(`"valid_until_ts":1`), 1)
		if bytes.Equal(tampered, signed) {
			t.Fatal("payload was not tampered")
		}
		if err := verifyServerKeys(t.Context(), tampered, parsed); err == nil {
			t.Fatal("tampered payload accepted")
		}
	})

	t.Run("tampered verify key", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		otherPublic, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("GenerateKey: %v", err)
		}
		parsed.VerifyKeys["ed25519:test"]["key"] = base64.RawStdEncoding.EncodeToString(otherPublic)
		if err := verifyServerKeys(t.Context(), signed, parsed); err == nil {
			t.Fatal("tampered verify key accepted")
		}
	})

	t.Run("malformed verify key", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		parsed.VerifyKeys["ed25519:test"]["key"] = "!!!"
		if err := verifyServerKeys(t.Context(), signed, parsed); err == nil {
			t.Fatal("malformed verify key accepted")
		}
	})

	t.Run("unknown key", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		parsed.VerifyKeys = nil
		if err := verifyServerKeys(t.Context(), signed, parsed); err == nil {
			t.Fatal("response with no published signing key accepted")
		}
	})

	t.Run("malformed extra verify key", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		parsed.VerifyKeys["ed25519:extra"] = map[string]string{"key": "!!!"}
		if err := verifyServerKeys(t.Context(), signed, parsed); err == nil {
			t.Fatal("malformed extra verify key accepted")
		}
	})

	t.Run("unsupported key algorithm", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		parsed.VerifyKeys["rsa:test"] = parsed.VerifyKeys["ed25519:test"]
		if err := verifyServerKeys(t.Context(), signed, parsed); err == nil {
			t.Fatal("unsupported key algorithm accepted")
		}
	})

	t.Run("too many signatures", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		signatures := parsed.Signatures[keysTestServerName]
		for i := len(signatures); i <= maxServerKeysSignatures; i++ {
			signatures[fmt.Sprintf("ed25519:extra%d", i)] = ""
		}
		if err := verifyServerKeys(t.Context(), signed, parsed); err == nil {
			t.Fatal("oversized signature set accepted")
		}
	})

	t.Run("too many verify keys", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		for i := len(parsed.VerifyKeys); i <= maxServerKeysVerifyKeys; i++ {
			parsed.VerifyKeys[fmt.Sprintf("ed25519:extra%d", i)] = parsed.VerifyKeys["ed25519:test"]
		}
		if err := verifyServerKeys(t.Context(), signed, parsed); err == nil {
			t.Fatal("oversized verify key set accepted")
		}
	})

	t.Run("invalid signature before valid", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		parsed.VerifyKeys["ed25519:bad"] = parsed.VerifyKeys["ed25519:test"]
		parsed.Signatures[keysTestServerName]["ed25519:bad"] = ""
		if err := verifyServerKeys(t.Context(), signed, parsed); err != nil {
			t.Fatalf("valid signature rejected alongside an invalid one: %v", err)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		signed, parsed := signedServerKeys(t)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if err := verifyServerKeys(ctx, signed, parsed); err == nil {
			t.Fatal("canceled context accepted")
		}
	})

	t.Run("oversized body", func(t *testing.T) {
		_, parsed := signedServerKeys(t)
		if err := verifyServerKeys(t.Context(), make([]byte, maxServerKeysBytes+1), parsed); err == nil {
			t.Fatal("oversized body accepted")
		}
	})
}

func TestDecodePublicKeys(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	keys := decodePublicKeys(map[string]map[string]string{
		"ed25519:good":  {"key": base64.RawStdEncoding.EncodeToString(publicKey)},
		"ed25519:bad":   {"key": "!!!"},
		"ed25519:short": {"key": base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize-1))},
	})
	if len(keys) != 1 {
		t.Fatalf("expected 1 valid key, got %d", len(keys))
	}
	if len(keys["ed25519:good"]) != ed25519.PublicKeySize {
		t.Fatal("valid key missing")
	}
}

func TestValidateAuthRejectsShortKey(t *testing.T) {
	s := &Server{}
	obj := map[string]any{"origin": "example.org", "destination": "mrs.example"}
	auth := &matrixAuth{Origin: "example.org", KeyID: "ed25519:short"}
	keys := map[string]ed25519.PublicKey{"ed25519:short": make([]byte, ed25519.PublicKeySize-1)}
	if err := s.validateAuth(obj, nil, auth, keys); err == nil {
		t.Fatal("short public key accepted")
	}
}
