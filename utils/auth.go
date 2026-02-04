package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const configPath = "zephyrus.conf"

// globalSession stores the session in RAM for the REPL/Stateless mode
var globalSession *Session

type Session struct {
	Username    string        `json:"username"`
	Password    string        `json:"password"`
	RawKey      []byte        `json:"raw_key"`
	Index       VaultIndex    `json:"index"`
	SharedIndex *SharedIndex  `json:"shared_index"`
	Settings    VaultSettings `json:"settings"`
}

// SetGlobalSession injects a session into RAM (used by the REPL)
func SetGlobalSession(s *Session) {
	globalSession = s
}

// Connect initializes the session and syncs the index locally
func Connect(username string, password string) error {
	fmt.Printf("Connecting and syncing vault for %s...\n", username)

	session, err := FetchSessionStateless(username, password)
	if err != nil {
		return err
	}

	return session.Save()
}

func (s *Session) Save() error {
	data, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(configPath, data, 0600)
}

// GetSession now checks memory first (REPL cache), then falls back to disk
func GetSession() (*Session, error) {
	// 1. Check RAM (REPL/Interactive mode)
	if globalSession != nil {
		return globalSession, nil
	}

	// 2. Check Disk (Standard CLI mode)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("not connected: run 'connect' first or use -u")
	}
	var s Session
	err = json.Unmarshal(data, &s)

	// 3. Ensure SharedIndex is initialized
	if s.SharedIndex == nil {
		s.SharedIndex = NewSharedIndex()
	}

	// 4. Apply defaults to settings if missing (for backward compatibility with old zephyrus.conf files)
	if s.Settings.CommitAuthorName == "" {
		s.Settings.CommitAuthorName = "Zephyrus"
	}
	if s.Settings.CommitAuthorEmail == "" {
		s.Settings.CommitAuthorEmail = "auchrio@proton.me"
	}
	if s.Settings.CommitMessage == "" {
		s.Settings.CommitMessage = "Zephyrus: Updated Vault"
	}
	if s.Settings.FileHashLength <= 0 {
		s.Settings.FileHashLength = 16
	}
	if s.Settings.ShareHashLength <= 0 {
		s.Settings.ShareHashLength = 6
	}

	return &s, err
}

func Disconnect() error {
	globalSession = nil // Clear memory cache
	return os.Remove(configPath)
}

// FetchSessionStateless performs the authentication and index fetch without saving to disk
func FetchSessionStateless(username string, password string) (*Session, error) {
	// 1. Fetch & Decrypt Master Key
	encryptedKey, err := FetchRaw(username, ".config/key")
	if err != nil {
		return nil, fmt.Errorf("master key not found: %w", err)
	}
	rawKey, err := Decrypt(encryptedKey, password)
	if err != nil {
		return nil, fmt.Errorf("auth failed: invalid password")
	}

	// 2. Fetch & Decrypt Index
	var index VaultIndex
	rawIndex, err := FetchRaw(username, ".config/index")
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			index = NewIndex()
		} else {
			index = NewIndex() // Fallback for new vaults
		}
	} else {
		index, err = FromBytes(rawIndex, password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt index: %w", err)
		}
	}

	// 3. Fetch & Decrypt Shared Index
	var sharedIndex *SharedIndex
	rawSharedIndex, err := FetchRaw(username, "shared/.config/index")
	if err != nil {
		// Shared index doesn't exist yet, that's fine
		sharedIndex = NewSharedIndex()
	} else {
		sharedIndex, err = DecryptSharedIndex(rawSharedIndex, password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt shared index: %w", err)
		}
	}

	// 4. Fetch & Decrypt Settings (use defaults if not present)
	var settings VaultSettings
	rawSettings, err := FetchRaw(username, ".config/settings")
	if err != nil {
		// Settings don't exist yet, use defaults
		settings = DefaultSettings()
	} else {
		settings, err = SettingsFromBytes(rawSettings, password)
		if err != nil {
			// If settings are corrupted, fall back to defaults
			settings = DefaultSettings()
		}
	}

	return &Session{
		Username:    username,
		Password:    password,
		RawKey:      rawKey,
		Index:       index,
		SharedIndex: sharedIndex,
		Settings:    settings,
	}, nil
}
