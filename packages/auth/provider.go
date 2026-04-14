package auth

// Provider represents a supported AI provider.
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderGoogle    Provider = "google"
)

// CredentialType indicates how a credential was obtained.
type CredentialType string

const (
	CredTypeAPIKey   CredentialType = "api_key"
	CredTypeOAuth    CredentialType = "oauth"
	CredTypeKeychain CredentialType = "keychain"
	CredTypeNone     CredentialType = "none"
)

// Credential holds a resolved credential and its metadata.
type Credential struct {
	Provider Provider
	Type     CredentialType
	Token    string
	Source   string // human-readable source description ("env:ANTHROPIC_API_KEY", "file:~/.spwn/.auth-token", "keychain")
	EnvVar   string // the env var to use when injecting (ANTHROPIC_API_KEY or CLAUDE_CODE_OAUTH_TOKEN)
}

// ProviderStatus represents the health/usage of a provider.
type ProviderStatus struct {
	Provider  Provider       `json:"provider"`
	Connected bool           `json:"connected"`
	CredType  CredentialType `json:"credentialType"`
	Source    string         `json:"source"`
	Error     string         `json:"error,omitempty"`
	Plan      string         `json:"plan,omitempty"`
	Email     string         `json:"email,omitempty"`
	Usage     *UsageInfo     `json:"usage,omitempty"`
}

// UsageInfo holds rate limit / usage data.
type UsageInfo struct {
	SessionPercent  float64 `json:"sessionPercent,omitempty"`
	SessionResetsAt string  `json:"sessionResetsAt,omitempty"`
	WeeklyPercent   float64 `json:"weeklyPercent,omitempty"`
	WeeklyResetsAt  string  `json:"weeklyResetsAt,omitempty"`
	CreditsUsed     float64 `json:"creditsUsed,omitempty"`
	CreditsLimit    float64 `json:"creditsLimit,omitempty"`
	Currency        string  `json:"currency,omitempty"`
}
