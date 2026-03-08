package room

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

const tokenLength = 10

func NewToken() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate room token: %w", err)
	}

	encoded := strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), "=")
	if len(encoded) < tokenLength {
		return "", fmt.Errorf("generated token is too short")
	}
	return encoded[:tokenLength], nil
}
