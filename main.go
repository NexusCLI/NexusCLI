package main

import (
	"fmt"
	"log"
	"nexus-cli/utils"
	"os"

	"github.com/spf13/cobra"
)

var (
	password string
	username string
	keyPath  string
	repoURL  string
)

func main() {
	var rootCmd = &cobra.Command{Use: "nexus"}

	// --- SETUP COMMAND ---
	var setupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Initialize the vault and encrypt your master key",
		Run: func(cmd *cobra.Command, args []string) {
			err := utils.SetupVault(username, keyPath, "")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Setup complete.")
		},
	}
	setupCmd.Flags().StringVarP(&username, "user", "u", "", "GitHub username")
	setupCmd.Flags().StringVarP(&keyPath, "key", "k", "", "Path to local private key")

	// --- CONNECT COMMAND ---
	var connectCmd = &cobra.Command{
		Use:   "connect",
		Short: "Establish a local session (nexus.conf)",
		Run: func(cmd *cobra.Command, args []string) {
			if username == "" {
				fmt.Print("Enter GitHub Username: ")
				fmt.Scanln(&username)
			}
			fmt.Print("Enter Vault Password: ")
			fmt.Scanln(&password)

			err := utils.Connect(username, password)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	connectCmd.Flags().StringVarP(&username, "user", "u", "", "GitHub username")

	// --- DISCONNECT COMMAND ---
	var disconnectCmd = &cobra.Command{
		Use:   "disconnect",
		Short: "Clear the local session",
		Run: func(cmd *cobra.Command, args []string) {
			utils.Disconnect()
		},
	}

	// --- UPLOAD COMMAND ---
	var uploadCmd = &cobra.Command{
		Use:   "upload [local-path] [vault-path]",
		Short: "Upload or overwrite a file in the vault",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := utils.GetSession()
			if err != nil {
				log.Fatal(err)
			}

			rURL := fmt.Sprintf("git@github.com:%s/.nexus.git", session.Username)
			err = utils.UploadFile(args[0], args[1], session.Password, session.RawKey, session.Username, rURL)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Upload successful.")
		},
	}

	// --- DELETE COMMAND ---
	var deleteCmd = &cobra.Command{
		Use:   "delete [vault-path]",
		Short: "Surgically remove a file from the vault",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session, err := utils.GetSession()
			if err != nil {
				log.Fatal(err)
			}

			rURL := fmt.Sprintf("git@github.com:%s/.nexus.git", session.Username)
			err = utils.DeleteFile(args[0], session.Password, session.RawKey, session.Username, rURL)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Deletion successful.")
		},
	}

	// --- PURGE COMMAND ---
	var purgeCmd = &cobra.Command{
		Use:   "purge",
		Short: "Wipe the entire vault and history",
		Run: func(cmd *cobra.Command, args []string) {
			session, err := utils.GetSession()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Print("⚠️  ARE YOU SURE? This wipes everything. (y/N): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" {
				fmt.Println("Purge cancelled.")
				return
			}

			rURL := fmt.Sprintf("git@github.com:%s/.nexus.git", session.Username)
			err = utils.PurgeVault(session.RawKey, rURL)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Vault purged.")
		},
	}

	// --- LIST COMMAND ---
	var listCmd = &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list", "index"},
		Short:   "List all files currently stored in the vault",
		Run: func(cmd *cobra.Command, args []string) {
			session, err := utils.GetSession()
			if err != nil {
				log.Fatal(err)
			}

			err = utils.ListFiles(session.Username, session.Password)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	// Register commands
	rootCmd.AddCommand(setupCmd, connectCmd, disconnectCmd, uploadCmd, deleteCmd, purgeCmd, listCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
