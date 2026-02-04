⚠️ **For detailed technical architecture, cryptographic design, and security analysis, please refer to the [Whitepaper](https://github.com/NexusCLI/NexusCLI/wiki/WHITEPAPER).**

---

# Nexus CLI

A secure, encrypted file vault backed by GitHub. Store your sensitive files with confidence using end-to-end encryption and git-based version control.

## What is Nexus CLI?

Nexus CLI is a command-line tool that transforms your GitHub repository into a private, encrypted file vault. Your files are encrypted locally before being uploaded, ensuring that even if your GitHub repository is compromised, your data remains secure.

**Key Features:**
- 🔐 **End-to-End Encryption**: Files are encrypted with AES-256-GCM before leaving your computer
- 🔑 **Per-File Encryption Keys**: Each file has its own unique encryption key for secure sharing
- ☁️ **GitHub-Backed Storage**: Uses a private GitHub repository as distributed storage
- 🗂️ **Hierarchical Organization**: Create folders and organize files with full path support
- 🔍 **Search Functionality**: Quickly find files by name or path
- 📝 **Version Control**: Full git history of all vault operations
- 💻 **Cross-Platform**: Works on Windows, macOS, and Linux
- 🎯 **Stateless Mode**: Use with `-u` flag to authenticate without persistent sessions
- 🔄 **Interactive Shell**: REPL mode for multiple commands without re-authentication
- 📤 **Secure File Sharing**: Share individual files without exposing your entire vault

## Prerequisites

Before you can use Nexus CLI, you'll need:

1. **A GitHub Account**
2. **A Private Key for GitHub**: SSH key for authentication (e.g., `id_ed25519`)
3. **An Empty GitHub Repository**: Create a repository named `.nexus` in your GitHub account
4. **Nexus CLI Binary**: Download from [releases](https://github.com/NexusCLI/NexusCLI/releases)

## Installation

### Download Binary

1. Visit the [Nexus CLI Releases Page](https://github.com/NexusCLI/NexusCLI/releases)
2. Download the latest release for your operating system:
   - **Windows**: `nexus-cli.exe`
   - **macOS**: Please download and compile from source.
   - **Linux**: Please download and compile from source, linux release coming soon.
3. Make the binary executable (macOS/Linux):
   ```bash
   chmod +x nexus-cli-*
   ```
4. Optionally, add to your PATH for system-wide access

### Setup Your Vault

Before you can use Nexus ClI, you must create a [deploy key](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/managing-deploy-keys#deploy-keys) on the `.nexus` repository you have created, to do this you must generate a ssh keypair through either the `ssh-keygen` command (most secure) or through a website such as [this one](https://www.wpoven.com/tools/create-ssh-key), then add the public key to the deploy key settings tab of your `.nexus` github repository.

Then to start using Nexus CLI, initialize your vault:

```bash
./nexus-cli setup <github-username> <path-to-ssh--private-key>
```

**Example:**
```bash
./nexus-cli setup Auchrio ./private_key.txt
```

When prompted:
- Enter your GitHub username (if not provided as argument)
- Provide the path to your SSH private key
- Create a vault password (you'll use this to encrypt your GitHub SSH key)

**What happens during setup:**
1. Verifies your `.nexus` repository exists on GitHub
2. Reads your SSH private key from disk
3. Encrypts the key with your vault password
4. Pushes the encrypted key to `.config/key` in your vault repository

⚠️ **Important**: Your vault password is never stored. You must remember it becasue nobody can recover it for you.

## Getting Started

### Interactive Mode (REPL)

Start the interactive shell:
```bash
./nexus-cli
```

You'll be prompted to enter your username and password. Once authenticated, you can run commands without re-entering credentials:

```
=== Nexus Interactive Shell ===
Username: myusername
Password: ••••••••••
✔ Welcome, myusername. Session Active.
Type 'help' for commands or 'exit' to quit.

nexus> upload ./document.pdf documents/report.pdf
✔ Upload successful.

nexus> ls documents
NAME          TYPE    STORAGE ID
----          ----    ----------
report.pdf    [FILE]  a3f2e1c9d4b6f8e2

nexus> exit
```

### Command Line Mode

Run commands directly without entering the shell:

```bash
# Upload a file
./nexus-cli upload ./document.pdf documents/report.pdf

# List files
./nexus-cli ls documents

# Download a file
./nexus-cli download documents/report.pdf ./report.pdf

# Exit
```

### Stateless Mode

If you haven't run `connect`, you can use any command with the `-u` flag to authenticate on-the-fly:

```bash
./nexus-cli upload -u myusername ./document.pdf documents/report.pdf
```

You'll be prompted for your vault password. This doesn't create a persistent session.

## Command Reference

### `setup` - Initialize Your Vault

Sets up a new Nexus vault by encrypting and storing your GitHub SSH key.

**Usage:**
```bash
./nexus-cli setup [username] [key-path]
```

**Arguments:**
- `username` (optional): Your GitHub username
- `key-path` (optional): Path to your SSH private key (e.g., `~/.ssh/id_ed25519`)

**Interactive Prompts (if arguments not provided):**
- GitHub Username
- SSH Key Path
- Vault Password (to encrypt your key)

**Example:**
```bash
./nexus-cli setup myusername ~/.ssh/id_ed25519
# Or interactively:
./nexus-cli setup
```

**Prerequisites:**
- Repository named `.nexus` must exist on your GitHub account
- SSH key must exist at the specified path
- SSH key must have access to GitHub

**Note**: This command must be run once before using other vault operations.

---

### `connect` - Create a Persistent Session

Authenticates and caches your vault index locally in `nexus.conf`. Subsequent commands won't require re-authentication.

**Usage:**
```bash
./nexus-cli connect [username]
```

**Aliases:** `conn`, `login`, `auth`, `signin`, `con`, `cn`

**Arguments:**
- `username` (optional): Your GitHub username

**Interactive Prompts (if username not provided):**
- Username
- Vault Password

**Example:**
```bash
./nexus-cli connect myusername
# Password: ••••••••••
# ✔ Connected.

./nexus-cli upload ./file.pdf documents/file.pdf
# Uses cached session - no password prompt
```

**When to Use:**
- You're running multiple commands in sequence
- You want to avoid entering your password repeatedly
- You're working on a secure machine where storing `nexus.conf` is safe

**Note**: Creates a file called `nexus.conf` in the current directory containing your cached index.

---

### `upload` - Upload a File

Encrypts and uploads a file from your local filesystem to the vault.

**Usage:**
```bash
./nexus-cli upload <local-path> <vault-path>
```

**Aliases:** `up`, `u`, `add`

**Arguments:**
- `local-path` (required): Path to the file on your computer
- `vault-path` (required): Destination path in the vault (e.g., `documents/report.pdf`)

**Features:**
- Automatically creates intermediate folders if they don't exist
- Updates existing files (overwrites content while keeping storage ID)
- Encrypts file with AES-256-GCM before uploading
- Updates vault index on remote repository

**Examples:**
```bash
# Upload a single file
./nexus-cli upload ./document.pdf documents/report.pdf

# Upload to nested folders (creates folders automatically)
./nexus-cli upload ./archive.zip backups/2024/full_backup.zip

# Update an existing file
./nexus-cli upload ./updated_report.pdf documents/report.pdf

# Using stateless mode
./nexus-cli upload -u myusername ./file.pdf documents/file.pdf
```

**Session Behavior:**
- If you have a persistent session (`nexus.conf`), it updates the local cache
- If using stateless mode, authentication happens once per command
- After upload, index is synced to remote repository

**File Size:** No hard limits in the code; practical limits depend on GitHub's API.

---

### `download` - Download a File

Decrypts and downloads a file from the vault to your local filesystem. Supports both vault downloads and shared file downloads.

**Usage:**
```bash
./nexus-cli download <vault-path> <local-path> [--shared <share-string>]
```

**Aliases:** `down`, `d`, `get`

**Arguments:**
- `vault-path` (required): Path to the file in the vault (e.g., `documents/report.pdf`). Use `_` when downloading a shared file.
- `local-path` (required): Where to save the file on your computer

**Flags:**
- `--shared` (optional): Download a shared file using the share string. Format: `username:storage_id:decryption_key`

**Features:**
- Finds file by vault path (not storage ID)
- Automatically decrypts using per-file encryption key
- Validates file authenticity via GCM authentication
- Supports downloading shared files from other users
- Fails safely if decryption fails

**Examples:**
```bash
# Download a file from your vault
./nexus-cli download documents/report.pdf ./report.pdf

# Download from nested folders
./nexus-cli download backups/2024/full_backup.zip ./backup.zip

# Using stateless mode
./nexus-cli download -u myusername documents/report.pdf ./report.pdf

# Download with a different name
./nexus-cli download documents/report.pdf ./my_report.pdf

# Download a shared file
./nexus-cli download _ ./report.pdf --shared "john:a3f2e1c9:abc123def456..."
```

**Error Handling:**
- Returns error if file not found in vault
- Returns error if vault path points to a folder
- Returns error if decryption fails (likely wrong password or invalid share key)
- Returns error if local file write fails

---

### `delete` - Delete Files or Folders

Deletes a file or entire folder (with all contents) from the vault.

**Usage:**
```bash
./nexus-cli delete <vault-path>
```

**Aliases:** `del`, `rm`, `remove`

**Arguments:**
- `vault-path` (required): Path to the file or folder to delete

**Features:**
- Supports single file deletion
- Supports recursive folder deletion (all nested files)
- Removes files from remote repository
- Updates vault index
- Syncs changes to GitHub

**Examples:**
```bash
# Delete a single file
./nexus-cli delete documents/report.pdf

# Delete an entire folder and all contents
./nexus-cli delete documents/archive

# Using stateless mode
./nexus-cli delete -u myusername documents/report.pdf
```

**Behavior:**
- File is removed from remote repository
- Folder deletion removes all nested files recursively
- Vault index is updated immediately
- Changes are committed and pushed to GitHub

**Irreversible**: Deleted files are not recoverable unless you have git history to roll back.

---

### `ls` - List Vault Contents

Lists all files and folders in a vault directory with formatted output.

**Usage:**
```bash
./nexus-cli ls [folder]
```

**Arguments:**
- `folder` (optional): Path to list (e.g., `documents`). Defaults to root if not provided.

**Output Format:**
```
NAME            TYPE    STORAGE ID
----            ----    ----------
document.pdf    [FILE]  a3f2e1c9d4b6f8e2
archive/        [DIR]   -
```

**Examples:**
```bash
# List root of vault
./nexus-cli ls

# List a specific folder
./nexus-cli ls documents

# List nested folder
./nexus-cli ls documents/archive

# Using stateless mode
./nexus-cli ls -u myusername documents
```

**Output Details:**
- **NAME**: File or folder name (folders have "/" suffix)
- **TYPE**: `[FILE]` or `[DIR]`
- **STORAGE ID**: Hex-encoded storage ID for files, "-" for folders

**Note**: Storage IDs are randomly generated identifiers; they don't reveal original file names.

---

### `search` - Search the Vault

Searches for files and folders by name or path using case-insensitive substring matching.

**Usage:**
```bash
./nexus-cli search <query>
```

**Aliases:** `s`

**Arguments:**
- `query` (required): Search term (case-insensitive, substring matching)

**Features:**
- Case-insensitive matching
- Searches both file names and full paths
- Displays results in formatted table
- Searches all folders recursively

**Examples:**
```bash
# Search for files with "report" in the name
./nexus-cli search report
# Output:
# VAULT PATH              TYPE    STORAGE ID
# ----------              ----    ----------
# documents/report.pdf    [FILE]  a3f2e1c9d4b6f8e2
# archive/2024_report     [DIR]   -

# Search by file type
./nexus-cli search .pdf

# Search by year
./nexus-cli search 2024

# Search by path
./nexus-cli search documents/invoices

# Using stateless mode
./nexus-cli search -u myusername "backup"
```

**Search Matching:**
- Matches if query appears anywhere in the file name or full path
- "report" matches "report.pdf", "2024_report", "reports/quarterly"
- ".pdf" matches all PDF files
- "2024" matches any file or folder with "2024" in the path

**Output**: If no matches found, displays "No matches found."

---

### `share` - Generate a Share String for a File

Generates a secure share string that allows others to download a specific file without requiring access to your entire vault.

**Usage:**
```bash
./nexus-cli share <vault-path>
```

**Aliases:** `sh`

**Arguments:**
- `vault-path` (required): Path to the file in the vault (e.g., `documents/report.pdf`)

**What It Does:**
1. Locates the file in your vault index
2. Decrypts the file's encryption key using your vault password
3. Generates a share string: `username:storage_id:decryption_key`
4. Displays the share string and usage instructions

**Features:**
- Generates unique per-file share tokens
- Recipient gets access to only that file, not your entire vault
- Share token includes the unique encryption key for that file
- Can be revoked by changing the file (uploading a new version)
- Works with files from any vault

**Examples:**
```bash
# Share a file from your vault
./nexus-cli share documents/report.pdf
# Output:
# Share this string to allow others to download the file:
# john:a3f2e1c9:abc123def456789abc123def456789ab
#
# Recipient can download with:
#   nexus-cli download _ output.file --shared "john:a3f2e1c9:abc123def456789abc123def456789ab"

# Using stateless mode
./nexus-cli share -u myusername documents/sensitive_file.pdf
```

**Security Implications:**
- **Revocation**: No way to revoke a share without changing the file
- **Access**: Recipient only gets access to that specific file
- **Key Exposure**: The encryption key is exposed in the share string; share securely (encrypted email, secure messaging, etc.)
- **No Audit**: You cannot see who has downloaded the shared file

**Best Practices for Sharing:**
1. Share the string via secure channels (encrypted email, password manager, Signal, etc.)
2. Never share via unencrypted channels like plain email or messaging apps
3. For sensitive files, consider sharing a copy instead of the original
4. If compromised, delete and re-upload the file to invalidate the share

---

### `disconnect` - Remove Local Session

Clears the local session cache and removes `nexus.conf`.

**Usage:**
```bash
./nexus-cli disconnect
```

**Aliases:** `disc`, `logout`, `signout`, `logoff`, `exit`, `dc`

**What It Does:**
- Removes `nexus.conf` from the current directory
- Clears in-memory session cache
- Returns you to unauthenticated state

**Examples:**
```bash
./nexus-cli disconnect
# ✔ Logged out.

# All subsequent commands require authentication
./nexus-cli ls
# No active session. Enter GitHub Username: ...
```

**When to Use:**
- Before leaving a shared machine
- To switch to a different account
- To clear cached credentials

**Note**: Doesn't affect files in the vault, only removes local cache.

---

### `purge` - Wipe the Entire Vault

Completely removes all files and history from the remote vault. **This is irreversible.**

**Usage:**
```bash
./nexus-cli purge
```

**What It Does:**
1. Asks for confirmation (type `y` to confirm)
2. Creates an empty git repository
3. Force-pushes to GitHub, erasing all history
4. Clears the local vault index
5. Returns to unauthenticated state

**Example:**
```bash
./nexus-cli purge
# ⚠️  Confirm PURGE? This wipes all remote data and history. (y/N): y
# ✔ Remote vault has been wiped and local index cleared.
```

**⚠️ WARNING: This operation is permanent and irreversible.**
- All files are deleted
- All git history is erased
- GitHub repository cannot recover the data
- Only use if you're absolutely certain

**When to Use:**
- You want to completely reset your vault
- You're decommissioning the vault
- You suspect a security compromise and want a clean slate

---

## Usage Patterns

### Pattern 1: Interactive Shell (Recommended)

Best for multiple operations in one session:

```bash
./nexus-cli
# Username: myusername
# Password: ••••••••••
# ✔ Welcome, myusername. Session Active.

nexus> upload ./report.pdf documents/2024/Q1.pdf
nexus> upload ./budget.xlsx documents/2024/budget.xlsx
nexus> ls documents/2024
nexus> search Q1
nexus> download documents/2024/Q1.pdf ./Q1_backup.pdf
nexus> exit
```

### Pattern 2: Persistent Session

Best for scripting or multiple commands:

```bash
./nexus-cli connect myusername
# Password: ••••••••••
# ✔ Connected.

./nexus-cli upload ./file1.pdf documents/file1.pdf
./nexus-cli upload ./file2.pdf documents/file2.pdf
./nexus-cli ls documents
./nexus-cli disconnect
```

### Pattern 3: One-Off Commands with Stateless Mode

Best for single commands without persistent session:

```bash
./nexus-cli upload -u myusername ./file.pdf documents/file.pdf
# Password: ••••••••••
# ✔ Upload successful.

./nexus-cli download -u myusername documents/file.pdf ./file.pdf
# Password: ••••••••••
# ✔ Download successful.
```

### Pattern 4: Automated Vault Access (CI/CD)

Store credentials securely and run commands programmatically:

```bash
# In your automation script
export NEXUS_USER="myusername"
export NEXUS_PASS="vaultpassword"  # Handle securely!

./nexus-cli upload -u $NEXUS_USER backup.zip backups/$(date +%Y-%m-%d).zip
```

## Security Considerations

### Encryption Architecture

**Vault-Level Encryption:**
- Your GitHub SSH key is encrypted with your vault password using AES-256-GCM
- The vault index is encrypted with your vault password using AES-256-GCM
- Algorithm: AES-256-GCM (authenticated encryption)
- Key Derivation: PBKDF2 with 100,000 iterations and SHA-256
- Nonce: 12-byte random nonce per encryption

**Per-File Encryption:**
- Each file has its own unique 32-byte encryption key
- File keys are encrypted with your vault password and stored in the index
- File content is encrypted with the per-file key using AES-256-GCM
- Separate nonce for each file encryption

**Sharing:**
- Shared files expose only the per-file key, not your vault password
- Recipients cannot access other files in your vault
- Each file can have multiple share links without affecting security of other files

Your files are encrypted **before** leaving your computer. GitHub never sees unencrypted data.

### Authentication

- **SSH Keys**: Uses your GitHub SSH key for repository access
- **Vault Password**: Encrypts your GitHub SSH key
- **Password Storage**: Never stored; must be provided each session

### Best Practices

1. **Secure Your SSH Key**:
   - Don't share your GitHub SSH key
   - Use SSH key passphrases if your SSH client supports it
   - Store keys on encrypted drives

2. **Protect Your Vault Password**:
   - Use a strong, unique password (minimum 12 characters recommended)
   - Don't share it with others
   - Don't store it in plain text
   - Your vault password protects both your SSH key and all file encryption keys

3. **Manage `nexus.conf`**:
   - Don't commit `nexus.conf` to git
   - Delete it after sensitive operations
   - It's not encrypted; treat it like a session token

4. **Repository Access**:
   - Keep your `.nexus` repository private
   - Only share SSH access with trusted users
   - Consider using deploy keys for automated access

5. **File Sharing Security**:
   - Share file links only via secure channels (encrypted email, secure messaging, password managers)
   - Never share via plain email or unencrypted messaging
   - If a share link is compromised, delete and re-upload the file to invalidate it
   - Document who you've shared files with in case of security incidents

6. **Backup Your SSH Key**:
   - Store a secure copy of your GitHub SSH key
   - If lost, you'll need to regenerate it
   - Update in Nexus vault after key rotation

7. **Revoke Access**:
   - To revoke a shared file link, re-upload the file with new content
   - This generates a new per-file key, invalidating previous share links
   - Delete the file to permanently revoke all share links

## Troubleshooting

### "Auth failed: invalid password"

**Cause**: Incorrect vault password
**Solution**: Double-check your vault password and try again

### "Master key not found"

**Cause**: Vault not initialized or `.config/key` missing from repository
**Solution**: Run `./nexus-cli setup` to initialize the vault

### "Repository '.nexus' not found"

**Cause**: `.nexus` repository doesn't exist on GitHub
**Solution**: Create an empty repository named `.nexus` on GitHub

### "Failed to clone: repository not found"

**Cause**: SSH key doesn't have access to the repository
**Solution**: 
- Verify SSH key is added to GitHub account
- Test with: `ssh -T git@github.com`
- Ensure key path in setup is correct

### "Path component not found"

**Cause**: File or folder doesn't exist at that path
**Solution**: Use `ls` or `search` to find the correct path

### "Is a directory, you can only download individual files"

**Cause**: Trying to download a folder instead of a file
**Solution**: Use `ls` to find individual files within the folder

### "Decryption failed: check your password"

**Cause**: Vault password is incorrect or file is corrupted
**Solution**: Verify vault password; if file was uploaded successfully, password should work

### SSH Issues on Windows

**Problem**: SSH key not found on Windows
**Solution**:
- Use full path to key: `C:\Users\YourName\.ssh\id_ed25519`
- Or copy key to a known location and reference it
- Ensure OpenSSH is installed

### Permission Denied Errors

**Cause**: Insufficient access to GitHub repository
**Solution**:
- Verify you own the `.nexus` repository
- Check SSH key permissions: `ls -l ~/.ssh/id_ed25519` (should be 600 or 400)
- On Windows, ensure OpenSSH has correct permissions

## File Format and Structure

### Vault Structure

Your vault is stored in a GitHub repository with this structure:

```
.nexus/
├── .config/
│   ├── key        (encrypted SSH key)
│   └── index      (encrypted vault index)
├── <hex_id_1>     (encrypted file)
├── <hex_id_2>     (encrypted file)
└── <hex_id_3>     (encrypted file)
```

### Index Format

The `index` file is encrypted JSON that maps paths to storage IDs:

```json
{
  "documents": {
    "type": "folder",
    "contents": {
      "report.pdf": {
        "type": "file",
        "realName": "a3f2e1c9d4b6f8e2"
      }
    }
  }
}
```

### Encryption Format

Encrypted files follow this format:

```
[16-byte salt][12-byte nonce][encrypted data + auth tag]
```

This allows each encryption to use a unique salt and nonce, preventing patterns.

## Advanced Usage

### Switching GitHub Accounts

To use a different GitHub account:

```bash
./nexus-cli disconnect
./nexus-cli connect different_username
# Password: ••••••••••
```

This creates a new session for the different account. Note: You need a separate `.nexus` repository for each account.

### Migrating Vaults

To move your vault to a new GitHub account you can [transfer ownership of the repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/transferring-a-repository) or you can migrate manually by:

1. Create new `.nexus` repository on new account
2. `./nexus-cli setup new_username path/to/key`
3. Download all files from old vault
4. Upload them to new vault
5. Delete old `.nexus` repository

Files in the vault are never visible to GitHub or other services.

## Getting Help

- **GitHub Issues**: Report bugs at [NexusCLI/NexusCLI/issues](https://github.com/NexusCLI/NexusCLI/issues)
- **Documentation**: See the [docs folder](./docs/) for detailed module documentation
- **Security Issues**: Email instead of opening public issues

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## License

Nexus CLI is released under the MIT License. See LICENSE file for details.

## Disclaimer

Nexus CLI is provided as-is. While we've implemented industry-standard encryption, we recommend:

- Testing with non-critical files first
- Maintaining separate backups of important data
- Keeping your vault password secure
- Regularly updating to the latest version

Data loss or security issues may occur. Use at your own risk.
