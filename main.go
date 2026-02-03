package main

import (
	"fmt"
	"log"
	"nexus-cli/utils"
	"os"

	"github.com/spf13/cobra"
)

var (
	username string
	keyPath  string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "Nexus-cli",
		Short: "Nexus CLI: A stateless, encrypted git-based vault",
		Long: `Nexus is a secure, git-backed vault that uses client-side encryption.
It supports both a persistent session (connect) or a stateless mode (-u flag).
Use 'nexus-cli help [command]' to view alias options for each command.`,
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

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
