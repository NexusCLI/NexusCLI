# encryption.go Documentation

## Package utils

This module provides cryptographic operations for the Zephyrus CLI, including AES-GCM encryption, PBKDF2 key derivation, and secure random name generation.

### Constants

- **SaltSize**: 16 bytes - Size of the salt used in key derivation
- **Iterations**: 100000 - Number of PBKDF2 iterations for key derivation
- **KeySize**: 32 bytes - AES-256 key size
- **NonceSize**: 12 bytes - Standard GCM nonce size

### Imports

- `crypto/aes`: Advanced Encryption Standard
- `crypto/cipher`: Cipher modes (GCM)
- `crypto/rand`: Cryptographically secure random number generation
- `crypto/sha256`: SHA-256 hashing
- `encoding/hex`: Hexadecimal encoding/decoding
- `errors`: Error handling
- `io`: I/O utilities
- `golang.org/x/crypto/pbkdf2`: PBKDF2 key derivation

### Functions

#### Encrypt

```go
func Encrypt(plaintext []byte, password string) ([]byte, error)
```

Encrypts plaintext using AES-256-GCM with PBKDF2 key derivation.

**Parameters:**
- `plaintext`: The data to encrypt
- `password`: The password used to derive the encryption key

**Return:**
- `[]byte`: Encrypted data formatted as `[Salt][Nonce][Ciphertext]`
- `error`: Returns an error if random generation fails

**Process:**

1. Generates a random salt (16 bytes)
2. Derives a 256-bit key from the password using PBKDF2 with 100,000 iterations
3. Creates an AES-256 cipher and GCM mode
4. Generates a random nonce (12 bytes)
5. Encrypts the plaintext using AES-GCM
6. Returns salt + nonce + ciphertext concatenated

**Security Features:**
- Uses PBKDF2 with 100,000 iterations to protect against brute force
- Uses AES-256 for strong encryption
- GCM mode provides authenticated encryption
- Random salt and nonce for each encryption

#### Decrypt

```go
func Decrypt(data []byte, password string) ([]byte, error)
```

Decrypts data encrypted by the `Encrypt` function using the same password.

**Parameters:**
- `data`: The encrypted data in format `[Salt][Nonce][Ciphertext]`
- `password`: The password used to derive the decryption key

**Return:**
- `[]byte`: The decrypted plaintext
- `error`: Returns an error if decryption fails or data is malformed

**Process:**

1. Extracts salt from the first 16 bytes
2. Extracts nonce from the next 12 bytes
3. Remaining bytes are the ciphertext
4. Re-derives the key using the same password and salt
5. Decrypts and verifies authenticity using AES-GCM

**Error Handling:**
- Returns error if data is shorter than salt + nonce size
- Returns error if key derivation fails
- Returns error if decryption/authentication fails

#### GenerateRandomName

```go
func GenerateRandomName() string
```

Creates a 16-character hexadecimal string from 8 random bytes.

**Return:**
- `string`: A 16-character hex string (e.g., "a3f2e1c9d4b6f8e2")

**Usage:**
- Used to generate obfuscated storage IDs for files in the vault
- Files are stored with these random names instead of their original names

**Notes:**
- Panics if random byte generation fails (should never happen on healthy systems)

### Example Usage

```go
// Encrypt data
plaintext := []byte("sensitive data")
password := "myVaultPassword"
encrypted, err := Encrypt(plaintext, password)
if err != nil {
    log.Fatal(err)
}

// Later, decrypt with the same password
decrypted, err := Decrypt(encrypted, password)
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(decrypted)) // "sensitive data"

// Generate a random storage name for a file
storageName := GenerateRandomName() // "a3f2e1c9d4b6f8e2"
```
