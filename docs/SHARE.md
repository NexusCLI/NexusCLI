# share.go - File Sharing Module

## Overview

The `share.go` module handles generation of secure share strings that allow users to share individual files from their vault without exposing the entire vault or requiring the recipient to know the vault password.

**File**: `utils/share.go`

---

## Core Functions

### `ShareFile(vaultPath string, session *Session) (string, error)`

Generates a shareable access token for a specific file in the vault.

**Signature:**
```go
func ShareFile(vaultPath string, session *Session) (string, error)
```

**Parameters:**
- `vaultPath` (string): Path to the file within the vault (e.g., `documents/report.pdf`)
- `session` (*Session): Active session containing username, password, and decrypted index

**Returns:**
- `string`: Share string in format `username:storage_id:hex_encoded_file_key`
- `error`: Non-nil if file not found, decryption fails, or index is corrupted

**Behavior:**

1. **Locate File**: Searches the vault index for the file at `vaultPath`
2. **Decrypt Key**: Retrieves the encrypted per-file key from the index entry
3. **Verify Password**: Decrypts the file key using the vault password (PBKDF2 key derivation)
4. **Encode Key**: Converts the raw 32-byte file key to hexadecimal for transmission
5. **Format String**: Constructs and returns: `{username}:{storage_id}:{hex_file_key}`

**Example Usage:**
```go
session := &Session{
    Username: "john",
    Password: "vault_password",
    Index: vaultIndex,
}

shareString, err := ShareFile("documents/report.pdf", session)
if err != nil {
    log.Fatal(err)
}
fmt.Println(shareString)
// Output: john:a3f2e1c9:abc123def456789abc123def456789ab
```

---

## Share String Format

### Structure

```
username:storage_id:decryption_key
```

**Components:**
- `username`: GitHub username of the vault owner
- `storage_id`: Unique identifier for the file in the vault (hex-encoded)
- `decryption_key`: Hexadecimal-encoded 32-byte per-file encryption key

### Example

```
john:a3f2e1c9d4b6f8e2:5a7e9c3b1f8d4a2e6b9c1d8e5a7b3c4f
```

---

## Security Considerations

### What the Share String Exposes

- **File Key**: The per-file encryption key, allowing decryption of that specific file
- **Storage ID**: The unique identifier where the file is stored
- **Username**: The vault owner's GitHub username

### What the Share String Does NOT Expose

- ❌ Vault password
- ❌ SSH private key
- ❌ Other files in the vault
- ❌ Vault index structure
- ❌ Access to any other vault operations

### Revocation

Shared files **cannot be unshared** without changing the file:

1. If you want to revoke access, **delete and re-upload** the file with new content
2. This generates a new per-file key and storage ID
3. Previous share strings become invalid
4. The file has a new share string for future sharing

**Note**: Once shared, you cannot track who has the share string or when it's used.

---

## Integration with CLI

### Command

```bash
./zep share <vault-path>
```

**Alias**: `sh`

### Implementation (main.go)

The `share` CLI command calls `ShareFile()` and displays the result:

```go
var shareCmd = &cobra.Command{
    Use:     "share [vault-path]",
    Aliases: []string{"sh"},
    Short:   "Generate a share string for a file",
    Args:    cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        session, err := getEffectiveSession()
        if err != nil {
            log.Fatal(err)
        }

        shareString, err := utils.ShareFile(args[0], session)
        if err != nil {
            log.Fatal(err)
        }

        fmt.Println("Share this string to allow others to download the file:")
        fmt.Println(shareString)
        fmt.Printf("  zep download _ output.file --shared \"%s\"\n", shareString)
    },
}
```

---

## Usage Workflow

### For File Owner

1. **Generate Share String**
   ```bash
   ./zep share documents/report.pdf
   ```

