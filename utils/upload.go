package utils

import (
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

func UploadFile(sourcePath string, vaultPath string, session *Session) error {
	repoURL := fmt.Sprintf("git@github.com:%s/.zephyrus.git", session.Username)

	// 1. Read source
	PrintProgressStep(1, 5, "Reading file...")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	PrintCompletionLine("File read successfully")

	// 2. Determine Storage Name and File Key
	PrintProgressStep(2, 5, "Validating vault...")
	var realName string
	var fileKey []byte

	entry, err := session.Index.FindEntry(vaultPath)
	if err == nil && entry.Type == "file" {
		// Existing file: reuse storage name, decrypt existing key
		realName = entry.RealName
		fmt.Printf("Updating existing file: %s (%s)\n", vaultPath, realName)

		// Decrypt the existing file key
		encryptedKey, _ := hex.DecodeString(entry.FileKey)
		fileKey, err = Decrypt(encryptedKey, session.Password)
		if err != nil {
			return fmt.Errorf("failed to decrypt file key: %w", err)
		}
	} else {
		// New file: generate new storage name and file key
		// Use configurable hash length from settings
		hashByteLength := session.Settings.FileHashLength / 2 // Convert hex chars to bytes
		realName = GenerateRandomNameWithLength(hashByteLength)
		fileKey = GenerateFileKey()

		// Encrypt the file key with the vault password
		encryptedKey, err := Encrypt(fileKey, session.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt file key: %w", err)
		}
		encryptedKeyHex := hex.EncodeToString(encryptedKey)

		session.Index.AddFile(vaultPath, realName, encryptedKeyHex)
		fmt.Printf("Uploading new file: %s as %s\n", vaultPath, realName)
	}
	PrintCompletionLine("File validated")

	// 3. Encrypt file data with the per-file key
	PrintProgressStep(3, 5, "Encrypting file...")
	time.Sleep(time.Millisecond * 100) // Simulate work for visibility
	encryptedData, err := EncryptWithKey(data, fileKey)
	if err != nil {
		return err
	}
	PrintCompletionLine("File encrypted")

	// 4. Encrypt updated index
	PrintProgressStep(4, 5, "Updating vault index...")
	indexBytes, err := session.Index.ToBytes(session.Password)
	if err != nil {
		return err
	}
	PrintCompletionLine("Vault index updated")

	// 5. Push to Git
	PrintProgressStep(5, 5, "Uploading to GitHub...")
	filesToPush := map[string][]byte{
		realName:        encryptedData,
		".config/index": indexBytes,
	}

	err = PushFilesWithAuthor(repoURL, session.RawKey, filesToPush, session.Settings.CommitMessage, session.Settings.CommitAuthorName, session.Settings.CommitAuthorEmail)
	if err != nil {
		return err
	}
	PrintCompletionLine("Upload to GitHub completed")

	// 6. Save updated index to local session to bypass cache
	return nil
}
