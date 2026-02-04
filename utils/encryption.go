package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	SaltSize   = 16
	Iterations = 100000
	KeySize    = 32 // AES-256
	NonceSize  = 12 // Standard for GCM
)

// Encrypt handles key derivation via PBKDF2 and encryption via AES-GCM
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, Iterations, KeySize, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Bundle as: [Salt][Nonce][Ciphertext]
	result := append(salt, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt extracts the salt, re-derives the key, and decrypts the data
func Decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < SaltSize+NonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// Extract metadata
	salt := data[:SaltSize]
	nonce := data[SaltSize : SaltSize+NonceSize]
	ciphertext := data[SaltSize+NonceSize:]

	// Re-derive key
	key := pbkdf2.Key([]byte(password), salt, Iterations, KeySize, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// GenerateRandomName creates a hex string with configurable length (default 16 chars = 8 bytes)
func GenerateRandomName() string {
	return GenerateRandomNameWithLength(8) // 8 bytes = 16 hex chars
}

// GenerateRandomNameWithLength creates a configurable-length hex string
func GenerateRandomNameWithLength(byteLength int) string {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

// GenerateFileKey creates a random 32-byte key for per-file encryption
func GenerateFileKey() []byte {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return key
}

// EncryptWithKey encrypts plaintext using a raw AES-256 key (no PBKDF2)
func EncryptWithKey(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Bundle as: [Nonce][Ciphertext]
	result := append(nonce, ciphertext...)
	return result, nil
}

// DecryptWithKey decrypts data using a raw AES-256 key (no PBKDF2)
func DecryptWithKey(data []byte, key []byte) ([]byte, error) {
	if len(data) < NonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce := data[:NonceSize]
	ciphertext := data[NonceSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// EncodeKey converts a raw key to hex string for sharing
func EncodeKey(key []byte) string {
	return hex.EncodeToString(key)
}

// DecodeKey converts a hex string back to raw key
func DecodeKey(hexKey string) ([]byte, error) {
	return hex.DecodeString(hexKey)
}

// DecryptHexToBytes decodes a hex-encoded encrypted string and decrypts it with password
// Returns the raw plaintext bytes
func DecryptHexToBytes(hexEncrypted string, password string) ([]byte, error) {
	encrypted, err := hex.DecodeString(hexEncrypted)
	if err != nil {
		return nil, err
	}
	return Decrypt(encrypted, password)
}
