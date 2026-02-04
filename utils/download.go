package utils

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func DownloadFile(vaultPath string, outputPath string, session *Session) error {
	// 1. Use your custom FindEntry logic to navigate the nested maps
	entry, err := session.Index.FindEntry(vaultPath)
	if err != nil {
		return fmt.Errorf("could not find file in vault: %w", err)
	}

	// 2. Safety check: Ensure we aren't trying to "download" a folder
	if entry.Type == "folder" {
		return fmt.Errorf("'%s' is a directory, you can only download individual files", vaultPath)
	}

	fmt.Printf("Downloading %s (Storage ID: %s)...\n", vaultPath, entry.RealName)

	// 3. Fetch the encrypted hex-named file from GitHub
	encryptedData, err := FetchRaw(session.Username, entry.RealName)
	if err != nil {
		return fmt.Errorf("failed to fetch storage file from remote: %w", err)
	}

	// 4. Decrypt the file key from the index
	encryptedKey, err := hex.DecodeString(entry.FileKey)
	if err != nil {
		return fmt.Errorf("invalid file key in index: %w", err)
	}
	fileKey, err := Decrypt(encryptedKey, session.Password)
	if err != nil {
		return fmt.Errorf("failed to decrypt file key: check your password")
	}

	// 5. Decrypt the file data with the file key
	decryptedData, err := DecryptWithKey(encryptedData, fileKey)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// 6. Save to the local output path
	return os.WriteFile(outputPath, decryptedData, 0644)
}

// DownloadSharedFile downloads a file using a share string (username:reference:sharepassword:base64filename)
func DownloadSharedFile(shareString string, outputPath string) error {
	// 1. Parse the share string (supports both old 3-part and new 4-part formats)
	parts := strings.Split(shareString, ":")
	if len(parts) < 3 || len(parts) > 4 {
		return fmt.Errorf("invalid share string format, expected 'username:reference:sharepassword' or 'username:reference:sharepassword:base64filename'")
	}

	username := parts[0]
	reference := parts[1]
	sharePassword := parts[2]
	var filename string

	// Decode filename if provided in share string
	if len(parts) == 4 {
		decoded, err := base64.StdEncoding.DecodeString(parts[3])
		if err != nil {
			return fmt.Errorf("invalid filename encoding in share string: %w", err)
		}
		filename = string(decoded)
	}

	// Use provided outputPath, or construct from filename if available
	finalOutputPath := outputPath
	if filename != "" && outputPath == "" {
		finalOutputPath = filename
	}

	if filename != "" {
		fmt.Printf("Downloading '%s' from %s (Reference: %s)...\n", filename, username, reference)
	} else {
		fmt.Printf("Downloading shared file from %s (Reference: %s)...\n", username, reference)
	}

	// 2. Fetch the encrypted file from the /shared/ folder
	sharedPath := fmt.Sprintf("shared/%s", reference)
	encryptedData, err := FetchRaw(username, sharedPath)
	if err != nil {
		return fmt.Errorf("failed to fetch shared file from remote: %w", err)
	}

	// 3. Decrypt with the share password
	decryptedData, err := Decrypt(encryptedData, sharePassword)
	if err != nil {
		return fmt.Errorf("decryption failed: invalid share password")
	}

	// 4. Save to the local output path
	return os.WriteFile(finalOutputPath, decryptedData, 0644)
}
