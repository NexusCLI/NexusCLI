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

// PurgeVault wipes the remote repository by forcing an empty commit history.
func PurgeVault(session *Session) error {
	repoURL := fmt.Sprintf("git@github.com:%s/.nexus.git", session.Username)

	// 1. Prepare an entirely new, empty Git environment in memory
	storer := memory.NewStorage()
	fs := memfs.New()

	publicKeys, err := ssh.NewPublicKeys("git", session.RawKey, "")
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}
	publicKeys.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()

	// 2. Initialize a fresh repo and create a "Wipe" commit
	r, _ := git.Init(storer, fs)
	w, _ := r.Worktree()

	commit, err := w.Commit("Nexus: Updated Vault", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Zephyrus",
			Email: "Auchrio@proton.me",
			When:  time.Now(),
		},
		AllowEmptyCommits: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create purge commit: %w", err)
	}

	// 3. Force push this empty state to GitHub to overwrite everything
	_, _ = r.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{repoURL}})

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       publicKeys,
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:refs/heads/master", commit))},
		Force:      true, // This is what actually wipes the remote history
	})
	if err != nil {
		return fmt.Errorf("failed to push purge: %w", err)
	}

	// 4. Update the session index in memory to be empty
	session.Index = NewIndex()

	return nil
}
