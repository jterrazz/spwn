package auth

// Resolve finds the best available credential for a provider.
// The per-provider logic lives in anthropic.go, openai.go, google.go.
func Resolve(p Provider) *Credential {
	switch p {
	case ProviderAnthropic:
		return resolveAnthropic()
	case ProviderOpenAI:
		return resolveOpenAI()
	case ProviderGoogle:
		return resolveGoogle()
	}
	return nil
}

// ResolveAll returns credentials for every known provider.
func ResolveAll() map[Provider]*Credential {
	result := make(map[Provider]*Credential)
	for _, p := range []Provider{ProviderAnthropic, ProviderOpenAI} {
		result[p] = Resolve(p)
	}
	return result
}
