package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"
	"time"
)

// GenerateShareReference generates a base62 reference with default 6 characters
func GenerateShareReference() (string, error) {
	return GenerateShareReferenceWithLength(6)
}

// GenerateShareReferenceWithLength generates a base62 reference with configurable length
func GenerateShareReferenceWithLength(length int) (string, error) {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	ref := make([]byte, length)
	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		ref[i] = charset[idx.Int64()]
	}
	return string(ref), nil
}

// ShareFile generates a share string with a new 6-char reference
// The shared file stores only a pointer (storage ID + encrypted file key) instead of a copy
func ShareFile(vaultPath string, sharePassword string, session *Session) (string, error) {
	// 1. Find the file entry in the index
	entry, err := session.Index.FindEntry(vaultPath)
	if err != nil {
		return "", fmt.Errorf("could not find file in vault: %w", err)
	}

	// 2. Ensure it's a file, not a folder
	if entry.Type == "folder" {
		return "", fmt.Errorf("'%s' is a directory, you can only share individual files", vaultPath)
	}

	// 3. Generate a new reference with configurable length from settings
	ref, err := GenerateShareReferenceWithLength(session.Settings.ShareHashLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate share reference: %w", err)
	}

	// 4. Decrypt the file key with vault password to get raw 32-byte key
	fileKeyBytes, err := DecryptHexToBytes(entry.FileKey, session.Password)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt file key: %w", err)
	}

	// 5. Create a pointer file containing: storage ID and file key (will be encrypted with share password)
	pointerData := map[string]string{
		"storageID": entry.RealName,
		"fileKey":   fmt.Sprintf("%x", fileKeyBytes),
	}
	pointerJSON, err := json.Marshal(pointerData)
	if err != nil {
		return "", fmt.Errorf("failed to create share pointer: %w", err)
	}

	// 6. Encrypt the pointer with the share password
	pointerEncrypted, err := Encrypt(pointerJSON, sharePassword)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt share pointer: %w", err)
	}

	// 7. Upload pointer to /shared/{ref}
	sharedPath := fmt.Sprintf("shared/%s", ref)
	filesToPush := map[string][]byte{
		sharedPath: pointerEncrypted,
	}

	err = PushFilesWithAuthor(
		fmt.Sprintf("git@github.com:%s/.zephyrus.git", session.Username),
		session.RawKey,
		filesToPush,
		session.Settings.CommitMessage,
		session.Settings.CommitAuthorName,
		session.Settings.CommitAuthorEmail,
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload share pointer: %w", err)
	}

	// 8. Add entry to shared index
	if session.SharedIndex == nil {
		session.SharedIndex = NewSharedIndex()
	}
	indexEntry := SharedFileEntry{
		Name:         vaultPath,
		Reference:    ref,
		Password:     sharePassword,
		SharedAt:     time.Now(),
		OriginalPath: vaultPath,
	}
	session.SharedIndex.AddEntry(indexEntry)

	// 9. Upload the updated shared index
	indexJSON, err := session.SharedIndex.EncryptForRemote(session.Password)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt shared index: %w", err)
	}

	indexFilesToPush := map[string][]byte{
		"shared/.config/index": indexJSON,
	}

	err = PushFilesWithAuthor(
		fmt.Sprintf("git@github.com:%s/.zephyrus.git", session.Username),
		session.RawKey,
		indexFilesToPush,
		session.Settings.CommitMessage,
		session.Settings.CommitAuthorName,
		session.Settings.CommitAuthorEmail,
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload shared index: %w", err)
	}

	// 10. Generate the share string: username:reference:sharepassword:base64filename
	filename := filepath.Base(vaultPath)
	encodedFilename := base64.StdEncoding.EncodeToString([]byte(filename))
	shareString := fmt.Sprintf("%s:%s:%s:%s", session.Username, ref, sharePassword, encodedFilename)

	return shareString, nil
}
