# git.go Documentation

## Package utils

This module provides git repository operations for pushing encrypted files to GitHub, maintaining version control while preserving existing remote data.

### Imports

- `fmt`: String formatting and printing
- `time`: Time package for commit timestamps
- `github.com/go-git/go-billy/v5/memfs`: In-memory filesystem
- `github.com/go-git/go-git/v5`: Git operations library
- `github.com/go-git/go-git/v5/config`: Git configuration
- `github.com/go-git/go-git/v5/plumbing`: Git plumbing operations
- `github.com/go-git/go-git/v5/plumbing/object`: Git object types
- `github.com/go-git/go-git/v5/plumbing/transport/ssh`: SSH transport for git
- `github.com/go-git/go-git/v5/storage/memory`: In-memory git storage
- `golang.org/x/crypto/ssh`: SSH cryptography utilities

### Functions

#### PushFiles

```go
func PushFiles(repoURL string, rawPrivateKey []byte, files map[string][]byte, commitMsg string) error
```

Performs stateless append/update operations to a git repository while preserving existing remote files. This function handles incremental updates without downloading the entire repository history.

**Parameters:**
- `repoURL`: The git repository URL (e.g., "git@github.com:username/.nexus.git")
- `rawPrivateKey`: The raw SSH private key bytes for authentication
- `files`: A map of file paths to their content to push
  - Key: File path in the repository
  - Value: File content as bytes
- `commitMsg`: The commit message to use

**Return:**
- `error`: Returns an error if clone, file write, commit, or push operations fail

**Process:**

1. **SSH Authentication**: Sets up SSH authentication using the provided private key
2. **Clone Repository**: Clones the repository with shallow depth (Depth=1) to minimize bandwidth
3. **Create/Update Files**: Writes files to the in-memory filesystem
   - New files are created
   - Existing files are overwritten with new content
4. **Stage Files**: Stages all modified files for commit
5. **Check for Changes**: Verifies that changes exist before committing
6. **Commit**: Creates a commit with the specified message and Zephyrus author information
7. **Push**: Pushes the commit to the remote master branch using SSH

**Error Handling:**
- Returns error if cloning fails
- Returns error if file creation fails
- Returns error if staging fails
- Returns error if commit fails
- Returns error if push fails
- Returns nil if no changes were made (clean status)

**Notes:**
- Uses shallow clone (Depth=1) for performance
- Ignores SSH host key verification (insecure but necessary for automation)
- Commits are attributed to "Zephyrus" user
- File paths can include subdirectories (e.g., ".config/index")
- Existing files on remote are not downloaded; only new/modified files are uploaded

**Example Usage:**

```go
files := map[string][]byte{
    "storage_id_1": encryptedFileData1,
    ".config/index": indexData,
}

err := PushFiles(
    "git@github.com:myusername/.nexus.git",
    privateKeyBytes,
    files,
    "Nexus: Uploaded new files",
)
if err != nil {
    log.Fatal(err)
}
```
