package main

// ============================================================
// ENCRYPTION MODULE
// AES-256-GCM End-to-End Encryption
// ============================================================

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// ============================================================
// KEY GENERATION
// ============================================================

// GenerateEncryptionKey creates a random 256-bit (32 bytes) key
//
// How it works:
//  1. Create a 32-byte slice (32 * 8 = 256 bits)
//  2. Fill it with cryptographically secure random bytes
//  3. Return as Base64 string (human-readable)
//
// Why Base64?
//
//	Raw bytes can contain unprintable characters (0x00, 0x1F, etc.)
//	Base64 converts bytes → letters/numbers/+/= (safe to copy/paste)
//	Example: [0x48, 0x65, 0x6C] → "SGVs"
func GenerateEncryptionKey() (string, error) {
	// Create a slice for our key (32 bytes = 256 bits)
	key := make([]byte, 32)

	// crypto/rand.Read fills the slice with secure random bytes
	// This uses the operating system's random number generator
	// (much more secure than math/rand!)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}

	// Encode to Base64 for human-readable output
	// base64.StdEncoding uses standard alphabet: A-Za-z0-9+/
	return base64.StdEncoding.EncodeToString(key), nil
}

// DecodeKey converts Base64 key string back to bytes
//
// Input:  "SGVsbG8gV29ybGQh" (Base64 string)
// Output: [72, 101, 108, 108, 111, ...] (bytes)
func DecodeKey(keyBase64 string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, errors.New("invalid encryption key format")
	}

	// AES-256 requires exactly 32 bytes
	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes (256 bits)")
	}

	return key, nil
}

// ============================================================
// ENCRYPTION
// ============================================================

// Encrypt encrypts a message using AES-256-GCM
//
// Parameters:
//   - plaintext: the message to encrypt (string)
//   - keyBase64: the 256-bit key in Base64 format
//
// Returns: encrypted message in Base64 format
//
// How AES-GCM works (simplified):
//  1. Create AES cipher from the key
//  2. Generate random 12-byte nonce (number used once)
//  3. Encrypt plaintext + authenticate with GCM
//  4. Prepend nonce to ciphertext (we need it for decryption)
//  5. Return as Base64
//
// Structure of output:
//
//	[nonce (12 bytes)][ciphertext][auth tag (16 bytes)]
//	All encoded as Base64
func Encrypt(plaintext string, keyBase64 string) (string, error) {
	// Step 1: Decode the Base64 key to bytes
	key, err := DecodeKey(keyBase64)
	if err != nil {
		return "", err
	}

	// Step 2: Create AES cipher block
	// aes.NewCipher creates a cipher from a key
	// The key length determines AES variant:
	//   16 bytes = AES-128
	//   24 bytes = AES-192
	//   32 bytes = AES-256 (our choice!)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Step 3: Create GCM mode wrapper
	// GCM = Galois/Counter Mode
	// It provides:
	//   - Encryption (confidentiality)
	//   - Authentication (integrity) — detects if data was modified
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Step 4: Generate random nonce
	// Nonce = "Number used ONCE" — prevents replay attacks
	// Each message MUST have a unique nonce!
	// GCM standard nonce size = 12 bytes
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Step 5: Encrypt the message
	// gcm.Seal(dst, nonce, plaintext, additionalData)
	//   - dst: where to append ciphertext (we use nonce as prefix)
	//   - nonce: our random nonce
	//   - plaintext: data to encrypt
	//   - additionalData: extra authenticated data (nil for us)
	//
	// Result: [nonce][ciphertext][tag]
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Step 6: Encode to Base64 for safe transmission
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// ============================================================
// DECRYPTION
// ============================================================

// Decrypt decrypts a message using AES-256-GCM
//
// Parameters:
//   - ciphertextBase64: encrypted message in Base64 format
//   - keyBase64: the 256-bit key in Base64 format
//
// Returns: original plaintext message
//
// How it works:
//  1. Decode Base64 → bytes
//  2. Extract nonce from the beginning (first 12 bytes)
//  3. Extract actual ciphertext (rest of data)
//  4. Decrypt and verify authentication tag
func Decrypt(ciphertextBase64 string, keyBase64 string) (string, error) {
	// Step 1: Decode the Base64 key
	key, err := DecodeKey(keyBase64)
	if err != nil {
		return "", err
	}

	// Step 2: Decode the Base64 ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", errors.New("invalid encrypted message format")
	}

	// Step 3: Create AES cipher block (same as encryption)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Step 4: Create GCM wrapper (same as encryption)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Step 5: Validate ciphertext length
	// We need at least: nonce (12) + tag (16) = 28 bytes minimum
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Step 6: Split nonce and actual ciphertext
	// Remember: ciphertext = [nonce][encrypted data][tag]
	nonce := ciphertext[:nonceSize]
	encryptedData := ciphertext[nonceSize:]

	// Step 7: Decrypt and verify
	// gcm.Open(dst, nonce, ciphertext, additionalData)
	// Returns error if:
	//   - Wrong key
	//   - Data was tampered with
	//   - Authentication tag doesn't match
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", errors.New("decryption failed (wrong key or corrupted data)")
	}

	return string(plaintext), nil
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

// IsValidKey checks if a string is a valid encryption key
//
// Valid key requirements:
//   - Must be valid Base64
//   - Must decode to exactly 32 bytes
func IsValidKey(keyBase64 string) bool {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return false
	}
	return len(key) == 32
}