2. **Get Output**
   ```
   Share this string to allow others to download the file:
   john:a3f2e1c9:5a7e9c3b1f8d4a2e6b9c1d8e5a7b3c4f
   
   Recipient can download with:
     zep download _ output.file --shared "john:a3f2e1c9:5a7e9c3b1f8d4a2e6b9c1d8e5a7b3c4f"
   ```

3. **Share Securely**
   - Use encrypted email, password manager, or secure messaging
   - Never share via plain email or unencrypted channels
   - Document recipients for security audit purposes

### For Recipient

1. **Receive Share String** (via secure channel)
2. **Download Shared File**
   ```bash
   ./zep download _ report.pdf --shared "john:a3f2e1c9:5a7e9c3b1f8d4a2e6b9c1d8e5a7b3c4f"
   ```
3. **File is decrypted** using the provided per-file key

---

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `path not found` | File doesn't exist at vault path | Use `ls` or `search` to find correct path |
| `is a directory` | Specified path is a folder | Share a specific file within the folder |
| `decryption failed` | Wrong vault password in session | Reconnect with correct password |
| `invalid index` | Corrupted vault index | Re-run `setup` or check repository |

---

## Cryptographic Details

### File Key Encryption

File keys are stored in the vault index encrypted with the vault password:

```
File Key (32 bytes)
    ↓
[Vault Password: PBKDF2-SHA256 × 100,000]
    ↓
AES-256-GCM Encryption
    ↓
Encrypted Key (stored in index)
```

### Key Encoding

Raw bytes are converted to hexadecimal for the share string:

```go
hex.EncodeToString(fileKey[:]) // 32 bytes → 64 hex characters
```

---

## Related Functions

- **[upload.go](UPLOAD.md)** - `GenerateFileKey()` creates per-file keys
- **[encryption.go](ENCRYPTION.md)** - `EncodeKey()` converts keys to hex
- **[download.go](DOWNLOAD.md)** - `DownloadSharedFile()` uses share strings
- **[index.go](INDEX.md)** - `FindEntry()` locates files in vault

---

## Best Practices

### When to Use Share

✅ **Good Use Cases:**
- Share a single report with a colleague
- Send a file to a client without exposing vault
- Allow team member to access one document
- Share medical records with a provider

### When NOT to Use Share

❌ **Avoid:**
- Sharing via unencrypted email
- Including share string in chat messages
- Posting in public channels
- Sharing with untrusted recipients
- Long-term file access (share expires when file changes)

### Security Checklist

- [ ] Use secure channel to share the string
- [ ] Document who received the share
- [ ] Set reminder to revoke if needed
- [ ] Re-upload file monthly to rotate keys
- [ ] Delete old files to prevent accidental sharing

---

## Performance

**Time Complexity**: O(n) where n = depth of file path in index tree
- Index lookup: O(n) traversal
- Decryption: O(1) constant time per file key
- Encoding: O(1) for 32-byte key

**Network I/O**: None (operation is local on decrypted index)

---

## Future Enhancements

Planned improvements for file sharing:

- [ ] **Share Expiration**: Auto-expire share strings after N days
- [ ] **Download Tracking**: Record who downloads via share string
- [ ] **Access Limits**: Limit share to N downloads
- [ ] **ACLs**: Share with multiple users under access control list
- [ ] **Revocation Log**: Track revocations and invalidated shares

---

## Testing

### Unit Test Example

```go
func TestShareFile(t *testing.T) {
    session := &Session{
        Username: "testuser",
        Password: "password123",
        Index: testIndex,
    }
    
    shareString, err := ShareFile("test/file.pdf", session)
    if err != nil {
        t.Fatal(err)
    }
    
    // Verify format
    parts := strings.Split(shareString, ":")
    if len(parts) != 3 {
        t.Errorf("Expected 3 parts, got %d", len(parts))
    }
    
    if parts[0] != "testuser" {
        t.Errorf("Expected username testuser, got %s", parts[0])
    }
}
```

---

**Module Version**: 1.0.0  
**Last Updated**: February 2026  
**Status**: Stable
