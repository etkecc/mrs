package model

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/goccy/go-json"
)

// ServerKeys is /_matrix/key/v2/server: a server's published signing keys, signed by that server.
// verify_keys is keyed by key ID by design (the spec key set is dynamic), so it stays a map, not lazy typing.
type ServerKeys struct {
	ServerName    string                       `json:"server_name"`          // the server these keys belong to
	ValidUntilTS  int64                        `json:"valid_until_ts"`       // keys are trustworthy until this ms timestamp
	VerifyKeys    map[string]map[string]string `json:"verify_keys"`          // keyID -> {"key": base64 ed25519 public key}
	OldVerifyKeys map[string]map[string]any    `json:"old_verify_keys"`      // rotated-out keys, kept so old signatures still verify
	Signatures    map[string]map[string]string `json:"signatures,omitempty"` // server -> keyID -> signature over this object
}

// ServerKeysQueryResponse is /_matrix/key/v2/query and /_matrix/key/v2/query/{serverName}:
// a notary handing back a batch of already-signed ServerKeys blobs verbatim, no re-wrapping.
type ServerKeysQueryResponse struct {
	ServerKeys []json.RawMessage `json:"server_keys"` // each element is a signed ServerKeys object
}

// Key is ed25519 key
type Key struct {
	ID      string
	Private ed25519.PrivateKey
	Public  string
}

// KeyFrom parses key from string
func KeyFrom(str string) (*Key, error) {
	parts := strings.Split(str, " ")
	if len(parts) != 3 || parts[0] != "ed25519" {
		return nil, fmt.Errorf("invalid key")
	}
	seed, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}

	priv := ed25519.NewKeyFromSeed(seed)
	pub, ok := priv.Public().(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("cannot cast public key")
	}

	return &Key{
		ID:      parts[0] + ":" + parts[1],
		Private: priv,
		Public:  base64.RawStdEncoding.EncodeToString(pub),
	}, nil
}
