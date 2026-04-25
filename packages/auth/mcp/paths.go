package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"

	"spwn.sh/packages/platform"
)

// CacheDir is the host directory we hand to mcp2cli as its
// MCP2CLI_CACHE_DIR. It also gets bind-mounted into every world
// spawn at the in-container default ($HOME/.cache/mcp2cli) so the
// OAuth tokens persisted here survive the container.
//
// Layout (mirrors mcp2cli's own layout):
//
//	~/.spwn/credentials/mcp/
//	└── oauth/
//	    └── <sha256(server_url)[:16]>/
//	        ├── tokens.json    # access + refresh tokens
//	        └── client.json    # DCR-issued client_id/secret
func CacheDir() string {
	return filepath.Join(platform.CredentialsDir(), "mcp")
}

// providerKey mirrors mcp2cli's FileTokenStorage hash:
//
//	hashlib.sha256(server_url.encode()).hexdigest()[:16]
//
// Keep these in lock-step — the on-disk directory is what makes
// the host cache reusable from inside the container.
func providerKey(url string) string {
	sum := sha256.Sum256([]byte(url))
	return hex.EncodeToString(sum[:])[:16]
}

// ProviderTokenPath returns the path to tokens.json for p, whether
// or not it exists yet.
func ProviderTokenPath(p Provider) string {
	return filepath.Join(CacheDir(), "oauth", providerKey(p.URL), "tokens.json")
}

// ProviderClientPath returns the path to client.json (DCR-issued
// client_id/secret) for p, whether or not it exists yet.
func ProviderClientPath(p Provider) string {
	return filepath.Join(CacheDir(), "oauth", providerKey(p.URL), "client.json")
}
