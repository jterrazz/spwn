package provider

// Anthropic implements the Provider port for Anthropic's Claude API.
type Anthropic struct{}

// NewAnthropic creates a new Anthropic provider adapter.
func NewAnthropic() *Anthropic {
	return &Anthropic{}
}

// Name returns the provider identifier.
func (a *Anthropic) Name() string {
	return "anthropic"
}

// RequiredEnvVars returns the environment variables needed by this provider.
func (a *Anthropic) RequiredEnvVars() []string {
	return []string{"ANTHROPIC_API_KEY"}
}
