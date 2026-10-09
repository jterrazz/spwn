package gh

import (
	"path/filepath"

	"spwn.sh/packages/platform"
)

// CacheDir is the host directory we point GH_CONFIG_DIR at when
// driving `gh` commands, and the host side of the bind-mount that
// surfaces gh's hosts.yml inside every spwn world. Layout mirrors
// gh's own:
//
//	~/.spwn/credentials/gh/
//	├── hosts.yml      # github.com host metadata + plaintext token
//	└── config.yml     # general gh config (optional)
func CacheDir() string {
	return filepath.Join(platform.CredentialsDir(), "gh")
}

// HostsPath returns the path to hosts.yml inside CacheDir, whether
// or not it exists yet.
func HostsPath() string {
	return filepath.Join(CacheDir(), "hosts.yml")
}
