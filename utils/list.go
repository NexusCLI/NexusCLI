package utils

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// ListFiles fetches the index and prints all files currently stored in the vault
func ListFiles(username string, password string) error {
	fmt.Printf("Listing files for vault: %s\n", username)

	// 1. Fetch the encrypted index from GitHub
	rawIndex, err := FetchRaw(username, ".config/index")
	if err != nil {
		// GRACEFUL ERROR HANDLING:
		// Check if the error is a 404 (file not found)
		if strings.Contains(err.Error(), "404") {
			fmt.Println("--------------------------------------------------")
			fmt.Println("ℹ Your vault is currently empty (no index found).")
			fmt.Println("Use 'nexus upload' to add your first file.")
			fmt.Println("--------------------------------------------------")
			return nil
		}
		return fmt.Errorf("could not fetch index: %w", err)
	}

	// 2. Decrypt the index
	index, err := FromBytes(rawIndex, password)
	if err != nil {
		return fmt.Errorf("failed to decrypt index (check your password): %w", err)
	}

	if len(index) == 0 {
		fmt.Println("Vault is currently empty.")
		return nil
	}

	// 3. Print in a clean, tab-aligned format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "VAULT PATH\tSTORAGE ID (HEX)")
	fmt.Fprintln(w, "----------\t----------------")

	for vPath, entry := range index {
		fmt.Fprintf(w, "%s\t%s\n", vPath, entry.RealName)
	}

	return w.Flush()
}
