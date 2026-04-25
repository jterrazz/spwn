package gate

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"spwn.sh/packages/platform"
)

// CookieProvider is one site whose session cookies the spwn-cookie-sync
// extension is allowed to push at the gate. Only the named cookies are
// persisted — anything else in the request body is silently dropped.
type CookieProvider struct {
	Name    string   `json:"name"`    // url-safe id, e.g. "x", "linkedin"
	Domains []string `json:"domains"` // suffix-matched against tab hostnames
	Cookies []string `json:"cookies"` // allowlisted cookie names
}

// DefaultCookieProviders is the seed registry. Add new entries here
// as new gate elements gain cookie-auth support; the extension's
// /sync/providers fetch picks them up at next refresh tick.
var DefaultCookieProviders = []CookieProvider{
	{Name: "x", Domains: []string{"x.com", "twitter.com"}, Cookies: []string{"auth_token", "ct0"}},
	{Name: "linkedin", Domains: []string{"linkedin.com"}, Cookies: []string{"li_at", "JSESSIONID"}},
}

// CookieSync owns the /sync/* endpoints — the receive-end of the
// browser extension. Wired into the server's mux alongside /mcp/*.
//
// State held on disk under ~/.spwn/gate/ on the host (bind-mounted
// to /gate inside the container):
//
//   /gate/cookie-sync-secret   — paired-extension shared secret (0600)
//
// Persisted cookies land under ~/.spwn/credentials/<provider>/cookies.json
// so the same path is reachable both from the gate (rw) and from
// other elements that read them (XActions, linkedin-mcp, …).
type CookieSync struct {
	providers []CookieProvider

	mu        sync.Mutex
	lastSync  map[string]time.Time // provider name → most-recent successful sync
	secret    string               // cached after first read; "" until register
	secretMTime time.Time
}

// NewCookieSync builds the sync service with the default provider
// list. Call AddProvider before serving for custom additions.
func NewCookieSync() *CookieSync {
	cs := &CookieSync{
		providers: append([]CookieProvider(nil), DefaultCookieProviders...),
		lastSync:  map[string]time.Time{},
	}
	_ = cs.loadSecret() // best-effort; absent secret = unpaired
	return cs
}

// AddProvider registers an extra provider. Not concurrent-safe; call
// during startup before RegisterRoutes.
func (cs *CookieSync) AddProvider(p CookieProvider) { cs.providers = append(cs.providers, p) }

// RegisterRoutes wires /sync/* into mux. /mcp/* stays its own
// namespace handled elsewhere in server.go.
func (cs *CookieSync) RegisterRoutes(mux *http.ServeMux) {
	// /sync/providers — public, no secret required (extension fetches
	// this before pairing to know what to listen for).
	mux.HandleFunc("/sync/providers", cs.handleProviders)

	// /sync/status — paired-only, used by popup to show last-sync timestamps.
	mux.HandleFunc("/sync/status", cs.handleStatus)

	// /sync/<provider> — paired-only, accepts cookies and persists.
	// Catch-all under /sync/; provider name is the path tail.
	mux.HandleFunc("/sync/", cs.handlePush)
}

func (cs *CookieSync) handleProviders(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "chrome-extension://")
	_ = json.NewEncoder(w).Encode(cs.providers)
}

func (cs *CookieSync) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !cs.checkSecret(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	cs.mu.Lock()
	type row struct {
		Name     string `json:"name"`
		LastSync string `json:"last_sync,omitempty"`
	}
	rows := make([]row, 0, len(cs.providers))
	for _, p := range cs.providers {
		r := row{Name: p.Name}
		if t, ok := cs.lastSync[p.Name]; ok {
			r.LastSync = t.UTC().Format(time.RFC3339)
		}
		rows = append(rows, r)
	}
	cs.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"paired": true, "providers": rows})
}

