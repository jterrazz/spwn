package google

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Login runs Google's OAuth installed-app flow with PKCE and
// persists the resulting tokens. Blocks until either:
//
//   - the user clicks Allow and the loopback callback completes, or
//   - the helper times out (5 minutes), or
//   - the user cancels (Ctrl-C cancels ctx).
//
// Output (the "open this URL" hint, browser launch attempt) is
// streamed to w so the user sees what's happening live.
func Login(ctx context.Context, c *ClientConfig, w io.Writer) error {
	if c == nil || c.ClientID == "" {
		return fmt.Errorf("nil or empty ClientConfig — run the wizard first")
	}
	if w == nil {
		w = io.Discard
	}
	scopes := c.Scopes
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}

	// Pick a free loopback port so concurrent logins (or other
	// processes binding 8080) don't collide.
	port, err := freeTCPPort()
	if err != nil {
		return fmt.Errorf("pick callback port: %w", err)
	}
	redirect := "http://127.0.0.1:" + strconv.Itoa(port) + "/callback"

	verifier, challenge, err := newPKCE()
	if err != nil {
		return fmt.Errorf("generate PKCE: %w", err)
	}
	state, err := newRandom(16)
	if err != nil {
		return err
	}

	// Build the authorization URL.
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", redirect)
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("access_type", "offline") // ask Google for a refresh_token
	q.Set("prompt", "consent")      // force consent so refresh_token is always returned
	authURL := AuthorizationEndpoint + "?" + q.Encode()

	// Spin up the loopback callback server.
	type result struct {
		code  string
		state string
		err   error
	}
	done := make(chan result, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(rw http.ResponseWriter, r *http.Request) {
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			done <- result{err: fmt.Errorf("OAuth error: %s", errMsg)}
			fmt.Fprintln(rw, "Login failed:", errMsg)
			return
		}
		code := r.URL.Query().Get("code")
		got := r.URL.Query().Get("state")
		if got != state {
			done <- result{err: fmt.Errorf("state mismatch (CSRF protection)")}
			http.Error(rw, "state mismatch", http.StatusBadRequest)
			return
		}
		if code == "" {
			done <- result{err: fmt.Errorf("no authorization code in callback")}
			http.Error(rw, "no code", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(rw, "spwn auth login google: success — you can close this tab.")
		done <- result{code: code, state: got}
	})
	server := &http.Server{Addr: "127.0.0.1:" + strconv.Itoa(port), Handler: mux}
	go func() { _ = server.ListenAndServe() }()
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutCtx)
	}()

	// Stream the URL + try to open the browser. The browser open is
	// best-effort: on headless systems the user copy-pastes from the
	// printed URL.
	fmt.Fprintln(w, "Open this URL in your browser to authorize spwn:")
	fmt.Fprintln(w, authURL)
	_ = openBrowser(authURL)

	// Wait for the callback or timeout.
	var res result
	select {
	case res = <-done:
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("OAuth timed out after 5 minutes")
	case <-ctx.Done():
		return ctx.Err()
	}
	if res.err != nil {
		return res.err
	}

	// Exchange the authorization code for tokens.
	tokens, err := exchangeCode(ctx, c, redirect, res.code, verifier)
	if err != nil {
		return fmt.Errorf("exchange code: %w", err)
	}
	if err := SaveTokens(tokens); err != nil {
		return fmt.Errorf("save tokens: %w", err)
	}
	return nil
}

// exchangeCode trades an authorization code for tokens using the
// PKCE code_verifier (no client_secret required for Desktop apps,
// but we send it when present for Web app type clients).
func exchangeCode(ctx context.Context, c *ClientConfig, redirect, code, verifier string) (*Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", c.ClientID)
	if c.ClientSecret != "" {
		form.Set("client_secret", c.ClientSecret)
	}
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	form.Set("redirect_uri", redirect)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint %d: %s", resp.StatusCode, body)
	}
	var t Tokens
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if t.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}
	return &t, nil
}

// newPKCE generates a code_verifier (43-128 unreserved chars per
// RFC 7636) and the matching SHA-256 code_challenge.
func newPKCE() (verifier, challenge string, err error) {
	verifier, err = newRandom(64)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

// newRandom returns a URL-safe random string of length n bytes
// (base64-url no-pad encoded → ~1.33n chars).
func newRandom(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func freeTCPPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// openBrowser is best-effort — failures are non-fatal because the
// auth URL is also printed to the terminal for headless / SSH cases.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("don't know how to open URL on %s", runtime.GOOS)
	}
	return cmd.Start()
}
