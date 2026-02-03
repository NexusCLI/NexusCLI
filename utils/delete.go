package utils

import (
	"fmt"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config" // Added for RefSpec
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	cryptossh "golang.org/x/crypto/ssh"
)

func DeleteFile(vaultPath string, password string, rawKey []byte, username string, repoURL string) error {
	storer := memory.NewStorage()
	fs := memfs.New()

	publicKeys, err := ssh.NewPublicKeys("git", rawKey, "")
	if err != nil {
		return err
	}
	publicKeys.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()

	fmt.Println("Cloning vault for deletion...")
	r, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:           repoURL,
		Auth:          publicKeys,
		ReferenceName: plumbing.ReferenceName("refs/heads/master"),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return fmt.Errorf("failed to clone: %w", err)
	}

	w, _ := r.Worktree()

	rawIndex, err := FetchRaw(username, ".config/index")
	if err != nil {
		return fmt.Errorf("could not fetch index: %w", err)
	}

	index, err := FromBytes(rawIndex, password)
	if err != nil {
		return fmt.Errorf("failed to decrypt index: %w", err)
	}

	entry, exists := index[vaultPath]
	if !exists {
		return fmt.Errorf("file '%s' not found in index", vaultPath)
	}

	fmt.Printf("Removing storage file: %s\n", entry.RealName)
	_, err = w.Remove(entry.RealName)
	if err != nil {
		fmt.Printf("Note: storage file %s already missing from Git\n", entry.RealName)
	}

	delete(index, vaultPath)
	newIndexBytes, err := index.ToBytes(password)
	if err != nil {
		return err
	}

	idxFile, _ := fs.Create(".config/index")
	idxFile.Write(newIndexBytes)
	idxFile.Close()
	w.Add(".config/index")

	commit, err := w.Commit("Nexus: Deleted "+vaultPath, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Nexus CLI",
			Email: "nexus@cli.io",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	fmt.Println("Pushing changes...")
	return r.Push(&git.PushOptions{
		Auth: publicKeys,
		// RefSpec is part of the 'config' package
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("%s:refs/heads/master", commit)),
		},
	})
}
