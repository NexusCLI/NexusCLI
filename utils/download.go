package utils

import (
	"fmt"
	"os"
)

func DownloadFile(vaultPath string, outputPath string, session *Session) error {
	// 1. Find the file in our local (cached) index
	entry, exists := session.Index[vaultPath]
	if !exists {
		return fmt.Errorf("file '%s' not found in vault index", vaultPath)
	}

	fmt.Printf("Downloading %s (Storage ID: %s)...\n", vaultPath, entry.RealName)

	// 2. Fetch the encrypted hex-named file from GitHub
	encryptedData, err := FetchRaw(session.Username, entry.RealName)
	if err != nil {
		return fmt.Errorf("failed to fetch storage file: %w", err)
	}

	// 3. Decrypt the data
	decryptedData, err := Decrypt(encryptedData, session.Password)
	if err != nil {
		return fmt.Errorf("decryption failed (wrong password?): %w", err)
	}

	// 4. Save to disk
	return os.WriteFile(outputPath, decryptedData, 0644)
}
