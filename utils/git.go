package utils

import (
	"fmt"
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

// PushFiles performs a stateless append/update to the repository.
// It preserves existing files on GitHub without downloading them.
func PushFiles(repoURL string, rawPrivateKey []byte, files map[string][]byte, commitMsg string) error {
	publicKeys, _ := ssh.NewPublicKeys("git", rawPrivateKey, "")
	publicKeys.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()

	storer := memory.NewStorage()
	fs := memfs.New()

	// 1. Clone the repo to get the current state
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

	// 2. Write files using the names provided in the 'files' map
	for path, content := range files {
		// If 'path' is the same 16-char string as before,
		// this Create call will overwrite the virtual file in memfs.
		f, err := fs.Create(path)
		if err != nil {
			return err
		}
		f.Write(content)
		f.Close()

		// 3. Re-stage the path
		// Git sees the same filename but new content.
		// It calculates a new content hash (SHA-1) but keeps the file identity.
		_, err = w.Add(path)
		if err != nil {
			return err
		}
	}

	// 4. Commit and Push
	status, _ := w.Status()
	if status.IsClean() {
		return nil // No changes to push
	}

	commit, _ := w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{Name: "Zephyrus", Email: "Auchrio@proton.me", When: time.Now()},
	})

	return r.Push(&git.PushOptions{
		Auth: publicKeys,
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("%s:refs/heads/master", commit)),
		},
	})
}
