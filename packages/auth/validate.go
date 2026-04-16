package auth

import "context"

// Validate checks if a credential is valid by making a test API
// call. Per-provider logic lives in anthropic.go, openai.go,
// google.go.
func Validate(ctx context.Context, cred *Credential) *ProviderStatus {
	if cred == nil {
		return &ProviderStatus{
			Connected: false,
			Error:     "no credentials",
		}
	}
	if cred.Type == CredTypeNone {
		return &ProviderStatus{
			Provider:  cred.Provider,
			Connected: false,
			CredType:  CredTypeNone,
			Source:    cred.Source,
			Error:     "no credentials configured",
		}
	}
	switch cred.Provider {
	case ProviderAnthropic:
		return validateAnthropic(ctx, cred)
	case ProviderOpenAI:
		return validateOpenAI(ctx, cred)
	case ProviderGoogle:
		return validateGoogle(ctx, cred)
	}
	return &ProviderStatus{Provider: cred.Provider, Error: "unknown provider"}
}

// ValidateAll checks every known provider.
func ValidateAll(ctx context.Context) []ProviderStatus {
	creds := ResolveAll()
	results := make([]ProviderStatus, 0, len(creds))
	for _, cred := range creds {
		status := Validate(ctx, cred)
		results = append(results, *status)
	}
	return results
}
