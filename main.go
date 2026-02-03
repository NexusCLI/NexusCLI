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
		Use:   "nexus",
		Short: "Nexus CLI: A stateless, encrypted git-based vault",
	}

	// --- SETUP ---
	// Supports: nexus setup [user] [key] OR nexus setup -u [user] -k [key]
	var setupCmd = &cobra.Command{
		Use:   "setup [username] [key-path]",
		Short: "Initialize the vault and encrypt your master key",
		Args:  cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// Override flags if positional arguments are provided
			if len(args) > 0 {
				username = args[0]
			}
			if len(args) > 1 {
				keyPath = args[1]
			}

			pass, _ := utils.GetPassword("Create a Vault Password: ")
			err := utils.SetupVault(username, keyPath, pass)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("✔ Setup complete.")
		},
	}
	setupCmd.Flags().StringVarP(&username, "user", "u", "", "GitHub username")
	setupCmd.Flags().StringVarP(&keyPath, "key", "k", "", "Path to local private key")

	// --- CONNECT ---
	// Supports: nexus connect [username]
	var connectCmd = &cobra.Command{
		Use:   "connect [username]",
		Short: "Login and sync the remote index to your local session",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			targetUser := ""
			if len(args) > 0 {
				targetUser = args[0]
			} else {
				fmt.Print("Enter GitHub Username: ")
				fmt.Scanln(&targetUser)
			}

			pass, _ := utils.GetPassword("Enter Vault Password: ")
			err := utils.Connect(targetUser, pass)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	// --- DISCONNECT ---
	var disconnectCmd = &cobra.Command{
		Use:   "disconnect",
		Short: "Clear the local session",
		Run: func(cmd *cobra.Command, args []string) {
			utils.Disconnect()
		},
	}

	// --- UPLOAD ---
	var uploadCmd = &cobra.Command{
		Use:   "upload [local-path] [vault-path]",
		Short: "Upload a file to the vault",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := utils.GetSession()
			if err != nil {
				log.Fatal(err)
			}
			err = utils.UploadFile(args[0], args[1], session)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("✔ Upload successful.")
		},
	}

	// --- DOWNLOAD ---
	var downloadCmd = &cobra.Command{
		Use:   "download [vault-path] [local-output-path]",
		Short: "Download a file from the vault",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := utils.GetSession()
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
		Use:   "delete [vault-path]",
		Short: "Remove a file from the vault",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := utils.GetSession()
			if err != nil {
				log.Fatal(err)
			}
			err = utils.DeleteFile(args[0], session)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("✔ File deleted.")
		},
	}

	// --- LIST ---
	var listCmd = &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List all files in the vault",
		Run: func(cmd *cobra.Command, args []string) {
			session, err := utils.GetSession()
			if err != nil {
				log.Fatal(err)
			}
			err = utils.ListFiles(session)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	// --- SEARCH ---
	var searchCmd = &cobra.Command{
		Use:   "search [query]",
		Short: "Search for files in the vault index",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := utils.GetSession()
			if err != nil {
				log.Fatal(err)
			}
			err = utils.SearchFiles(session, args[0])
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	// --- PURGE ---
	var purgeCmd = &cobra.Command{
		Use:   "purge",
		Short: "Hard-reset the remote repository",
		Run: func(cmd *cobra.Command, args []string) {
			session, err := utils.GetSession()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Print("⚠️  Confirm PURGE? (y/N): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" {
				return
			}
			err = utils.PurgeVault(session)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("✔ Vault purged.")
		},
	}

	rootCmd.AddCommand(setupCmd, connectCmd, disconnectCmd, uploadCmd, downloadCmd, deleteCmd, listCmd, searchCmd, purgeCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
