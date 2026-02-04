package utils

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	cryptossh "golang.org/x/crypto/ssh"
)

func SetupVault(githubUser string, keyFilePath string, password string) error {
	reader := bufio.NewReader(os.Stdin)

	// 1. Resolve Username
	if githubUser == "" {
		fmt.Print("Enter GitHub Username: ")
		githubUser, _ = reader.ReadString('\n')
		githubUser = strings.TrimSpace(githubUser)
	}

	// 2. Verify Repo (Public Check)
	repoURL := fmt.Sprintf("git@github.com:%s/.zephyrus.git", githubUser)
	repoWebURL := fmt.Sprintf("https://github.com/%s/.zephyrus", githubUser)
	resp, err := http.Head(repoWebURL)
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("repository '.zephyrus' not found at %s. Please create it manually on GitHub first", repoWebURL)
	}

	// 3. Resolve Key Path
	if keyFilePath == "" {
		fmt.Print("Enter Path to GitHub Private Key (e.g., ~/.ssh/id_ed25519): ")
		keyFilePath, _ = reader.ReadString('\n')
		keyFilePath = strings.TrimSpace(keyFilePath)
	}

	// 4. Resolve Password (Always prompted if not provided, per your requirement)
	if password == "" {
		fmt.Print("Create a Vault Password (to encrypt your cloud key): ")
		fmt.Scanln(&password)
	}

	// 5. Encrypt and Push
	rawKey, err := os.ReadFile(keyFilePath)
	if err != nil {
		return fmt.Errorf("failed to read local key: %w", err)
	}

	encryptedKey, err := Encrypt(rawKey, password)
	if err != nil {
		return err
	}

	storer := memory.NewStorage()
	fs := memfs.New()
	r, _ := git.Init(storer, fs)
	w, _ := r.Worktree()

	fs.MkdirAll(".config", 0755)
	f, _ := fs.Create(".config/key")
	f.Write(encryptedKey)
	f.Close()
	w.Add(".config/key")

	// Fetch and add README from the application source repository
	resp, err = http.Get("https://raw.githubusercontent.com/zephyrus-development/zephyrus-cli/main/README.md")
	if err == nil {
		defer resp.Body.Close()
		readmeContent, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			f, _ := fs.Create("README.md")
			f.Write(readmeContent)
			f.Close()
			w.Add("README.md")
		}
	}

	commit, _ := w.Commit("Zephyrus: Updated Vault", &git.CommitOptions{
		Author: &object.Signature{Name: "Zephyrus", Email: "Auchrio@proton.me", When: time.Now()},
	})

	publicKeys, _ := ssh.NewPublicKeys("git", rawKey, "")
	publicKeys.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()

	_, err = r.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{repoURL}})
	if err != nil {
		return err
	}

	return r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       publicKeys,
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:refs/heads/master", commit))},
		Force:      true,
	})
}
