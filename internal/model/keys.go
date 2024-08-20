package model

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

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
