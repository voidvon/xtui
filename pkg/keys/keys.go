package keys

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/curve25519"
)

// GenerateX25519 creates a new X25519 keypair for Reality.
func GenerateX25519() (privateKeyBase64, publicKeyBase64 string, err error) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return "", "", err
	}
	// curve25519 clamping
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)

	return base64.RawURLEncoding.EncodeToString(priv[:]),
		base64.RawURLEncoding.EncodeToString(pub[:]),
		nil
}

// DerivePublicKey derives the corresponding X25519 public key from a base64 private key.
func DerivePublicKey(privateKeyBase64 string) (string, error) {
	priv, err := base64.RawURLEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		// try standard base64 if raw url fails
		priv, err = base64.StdEncoding.DecodeString(privateKeyBase64)
		if err != nil {
			return "", fmt.Errorf("invalid base64 key: %w", err)
		}
	}
	if len(priv) != 32 {
		return "", fmt.Errorf("private key length must be 32 bytes (got %d)", len(priv))
	}
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, (*[32]byte)(priv))
	return base64.RawURLEncoding.EncodeToString(pub[:]), nil
}

// GenerateShortId generates an 8-byte random hex string.
func GenerateShortId() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// GenerateUUID generates a random standard UUID v4 string.
func GenerateUUID() string {
	return uuid.New().String()
}