func (cs *CookieSync) handlePush(w http.ResponseWriter, r *http.Request) {
	// /sync/providers + /sync/status are handled above; this is the
	// catch-all for /sync/<provider>. Skip the well-known sub-paths.
	tail := strings.TrimPrefix(r.URL.Path, "/sync/")
	if tail == "" || tail == "providers" || tail == "status" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if !cs.checkSecret(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	provider := cs.providerByName(tail)
	if provider == nil {
		http.Error(w, fmt.Sprintf("unknown provider %q", tail), http.StatusNotFound)
		return
	}

	var body struct {
		Cookies  map[string]string `json:"cookies"`
		Captured string            `json:"captured"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Drop any cookie name not in the allowlist — defense-in-depth
	// against an extension fork that tries to leak unrelated cookies.
	allowed := map[string]bool{}
	for _, n := range provider.Cookies {
		allowed[n] = true
	}
	clean := map[string]string{}
	for k, v := range body.Cookies {
		if allowed[k] {
			clean[k] = v
		}
	}
	if len(clean) == 0 {
		http.Error(w, "no allowlisted cookies in body", http.StatusBadRequest)
		return
	}

	out := map[string]any{
		"cookies":  clean,
		"captured": body.Captured,
		"updated":  time.Now().UTC().Format(time.RFC3339),
	}
	raw, _ := json.MarshalIndent(out, "", "  ")
	if err := writeProviderCookies(provider.Name, raw); err != nil {
		http.Error(w, "persist failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	cs.mu.Lock()
	cs.lastSync[provider.Name] = time.Now()
	cs.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (cs *CookieSync) providerByName(name string) *CookieProvider {
	for i := range cs.providers {
		if cs.providers[i].Name == name {
			return &cs.providers[i]
		}
	}
	return nil
}

// checkSecret validates the X-Spwn-Secret header against the secret
// stored in /gate/cookie-sync-secret. Re-reads the file on each call
// so `spwn cookie-sync register --rotate` takes effect without a
// gate restart.
func (cs *CookieSync) checkSecret(r *http.Request) bool {
	got := r.Header.Get("X-Spwn-Secret")
	if got == "" {
		return false
	}
	if err := cs.loadSecret(); err != nil || cs.secret == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(cs.secret)) == 1
}

// loadSecret refreshes the cached secret from disk if the file's
// mtime has changed. Cheap (one stat per request); avoids re-reading
// the file when it hasn't moved.
func (cs *CookieSync) loadSecret() error {
	path := SecretPath()
	fi, err := os.Stat(path)
	if err != nil {
		cs.secret = ""
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !fi.ModTime().After(cs.secretMTime) && cs.secret != "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cs.secret = strings.TrimSpace(string(raw))
	cs.secretMTime = fi.ModTime()
	return nil
}

// SecretPath returns the on-disk location of the pairing secret.
func SecretPath() string {
	return filepath.Join(platform.UserDir(), "gate", "cookie-sync-secret")
}

// CookieDir returns the directory where /sync/<name> persists per-
// provider cookies.json files. Bind-mounted into /credentials.
func CookieDir(provider string) string {
	return filepath.Join(platform.CredentialsDir(), provider)
}

// CookieFile is the canonical cookies.json path for a provider.
func CookieFile(provider string) string {
	return filepath.Join(CookieDir(provider), "cookies.json")
}

// GenerateSecret creates a fresh pairing secret and writes it to
// SecretPath() with 0600 perms. Returns the human-formatted display
// string (SP-XXXX-XXXX-XXXX) the user pastes into the popup.
func GenerateSecret() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	raw := strings.ToUpper(hex.EncodeToString(b))
	display := "SP-" + raw[0:6] + "-" + raw[6:12] + "-" + raw[12:18] + "-" + raw[18:24]

	dir := filepath.Dir(SecretPath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(SecretPath(), []byte(display), 0o600); err != nil {
		return "", err
	}
	return display, nil
}

// HasSecret reports whether the gate has been paired (a secret file
// exists). Used by the CLI dashboard.
func HasSecret() bool {
	_, err := os.Stat(SecretPath())
	return err == nil
}

// ProviderLastSync returns the most recent in-memory last-sync time
// for a provider, or zero. CLI status uses this to render "synced
// 2 min ago" rows.
func (cs *CookieSync) ProviderLastSync(name string) time.Time {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.lastSync[name]
}

// Providers returns a copy of the registered provider list.
func (cs *CookieSync) Providers() []CookieProvider {
	out := make([]CookieProvider, len(cs.providers))
	copy(out, cs.providers)
	return out
}

func writeProviderCookies(provider string, body []byte) error {
	dir := CookieDir(provider)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".cookies-*.tmp")
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
	return os.Rename(tmp.Name(), CookieFile(provider))
}
