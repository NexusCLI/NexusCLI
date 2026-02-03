package main

import (
	"fmt"
	"log"
	"os"

	"nexus-cli/utils"
)

const (
	username = "Auchrio"
	repoURL  = "git@github.com:Auchrio/.nexus.git"
	keyPath  = ".config/key"
)

func main() {
	// 1. Credentials
	fmt.Print("Enter Vault Password: ")
	var password string
	fmt.Scanln(&password)

	rawKey, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("Failed to read SSH key: %v", err)
	}

	// --- TEST STEP 1: PURGE ---
	fmt.Println("\n[TEST 1] Purging vault...")
	err = utils.PurgeVault(rawKey, repoURL)
	if err != nil {
		log.Fatalf("Purge failed: %v", err)
	}

	// --- TEST STEP 2: TRIPLE UPLOAD ---
	fmt.Println("\n[TEST 2] Uploading test1, test2, and test3...")
	testFiles := []string{"test1.txt", "test2.txt", "test3.txt"}
	for _, vPath := range testFiles {
		// We use "test.txt" as the local source for all three
		err = utils.UploadFile("test.txt", vPath, password, rawKey, username, repoURL)
		if err != nil {
			log.Fatalf("Upload of %s failed: %v", vPath, err)
		}
	}

	// --- TEST STEP 3: DELETE TEST1 ---
	fmt.Println("\n[TEST 3] Deleting test1.txt...")
	err = utils.DeleteFile("test1.txt", password, rawKey, username, repoURL)
	if err != nil {
		log.Fatalf("Delete failed: %v", err)
	}

	// --- TEST STEP 4: OVERWRITE TEST2 ---
	// Note: To truly test the overwrite, ensure 'test.txt' has different content
	// or create a temp file here.
	fmt.Println("\n[TEST 4] Overwriting test2.txt...")
	err = utils.UploadFile("test.txt", "test2.txt", password, rawKey, username, repoURL)
	if err != nil {
		log.Fatalf("Overwrite failed: %v", err)
	}

	fmt.Println("\n--------------------------------------------------")
	fmt.Println("✔ ALL TESTS COMPLETE")
	fmt.Println("Expected Result on GitHub:")
	fmt.Println("1. '.config/index' exists")
	fmt.Println("2. Two hex-named files exist (for test2 and test3)")
	fmt.Println("3. Hex file for test1 is GONE")
	fmt.Println("4. History shows multiple commits")
	fmt.Println("--------------------------------------------------")
}
