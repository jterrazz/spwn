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
		return fmt.Errorf("nil or empty ClientConfig: run the wizard first")
	}
	if w == nil {
		w = io.Discard
	}
	scopes := c.Scopes
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}

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

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", redirect)
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	authURL := AuthorizationEndpoint + "?" + q.Encode()

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
		fmt.Fprintln(rw, "spwn auth login google: success, you can close this tab.")
		done <- result{code: code, state: got}
	})
	server := &http.Server{Addr: "127.0.0.1:" + strconv.Itoa(port), Handler: mux}
	go func() { _ = server.ListenAndServe() }()
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutCtx)
	}()

	fmt.Fprintln(w, "Open this URL in your browser to authorize spwn:")
	fmt.Fprintln(w, authURL)
	_ = openBrowser(authURL)

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

	tokens, err := exchangeCode(ctx, c, redirect, res.code, verifier)
	if err != nil {
		return fmt.Errorf("exchange code: %w", err)
	}
	if err := SaveTokens(tokens); err != nil {
		return fmt.Errorf("save tokens: %w", err)
	}
	return nil
}

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

func newPKCE() (verifier, challenge string, err error) {
	verifier, err = newRandom(64)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

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
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("listener bound to %T, not TCP", l.Addr())
	}
	return addr.Port, nil
}

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
