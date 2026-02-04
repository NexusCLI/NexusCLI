package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	cryptossh "golang.org/x/crypto/ssh"
)

// DeletePath handles both single file deletion and recursive folder deletion
func DeletePath(vaultPath string, session *Session) error {
	repoURL := fmt.Sprintf("git@github.com:%s/.zephyrus.git", session.Username)

	// 1. Navigate to the target
	PrintProgressStep(1, 4, "Locating path in vault...")
	parts := strings.Split(strings.Trim(vaultPath, "/"), "/")
	currentMap := session.Index
	var targetName = parts[len(parts)-1]

	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		entry, ok := currentMap[part]
		if !ok || entry.Type != "folder" {
			return fmt.Errorf("path component '%s' not found", part)
		}
		currentMap = entry.Contents
	}

	targetEntry, exists := currentMap[targetName]
	if !exists {
		return fmt.Errorf("path '%s' not found in vault", vaultPath)
	}
	PrintCompletionLine("Path located")

	// 2. Identify all storage IDs to be removed
	PrintProgressStep(2, 4, "Preparing deletion...")
	var idsToDelete []string
	if targetEntry.Type == "file" {
		idsToDelete = append(idsToDelete, targetEntry.RealName)
	} else {
		// Recursive helper to find all nested realNames
		var collectIDs func(Entry)
		collectIDs = func(e Entry) {
			if e.Type == "file" {
				idsToDelete = append(idsToDelete, e.RealName)
			} else {
				for _, subEntry := range e.Contents {
					collectIDs(subEntry)
				}
			}
		}
		collectIDs(targetEntry)
		fmt.Printf("Preparing to recursively delete folder '%s' (%d files)...\n", vaultPath, len(idsToDelete))
	}
	PrintCompletionLine("Deletion prepared")

	// 3. Git Setup
	PrintProgressStep(3, 4, "Updating vault index...")
	storer := memory.NewStorage()
	fs := memfs.New()
	publicKeys, _ := ssh.NewPublicKeys("git", session.RawKey, "")
	publicKeys.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()

	r, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:           repoURL,
		Auth:          publicKeys,
		ReferenceName: plumbing.ReferenceName("refs/heads/master"),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return err
	}
	w, _ := r.Worktree()

	// 4. Physically remove all collected files from Git
	for _, id := range idsToDelete {
		_, err := w.Remove(id)
		if err != nil {
			fmt.Printf("Skipping %s (already gone from remote)\n", id)
		}
	}

	// 5. Update the Index: Remove the target from its parent map
	delete(currentMap, targetName)

	// 6. Push updated index
	newIndexBytes, _ := session.Index.ToBytes(session.Password)
	idxFile, _ := fs.Create(".config/index")
	idxFile.Write(newIndexBytes)
	idxFile.Close()
	w.Add(".config/index")

	// 7. Commit the changes
	PrintProgressStep(4, 4, "Uploading to GitHub...")
	// Ensure you are passing *git.CommitOptions, not just the Signature
	commit, err := w.Commit(session.Settings.CommitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  session.Settings.CommitAuthorName,
			Email: session.Settings.CommitAuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit deletion: %w", err)
	}

	err = r.Push(&git.PushOptions{
		Auth:     publicKeys,
		RefSpecs: []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:refs/heads/master", commit))},
	})
	if err != nil {
		return err
	}
	PrintCompletionLine("Deletion completed")

	return nil
}
