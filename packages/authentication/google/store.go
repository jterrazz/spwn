package google

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LoadClient reads ClientConfig from ClientPath(). Returns
// (nil, nil) if absent — callers (the wizard) treat that as "user
// hasn't done first-run setup yet".
func LoadClient() (*ClientConfig, error) {
	raw, err := os.ReadFile(ClientPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read client.json: %w", err)
	}
	var c ClientConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse client.json: %w", err)
	}
	if c.ClientID == "" {
		return nil, fmt.Errorf("client.json has empty client_id")
	}
	return &c, nil
}

// SaveClient writes ClientConfig atomically.
func SaveClient(c *ClientConfig) error {
	if c == nil || c.ClientID == "" {
		return fmt.Errorf("invalid client config: empty client_id")
	}
	if err := os.MkdirAll(CacheDir(), 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(ClientPath(), body)
}

// LoadTokens reads the cached tokens. (nil, nil) when absent.
func LoadTokens() (*Tokens, error) {
	raw, err := os.ReadFile(TokensPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tokens.json: %w", err)
	}
	var t Tokens
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, fmt.Errorf("parse tokens.json: %w", err)
	}
	return &t, nil
}

// SaveTokens writes the tokens, populating ExpiresAt from ExpiresIn
// (if the latter was set by a fresh OAuth response). Atomic.
func SaveTokens(t *Tokens) error {
	if t == nil || t.AccessToken == "" {
		return fmt.Errorf("invalid tokens: empty access_token")
	}
	if t.ExpiresIn > 0 && t.ExpiresAt.IsZero() {
		t.ExpiresAt = time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)
	}
	if err := os.MkdirAll(CacheDir(), 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(TokensPath(), body)
}

// IsAuthenticated reports whether a tokens.json exists. Token
// freshness is the responsibility of AccessToken (which refreshes
// on demand).
func IsAuthenticated() bool {
	_, err := os.Stat(TokensPath())
	return err == nil
}

func atomicWrite(path string, body []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
