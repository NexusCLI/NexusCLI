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
	repoURL := fmt.Sprintf("git@github.com:%s/.zephyrus.git", session.Username)

	// 1. Prepare an entirely new, empty Git environment in memory
	PrintProgressStep(1, 3, "Initializing purge...")
	storer := memory.NewStorage()
	fs := memfs.New()

	publicKeys, err := ssh.NewPublicKeys("git", session.RawKey, "")
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}
	publicKeys.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()
	PrintCompletionLine("Purge initialized")

	// 2. Initialize a fresh repo and create a "Wipe" commit
	PrintProgressStep(2, 3, "Creating purge commit...")
	r, _ := git.Init(storer, fs)
	w, _ := r.Worktree()

	commit, err := w.Commit(session.Settings.CommitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  session.Settings.CommitAuthorName,
			Email: session.Settings.CommitAuthorEmail,
			When:  time.Now(),
		},
		AllowEmptyCommits: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create purge commit: %w", err)
	}
	PrintCompletionLine("Purge commit created")

	// 3. Force push this empty state to GitHub to overwrite everything
	PrintProgressStep(3, 3, "Force pushing to GitHub (wiping remote vault)...")
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
	PrintCompletionLine("Vault purged successfully")

	// 4. Update the session index in memory to be empty
	session.Index = NewIndex()

	return nil
}
