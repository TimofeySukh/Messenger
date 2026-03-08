package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

func GenerateEncryptionKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func IsValidKey(keyBase64 string) bool {
	decoded, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return false
	}
	return len(decoded) == 32
}

func decodeKey(keyBase64 string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, errors.New("invalid encryption key format")
	}
	if len(decoded) != 32 {
		return nil, errors.New("encryption key must decode to 32 bytes")
	}
	return decoded, nil
}

func Encrypt(plaintext string, keyBase64 string) (string, error) {
	key, err := decodeKey(keyBase64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(ciphertextBase64 string, keyBase64 string) (string, error) {
	key, err := decodeKey(keyBase64)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", errors.New("invalid encrypted payload")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("encrypted payload is too short")
	}

	nonce := ciphertext[:nonceSize]
	payload := ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", errors.New("decryption failed")
	}

	return string(plaintext), nil
}
