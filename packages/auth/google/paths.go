package google

import (
	"path/filepath"

	"spwn.sh/packages/platform"
)

// CacheDir is the host directory that holds Google OAuth artifacts.
// Bind-mounted into the gate as /credentials/google.
//
// Layout:
//
//	~/.spwn/credentials/google/
//	├── client.json   # client_id, client_secret (optional), scopes
//	└── tokens.json   # access_token, refresh_token, expiry
func CacheDir() string {
	return filepath.Join(platform.CredentialsDir(), "google")
}

// ClientPath is where the user's OAuth client config lives.
// Captured by the wizard on first `spwn auth login google`.
func ClientPath() string {
	return filepath.Join(CacheDir(), "client.json")
}

// TokensPath is where the access + refresh tokens live after OAuth.
func TokensPath() string {
	return filepath.Join(CacheDir(), "tokens.json")
}
