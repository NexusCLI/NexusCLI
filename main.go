package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"zep/utils"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	username    string
	keyPath     string
	historyFile = filepath.Join(os.TempDir(), ".zephyrus_history")
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "zep",
		Short: "Zephyrus CLI - Secure Vault on GitHub",
	}

	// Persistent flag allows -u to be used across all subcommands
	rootCmd.PersistentFlags().StringVarP(&username, "user", "u", "", "GitHub username (forces stateless mode if no session exists)")

	// --- SESSION HELPER ---
	// This logic prioritizes the local zephyrus.conf, but falls back to
	// manual auth if -u is provided or if the user is not connected.
	getEffectiveSession := func() (*utils.Session, error) {
		// 1. Check for active local session
		sess, err := utils.GetSession()
		if err == nil {
			return sess, nil
		}

		// 2. Stateless Fallback: If not connected, prompt for info
		if username == "" {
			fmt.Print("No active session. Enter GitHub Username: ")
			fmt.Scanln(&username)
		}

		pass, err := utils.GetPassword("Enter Vault Password: ")
		if err != nil {
			return nil, err
		}

		fmt.Println("Authenticating and fetching index (Stateless Mode)...")
		return utils.FetchSessionStateless(username, pass)
	}

	// --- SETUP ---
	var setupCmd = &cobra.Command{
		Use:   "setup [username] [key-path]",
		Short: "Initialize vault and encrypt master key",
		Args:  cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				username = args[0]
			}
			if len(args) > 1 {
				keyPath = args[1]
			}

			// Interactive guide if no arguments provided
			if len(args) == 0 {
				fmt.Println("\n=== Zephyrus Vault Setup Guide ===")
				fmt.Println("Before we begin, please ensure you have completed the following steps:")
				fmt.Println("1. ✓ Created a GitHub account (https://github.com)")
				fmt.Println("2. ✓ Created an EMPTY repository named `.zephyrus` in your GitHub account")
				fmt.Println("3. ✓ Generated an SSH key pair (run: ssh-keygen -t ed25519)")
				fmt.Println("4. ✓ Added your PUBLIC key as a Deploy Key to your `.zephyrus` repository")
				fmt.Println("   - Go to: https://github.com/YOUR_USERNAME/.zephyrus/settings/keys")
				fmt.Println("   - Click 'Add deploy key'")
				fmt.Println("   - Paste your PUBLIC key (id_ed25519.pub) content")
				fmt.Println("   - Enable 'Allow write access' ✓")
				fmt.Println("Do you have all of this ready? (y/n): ")
				var ready string
				fmt.Scanln(&ready)
				if ready != "y" && ready != "yes" {
					fmt.Println("\n❌ Setup cancelled. Please complete the prerequisites first.")
					fmt.Println("📖 For detailed instructions, visit: https://github.com/zephyrus-development/zephyrus-cli#setup-your-vault")
					return
				}

				fmt.Println("\n--- Step 1: GitHub Username ---")
				fmt.Print("Enter your GitHub username: ")
				fmt.Scanln(&username)
				if username == "" {
					fmt.Println("❌ Username cannot be empty.")
					return
				}

				fmt.Println("\n--- Step 2: SSH Private Key Path ---")
				fmt.Print("Enter the path to your SSH PRIVATE key (e.g., ~/.ssh/id_ed25519): ")
				reader := bufio.NewReader(os.Stdin)
				keyPathInput, _ := reader.ReadString('\n')
				keyPath = strings.TrimSpace(keyPathInput)
				if keyPath == "" {
					fmt.Println("❌ Key path cannot be empty.")
					return
				}

				// Expand ~ to home directory
				if strings.HasPrefix(keyPath, "~") {
					home, err := os.UserHomeDir()
					if err == nil {
						keyPath = strings.Replace(keyPath, "~", home, 1)
					}
				}

				// Verify key file exists
				if _, err := os.Stat(keyPath); err != nil {
					fmt.Printf("❌ SSH key file not found at: %s\n", keyPath)
					return
				}

				fmt.Println("\n--- Step 3: Vault Password ---")
				fmt.Println("Create a strong password to encrypt your SSH key.")
				fmt.Println("⚠️  IMPORTANT: This password cannot be recovered. Please remember it!")
				pass, _ := utils.GetPassword("Create Vault Password: ")
				if pass == "" {
					fmt.Println("❌ Password cannot be empty.")
					return
				}

				passConfirm, _ := utils.GetPassword("Confirm Vault Password: ")
				if pass != passConfirm {
					fmt.Println("❌ Passwords do not match.")
					return
				}

				fmt.Println("\n--- Initializing Vault ---")
				fmt.Printf("Setting up vault for user: %s\n", username)
				err := utils.SetupVault(username, keyPath, pass)
				if err != nil {
					fmt.Printf("❌ Setup failed: %v\n", err)
					fmt.Println("\n📖 Troubleshooting:")
					fmt.Println("- Ensure .zephyrus repository exists at https://github.com/" + username + "/.zephyrus")
					fmt.Println("- Verify your SSH key has been added as a Deploy Key with write access")
					fmt.Println("- Check that your SSH key has permissions (chmod 600 on Unix-like systems)")
					return
				}

				fmt.Println("\n✔ Setup complete!")
				fmt.Println("\n--- Next Steps ---")
				fmt.Println("1. Run 'zep connect' to create a local session")
				fmt.Println("2. Run 'zep upload <file> <vault-path>' to upload your first file")
				fmt.Println("3. Run 'zep help' to see all available commands")
				return
			}

			// Non-interactive mode (arguments provided)
			if username == "" || keyPath == "" {
				fmt.Println("Error: Username and Key Path are required.")
				return
			}
			pass, _ := utils.GetPassword("Create Vault Password: ")
			err := utils.SetupVault(username, keyPath, pass)
			if err != nil {
				fmt.Printf("❌ Setup failed: %v\n", err)
				return
			}
			fmt.Println("✔ Setup complete.")
		},
	}

	// --- CONNECT ---
	var connectCmd = &cobra.Command{
		Use:     "connect [username]",
		Aliases: []string{"conn", "login", "auth", "signin", "con", "cn"},
		Short:   "Login and create a local session (caching the index)",
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := username
			if len(args) > 0 {
				target = args[0]
			}
			if target == "" {
				fmt.Print("Enter Username: ")
				fmt.Scanln(&target)
			}
			pass, _ := utils.GetPassword("Enter Password: ")
			err := utils.Connect(target, pass)
			if err != nil {
				fmt.Printf("❌ Connection failed: %v\n", err)
				return
			}
			fmt.Println("✔ Connected.")
		},
	}

	// --- UPLOAD ---
	var uploadCmd = &cobra.Command{
		Use:     "upload [local-path] [vault-path]",
		Aliases: []string{"up", "u", "add"}, // Multiple aliases allowed
		Short:   "Upload a file to the vault",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			localPath := args[0]
			vaultPath := localPath
			if len(args) > 1 {
				vaultPath = args[1]
			} else {
				// Use basename of local path if vault path not provided
				vaultPath = filepath.Base(localPath)
			}

			// 1. Check if the config file exists BEFORE starting
			_, err := os.Stat("zephyrus.conf")
			isPersistent := err == nil

			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			err = utils.UploadFile(localPath, vaultPath, session)
			if err != nil {
				fmt.Printf("❌ Upload failed: %v\n", err)
				return
			}

			// 2. Only save the updated index if we were already in a persistent session
			if isPersistent {
				session.Save()
			}
			fmt.Println("✔ Upload successful.")
		},
	}

	// --- DOWNLOAD ---
	var sharedFlag string
	var downloadCmd = &cobra.Command{
		Use:     "download [vault-path] [local-path]",
		Aliases: []string{"down", "d", "get"},
		Short:   "Download a file from the vault",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			vaultPath := args[0]
			localPath := vaultPath
			if len(args) > 1 {
				localPath = args[1]
			} else {
				// Use basename of vault path if local path not provided
				localPath = filepath.Base(vaultPath)
			}

			// Check if downloading a shared file
			if sharedFlag != "" {
				err := utils.DownloadSharedFile(sharedFlag, localPath)
				if err != nil {
					fmt.Printf("❌ Shared file download failed: %v\n", err)
					return
				}
				fmt.Println("✔ Shared file download successful.")
				return
			}

			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			err = utils.DownloadFile(vaultPath, localPath, session)
			if err != nil {
				fmt.Printf("❌ Download failed: %v\n", err)
				return
			}
			fmt.Println("✔ Download successful.")
		},
	}
	downloadCmd.Flags().StringVar(&sharedFlag, "shared", "", "Download a shared file using share string (username:storage_id:key)")

	// --- DELETE ---
	var deleteCmd = &cobra.Command{
		Use:     "delete [vault-path]",
		Aliases: []string{"del", "rm", "remove"},
		Short:   "Delete a file or folder (recursive)",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_, err := os.Stat("zephyrus.conf")
			isPersistent := err == nil

			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			err = utils.DeletePath(args[0], session)
			if err != nil {
				fmt.Printf("❌ Delete failed: %v\n", err)
				return
			}

			if isPersistent {
				session.Save()
			}
			fmt.Println("✔ Item removed.")
		},
	}

	// --- LIST ---
	var listCmd = &cobra.Command{
		Use:   "ls [folder]",
		Short: "List vault contents",
		Run: func(cmd *cobra.Command, args []string) {
			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			path := ""
			if len(args) > 0 {
				path = args[0]
			}
			utils.ListFiles(session, path)
		},
	}

	// --- SEARCH ---
	var searchCmd = &cobra.Command{
		Use:     "search [query]",
		Aliases: []string{"s"},
		Short:   "Search the vault index",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}
			utils.SearchFiles(session, args[0])
		},
	}

	// --- PURGE ---
	var purgeCmd = &cobra.Command{
		Use:   "purge",
		Short: "Wipe all remote data",
		Run: func(cmd *cobra.Command, args []string) {
			// Check if we are persistent BEFORE running
			_, statErr := os.Stat("zephyrus.conf")
			isPersistent := statErr == nil

			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			fmt.Print("⚠️  Confirm PURGE? This wipes all remote data and history. (y/N): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" {
				return
			}

			err = utils.PurgeVault(session)
			if err != nil {
				fmt.Printf("❌ Purge failed: %v\n", err)
				return
			}

			// Only update the local session file if it existed
			if isPersistent {
				session.Save()
			}

			fmt.Println("✔ Remote vault has been wiped and local index cleared.")
		},
	}

	// --- DISCONNECT ---
	var disconnectCmd = &cobra.Command{
		Use:     "disconnect",
		Aliases: []string{"disc", "logout", "signout", "logoff", "exit", "dc"},
		Short:   "Remove local session cache",
		Run: func(cmd *cobra.Command, args []string) {
			utils.Disconnect()
			fmt.Println("✔ Logged out.")
		},
	}

	// --- SHARE ---
	var shareCmd = &cobra.Command{
		Use:   "share [vault-path]",
		Short: "Generate a share string for a file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Check if the config file exists BEFORE starting
			_, err := os.Stat("zephyrus.conf")
			isPersistent := err == nil

			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			// Prompt for share password
			sharePassword, _ := utils.GetPassword("Enter Share Password (recipients will use this to decrypt): ")
			if sharePassword == "" {
				fmt.Println("❌ Share password cannot be empty.")
				return
			}

			shareString, err := utils.ShareFile(args[0], sharePassword, session)
			if err != nil {
				fmt.Printf("❌ Share failed: %v\n", err)
				return
			}

			// Save the updated session if we were already in a persistent session
			if isPersistent {
				session.Save()
			}

			// Extract filename for display
			filename := filepath.Base(args[0])

			fmt.Println("\n✔ File shared successfully!")
			fmt.Printf("Filename: %s\n", filename)
			fmt.Println("\nShare this string with recipient:")
			fmt.Println(shareString)
			fmt.Println("\nWeb Share Link:")
			fmt.Printf("  https://zephyrus-development.github.io/shared/#%s\n", shareString)
			fmt.Println("\nRecipient can download with:")
			fmt.Printf("  zep download _ output.file --shared \"%s\"\n", shareString)

			fmt.Println("\nOr read with:")
			fmt.Printf("  zep read _ --shared \"%s\"\n", shareString)
		},
	}

	// --- READ ---
	var readSharedFlag string
	var readCmd = &cobra.Command{
		Use:     "read [vault-path]",
		Aliases: []string{"cat"},
		Short:   "Read and display file content (no download)",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Check if reading a shared file
			if readSharedFlag != "" {
				err := utils.ReadSharedFile(readSharedFlag)
				if err != nil {
					fmt.Printf("❌ Shared file read failed: %v\n", err)
					return
				}
				return
			}

			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			err = utils.ReadFile(args[0], session)
			if err != nil {
				fmt.Printf("❌ Read failed: %v\n", err)
				return
			}
		},
	}
	readCmd.Flags().StringVar(&readSharedFlag, "shared", "", "Read a shared file using share string (username:storage_id:key)")

	// --- SHARED MANAGEMENT ---
	var sharedCmd = &cobra.Command{
		Use:   "shared",
		Short: "Manage shared files",
	}

	var sharedLsCmd = &cobra.Command{
		Use:   "ls",
		Short: "List all shared files",
		Run: func(cmd *cobra.Command, args []string) {
			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			files := utils.ListSharedFiles(session)
			if len(files) == 0 {
				fmt.Println("No shared files.")
				return
			}

			fmt.Println("\n📤 SHARED FILES")
			fmt.Println("REFERENCE  FILE NAME              SHARED AT")
			fmt.Println("---------  ----------              ---------")
			for _, f := range files {
				fmt.Printf("%-9s %-24s %s\n", f.Reference, f.OriginalPath, f.SharedAt.Format("2006-01-02 15:04"))
			}
			fmt.Println()
		},
	}

	var sharedRmCmd = &cobra.Command{
		Use:     "rm [reference]",
		Aliases: []string{"revoke", "delete"},
		Short:   "Revoke/remove a shared file",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			reference := args[0]

			// Confirm revocation
			fmt.Printf("⚠️  Revoke shared file %s? (y/N): ", reference)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "yes" {
				fmt.Println("Cancelled.")
				return
			}

			err = utils.RevokeSharedFile(reference, session)
			if err != nil {
				fmt.Printf("❌ Revoke failed: %v\n", err)
				return
			}

			// Save updated session if persistent
			_, statErr := os.Stat("zephyrus.conf")
			if statErr == nil {
				session.Save()
			}

			fmt.Printf("✔ Shared file '%s' revoked.\n", reference)
		},
	}

	var sharedInfoCmd = &cobra.Command{
		Use:   "info [reference]",
		Short: "Show info about a shared file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			reference := args[0]
			entry, err := utils.GetSharedFileInfo(reference, session)
			if err != nil {
				fmt.Printf("❌ %v\n", err)
				return
			}

			// Encode filename to base64 for share string
			encodedFilename := base64.StdEncoding.EncodeToString([]byte(entry.Name))
			shareString := fmt.Sprintf("%s:%s:%s:%s", session.Username, entry.Reference, entry.Password, encodedFilename)

			fmt.Printf("\n📄 SHARED FILE INFO\n")
			fmt.Printf("Reference:     %s\n", entry.Reference)
			fmt.Printf("File Name:     %s\n", entry.OriginalPath)
			fmt.Printf("Shared At:     %s\n", entry.SharedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Password:      %s\n", entry.Password)
			fmt.Printf("\nShare String:  %s\n", shareString)
			fmt.Printf("\nWeb Share Link: https://zephyrus-development.github.io/shared/#%s\n\n", shareString)
		},
	}

	sharedCmd.AddCommand(sharedLsCmd, sharedRmCmd, sharedInfoCmd)

	// --- SETTINGS MANAGEMENT ---
	var settingsCmd = &cobra.Command{
		Use:   "settings",
		Short: "Manage vault settings",
	}

	var settingsInfoCmd = &cobra.Command{
		Use:   "info",
		Short: "Display current vault settings",
		Run: func(cmd *cobra.Command, args []string) {
			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			fmt.Println("\n⚙️  VAULT SETTINGS")
			fmt.Println("─────────────────────────────────────────")
			fmt.Printf("Commit Author Name (author-name):      	 %s\n", session.Settings.CommitAuthorName)
			fmt.Printf("Commit Author Email (author-email):     %s\n", session.Settings.CommitAuthorEmail)
			fmt.Printf("Commit Message (commit-message):        %s\n", session.Settings.CommitMessage)
			fmt.Printf("File Hash Length (file-hash-length):    %d characters\n", session.Settings.FileHashLength)
			fmt.Printf("Share Hash Length (share-hash-length):  %d characters\n", session.Settings.ShareHashLength)
			fmt.Println("─────────────────────────────────────────")
		},
	}

	var settingsSetCmd = &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Update a vault setting",
		Long:  "Update a setting. Keys: author-name, author-email, commit-message, file-hash-length, share-hash-length",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := getEffectiveSession()
			if err != nil {
				fmt.Printf("❌ Authentication failed: %v\n", err)
				return
			}

			key := args[0]
			value := args[1]

			switch key {
			case "author-name":
				session.Settings.CommitAuthorName = value
			case "author-email":
				session.Settings.CommitAuthorEmail = value
			case "commit-message":
				session.Settings.CommitMessage = value
			case "file-hash-length":
				var length int
				_, err := fmt.Sscanf(value, "%d", &length)
				if err != nil {
					fmt.Printf("❌ Invalid number: %v\n", err)
					return
				}
				session.Settings.FileHashLength = length
			case "share-hash-length":
				var length int
				_, err := fmt.Sscanf(value, "%d", &length)
				if err != nil {
					fmt.Printf("❌ Invalid number: %v\n", err)
					return
				}
				session.Settings.ShareHashLength = length
			default:
				fmt.Printf("❌ Unknown setting: %s\n", key)
				fmt.Println("Available keys: author-name, author-email, commit-message, file-hash-length, share-hash-length")
				return
			}

			// Validate the settings
			if err := session.Settings.Validate(); err != nil {
				fmt.Printf("❌ Invalid setting: %v\n", err)
				return
			}

			// Save settings to remote vault
			err = utils.SaveSettings(session.Username, session.Password, session.RawKey, session.Settings)
			if err != nil {
				fmt.Printf("❌ Failed to save settings: %v\n", err)
				return
			}

			// Save updated session if persistent
			_, statErr := os.Stat("zephyrus.conf")
			if statErr == nil {
				session.Save()
			}

			fmt.Printf("✔ Setting '%s' updated to '%v'\n", key, value)
		},
	}

	settingsCmd.AddCommand(settingsInfoCmd, settingsSetCmd)

	// --- SHELL ---
	var shellCmd = &cobra.Command{
		Use:     "shell [username]",
		Aliases: []string{"sh"},
		Short:   "Launch interactive REPL",
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				username = args[0]
			}
			runInteractiveShell(rootCmd)
		},
	}

	rootCmd.AddCommand(
		setupCmd, connectCmd, disconnectCmd,
		uploadCmd, downloadCmd, deleteCmd,
		listCmd, searchCmd, purgeCmd, shareCmd, readCmd, sharedCmd, settingsCmd,
		shellCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func resetTerminal() {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		// Get the current state
		state, err := term.GetState(fd)
		if err == nil {
			// Force the terminal to restore to a clean, "echo-on" state
			term.Restore(fd, state)
		}
	}
}

