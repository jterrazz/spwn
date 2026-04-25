package gate

import (
	"bytes"
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
// extension is allowed to push at the gate. Only the named cookies
// are persisted — anything else in the request body is silently
// dropped as defense-in-depth against fork extensions trying to leak
// unrelated cookies.
//
// Providers are registered by gate elements that need cookie auth
// (see XCookieProvider in x.go for the pattern), and wired into the
// CookieSync at startup in apps/gate/cmd/spwn-gate/main.go.
type CookieProvider struct {
	Name    string   `json:"name"`    // url-safe id, e.g. "x", "linkedin"
	Domains []string `json:"domains"` // suffix-matched against tab hostnames
	Cookies []string `json:"cookies"` // allowlisted cookie names
}

// CookieSync owns the /sync/* endpoints — the receive-end of the
// browser extension. Wired into the server's mux alongside /mcp/*.
//
// Trust model: localhost-only binding + cookie-name allowlist. No
// shared secret. The gate listens on 127.0.0.1, so the only callers
// that can reach /sync are processes already running on the user's
// machine; anything that has local execution can already do worse.
// Defense-in-depth is the cookie allowlist (only the named cookies
// per provider are persisted; anything else is dropped silently).
type CookieSync struct {
	mu        sync.RWMutex
	providers map[string]CookieProvider // keyed by name
	lastSync  map[string]time.Time      // provider name → most-recent successful sync
}

// NewCookieSync returns an empty sync service. Register providers
// before calling RegisterRoutes (typically from element constructors
// in apps/gate/cmd/spwn-gate/main.go).
func NewCookieSync() *CookieSync {
	return &CookieSync{
		providers: map[string]CookieProvider{},
		lastSync:  map[string]time.Time{},
	}
}

// RegisterProvider adds a provider to the registry. Idempotent
// (later calls with the same Name overwrite the earlier entry).
// Concurrent-safe.
func (cs *CookieSync) RegisterProvider(p CookieProvider) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.providers[p.Name] = p
}

// RegisterRoutes wires /sync/* into mux.
func (cs *CookieSync) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/sync/providers", cs.handleProviders)
	mux.HandleFunc("/sync/status", cs.handleStatus)
	mux.HandleFunc("/sync/", cs.handlePush)
}

func (cs *CookieSync) handleProviders(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Localhost-only binding makes CORS lax safe — the only origins
	// that can reach this endpoint live on this machine.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_ = json.NewEncoder(w).Encode(cs.Providers())
}

func (cs *CookieSync) handleStatus(w http.ResponseWriter, _ *http.Request) {
	type row struct {
		Name       string   `json:"name"`
		Domains    []string `json:"domains"`
		LastSync   string   `json:"last_sync,omitempty"`
		HasCookies bool     `json:"has_cookies"`
	}
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	rows := make([]row, 0, len(cs.providers))
	for _, p := range cs.providers {
		r := row{Name: p.Name, Domains: p.Domains}
		if t, ok := cs.lastSync[p.Name]; ok {
			r.LastSync = t.UTC().Format(time.RFC3339)
		}
		// has_cookies is on-disk presence — survives gate restarts so
		// the popup can still say "connected" even if in-memory
		// lastSync is empty after a fresh start.
		if _, err := os.Stat(CookieFile(p.Name)); err == nil {
			r.HasCookies = true
		}
		rows = append(rows, r)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_ = json.NewEncoder(w).Encode(map[string]any{"providers": rows})
}

func (cs *CookieSync) handlePush(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, "/sync/")
	if tail == "" || tail == "providers" || tail == "status" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
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
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	p, ok := cs.providers[name]
	if !ok {
		return nil
	}
	return &p
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

// ProviderLastSync returns the most recent in-memory last-sync time
// for a provider, or zero. CLI status uses this to render "synced
// 2 min ago" rows.
func (cs *CookieSync) ProviderLastSync(name string) time.Time {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.lastSync[name]
}

// WriteDomainHints persists each provider's `Domains` to
// /credentials/<provider>/.domains (one host per line) so the
// gate-browser sidecar can seed cookies on the right hosts when a
// session for that provider is opened. Idempotent — overwrites on
// every gate startup.
func (cs *CookieSync) WriteDomainHints() error {
	for _, p := range cs.Providers() {
		dir := CookieDir(p.Name)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		var buf bytes.Buffer
		for _, d := range p.Domains {
			buf.WriteString(d)
			buf.WriteByte('\n')
		}
		if err := os.WriteFile(filepath.Join(dir, ".domains"), buf.Bytes(), 0o600); err != nil {
			return fmt.Errorf("write hints for %s: %w", p.Name, err)
		}
	}
	return nil
}

// Providers returns a copy of the registered provider list, sorted
// by name for deterministic output (popup + tests rely on it).
func (cs *CookieSync) Providers() []CookieProvider {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	out := make([]CookieProvider, 0, len(cs.providers))
	for _, p := range cs.providers {
		out = append(out, p)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1].Name > out[j].Name; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
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
