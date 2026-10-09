package google

import "time"

// ClientConfig is the on-disk shape of client.json. Captured by the
// wizard from the GCP console.
type ClientConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"` // optional for Desktop apps using PKCE
	Scopes       []string `json:"scopes"`
}

// Tokens is the on-disk shape of tokens.json. Mirrors the standard
// OAuth 2.0 token response so an unmarshal of the token endpoint
// reply lands here directly.
type Tokens struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"` // seconds, present in fresh responses
	ExpiresAt    time.Time `json:"expires_at,omitempty"` // computed on save
}

// Expired reports whether t is past its expires_at, with a leeway
// applied so callers don't race the clock on a borderline-fresh
// token.
func (t Tokens) Expired(leeway time.Duration) bool {
	if t.ExpiresAt.IsZero() {
		return false // unknown expiry — treat as fresh, refresh if API returns 401
	}
	return time.Now().Add(leeway).After(t.ExpiresAt)
}

// DefaultScopes is the minimum set covering the gmail/gcal tool
// surface the gate exposes today (search/read threads, manage
// drafts, list labels; full calendar). Users can expand at login
// time with --scope flags.
var DefaultScopes = []string{
	"https://www.googleapis.com/auth/gmail.modify",
	"https://www.googleapis.com/auth/calendar",
}

// AuthorizationEndpoint and TokenEndpoint are Google's well-known
// URLs. They're stable across all Workspace OAuth flows and don't
// vary by tenant, so we hardcode rather than discover.
const (
	AuthorizationEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
	TokenEndpoint         = "https://oauth2.googleapis.com/token"
)
