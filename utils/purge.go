package utils

import (
	"fmt"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	cryptossh "golang.org/x/crypto/ssh"
)

// PurgeVault wipes the entire repository history and leaves it completely empty.
func PurgeVault(rawKey []byte, repoURL string) error {
	storer := memory.NewStorage()
	fs := memfs.New()

	// 1. Setup Auth
	publicKeys, err := ssh.NewPublicKeys("git", rawKey, "")
	if err != nil {
		return err
	}
	publicKeys.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()

	// 2. Initialize a brand new repository in memory
	r, err := git.Init(storer, fs)
	if err != nil {
		return err
	}

	w, _ := r.Worktree()

	// 3. Create a 100% empty commit
	// AllowEmptyCommits: true is the key here to avoid needing a README file.
	commit, err := w.Commit("Nexus: PURGE VAULT (Total Wipe)", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Nexus CLI",
			Email: "nexus@cli.io",
			When:  time.Now(),
		},
		AllowEmptyCommits: true,
	})
	if err != nil {
		return err
	}

	// 4. Force push to remote master
	fmt.Println("⚠️  Wiping remote repository and all files...")

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})
	if err != nil {
		return err
	}

	return r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       publicKeys,
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("%s:refs/heads/master", commit)),
		},
		Force: true,
	})
}
