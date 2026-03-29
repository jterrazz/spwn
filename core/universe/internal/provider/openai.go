package provider

// OpenAI implements the Provider port for OpenAI's API (stub for future use).
type OpenAI struct{}

// NewOpenAI creates a new OpenAI provider adapter.
func NewOpenAI() *OpenAI {
	return &OpenAI{}
}

// Name returns the provider identifier.
func (o *OpenAI) Name() string {
	return "openai"
}

// RequiredEnvVars returns the environment variables needed by this provider.
func (o *OpenAI) RequiredEnvVars() []string {
	return []string{"OPENAI_API_KEY"}
}