func runInteractiveShell(rootCmd *cobra.Command) {
	resetTerminal()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Zephyrus Interactive Shell ===")

	var cachedSession *utils.Session
	var err error

	// 1. Authentication Loop
	for {
		if username == "" {
			fmt.Print("Username: ")
			os.Stdout.Sync()
			un, _ := reader.ReadString('\n')
			username = strings.TrimSpace(un)
		}

		pass, getPassErr := utils.GetPassword("Password: ")
		if getPassErr != nil {
			fmt.Printf("Error reading password: %v\n", getPassErr)
			return
		}

		fmt.Println("Authenticating...")
		cachedSession, err = utils.FetchSessionStateless(username, pass)

		if err != nil {
			// Check if it's an auth failure or a network/not-found issue
			fmt.Printf("❌ Login failed: %v\n", err)

			// Reset username if you want them to be able to change it on retry
			// Otherwise, leave it so they only have to re-type the password
			fmt.Println("Please try again.")
			fmt.Println("-----------------------")
			continue
		}

		// If we reach here, login was successful
		break
	}

	// 2. Set the global cache and start the REPL
	utils.SetGlobalSession(cachedSession)
	fmt.Printf("✔ Welcome, %s. Session Active.\n", username)
	fmt.Println("Type 'help' for commands or 'exit' to quit.")

	for {
		fmt.Print("zep> ")
		os.Stdout.Sync()

		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" || input == "logout" || input == "disc" || input == "dc" || input == "signout" || input == "logoff" || input == "disconnect" {
			break
		}

		// Handle commands
		args := strings.Fields(input)
		rootCmd.SetArgs(args)

		// We capture the error here so a failed command doesn't kill the shell
		if cmdErr := rootCmd.Execute(); cmdErr != nil {
			fmt.Printf("❌ Error: %v\n", cmdErr)
		}
	}
}
