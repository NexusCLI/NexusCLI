package main

import (
	"bufio"
	"fmt"
	"log"
	"nexus-cli/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	username    string
	keyPath     string
	historyFile = filepath.Join(os.TempDir(), ".nexus_history")
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "Nexus-cli",
		Short: "Nexus CLI",
		// This bit ensures the root command doesn't just print help
		// if we want to trigger the REPL instead.
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				runInteractiveShell(cmd)
			}
		},
	}

	// Persistent flag allows -u to be used across all subcommands
	rootCmd.PersistentFlags().StringVarP(&username, "user", "u", "", "GitHub username (forces stateless mode if no session exists)")

	// --- SESSION HELPER ---
	// This logic prioritizes the local nexus.conf, but falls back to
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

			if username == "" || keyPath == "" {
				fmt.Println("Error: Username and Key Path are required.")
				return
			}
			pass, _ := utils.GetPassword("Create Vault Password: ")
			err := utils.SetupVault(username, keyPath, pass)
			if err != nil {
				log.Fatal(err)
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
				log.Fatal(err)
			}
			fmt.Println("✔ Connected.")
		},
	}

	// --- UPLOAD ---
	var uploadCmd = &cobra.Command{
		Use:     "upload [local-path] [vault-path]",
		Aliases: []string{"up", "u", "add"}, // Multiple aliases allowed
		Short:   "Upload a file to the vault",
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// 1. Check if the config file exists BEFORE starting
			_, err := os.Stat("nexus.conf")
			isPersistent := err == nil

			session, err := getEffectiveSession()
			if err != nil {
				log.Fatal(err)
			}

			err = utils.UploadFile(args[0], args[1], session)
			if err != nil {
				log.Fatal(err)
			}

			// 2. Only save the updated index if we were already in a persistent session
			if isPersistent {
				session.Save()
			}
			fmt.Println("✔ Upload successful.")
		},
	}

	// --- DOWNLOAD ---
	var downloadCmd = &cobra.Command{
		Use:     "download [vault-path] [local-path]",
		Aliases: []string{"down", "d", "get"},
		Short:   "Download a file from the vault",
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := getEffectiveSession()
			if err != nil {
				log.Fatal(err)
			}

			err = utils.DownloadFile(args[0], args[1], session)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("✔ Download successful.")
		},
	}

	// --- DELETE ---
	var deleteCmd = &cobra.Command{
		Use:     "delete [vault-path]",
		Aliases: []string{"del", "rm", "remove"},
		Short:   "Delete a file or folder (recursive)",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_, err := os.Stat("nexus.conf")
			isPersistent := err == nil

			session, err := getEffectiveSession()
			if err != nil {
				log.Fatal(err)
			}

			err = utils.DeletePath(args[0], session)
			if err != nil {
				log.Fatal(err)
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
				log.Fatal(err)
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
				log.Fatal(err)
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
			_, statErr := os.Stat("nexus.conf")
			isPersistent := statErr == nil

			session, err := getEffectiveSession()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Print("⚠️  Confirm PURGE? This wipes all remote data and history. (y/N): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" {
				return
			}

			err = utils.PurgeVault(session)
			if err != nil {
				log.Fatal(err)
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

	rootCmd.AddCommand(
		setupCmd, connectCmd, disconnectCmd,
		uploadCmd, downloadCmd, deleteCmd,
		listCmd, searchCmd, purgeCmd,
	)

	// Check if we should enter REPL or just execute once
	if len(os.Args) > 1 {
		if err := rootCmd.Execute(); err != nil {
			os.Exit(1)
		}
	} else {
		runInteractiveShell(rootCmd)
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

	fmt.Println("=== Nexus Interactive Shell ===")

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
		fmt.Print("nexus> ")
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
			// Cobra usually prints the error itself, but this is a safety net
		}
	}
}
