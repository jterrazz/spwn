package auth

// Resolve returns the single credential spwn should use for a provider,
// honouring the user's auth.yaml preferences (Disabled, ActiveMethod).
// When no credentials are detected or the provider is disabled, returns
// a Credential with Type=CredTypeNone so callers can branch without a
// nil check.
func Resolve(p Provider) *Credential {
	if IsProviderDisabled(p) {
		return noneCredential(p, "disabled")
	}
	detected := DetectMethods(p)
	return pickByPref(p, detected)
}

// DetectMethods returns every detected credential for a provider, in
// the provider's discovery order. Unlike Resolve, it does not honour
// the Disabled flag or ActiveMethod — callers use this to show the
// user every credential spwn could pick from.
func DetectMethods(p Provider) []*Credential {
	switch p {
	case ProviderAnthropic:
		return detectAnthropic()
	case ProviderOpenAI:
		return detectOpenAI()
	case ProviderGoogle:
		return detectGoogle()
	}
	return nil
}

// ResolveAll returns credentials for every supported provider. Google
// is intentionally excluded — it has no runtime wired up and keeping
// it here would pollute the dashboard and credential sync.
func ResolveAll() map[Provider]*Credential {
	result := make(map[Provider]*Credential)
	for _, p := range []Provider{ProviderAnthropic, ProviderOpenAI} {
		result[p] = Resolve(p)
	}
	return result
}

// pickByPref selects one credential from a detection list, applying
// the user's ActiveMethod preference if any. Falls back to the first
// detected credential when no preference is set, or when the preferred
// method wasn't found. Returns a Type=CredTypeNone sentinel when the
// list is empty so callers never see nil.
//
// Provider-level Disabled is intentionally NOT checked here — Resolve
// handles that short-circuit before calling pickByPref so DetectMethods
// callers (the dashboard) still get the full list to render.
func pickByPref(p Provider, detected []*Credential) *Credential {
	if len(detected) == 0 {
		return noneCredential(p, "not configured")
	}

	if preferred := ActiveMethod(p); preferred != "" {
		for _, cred := range detected {
			if cred.Method() == preferred {
				return cred
			}
		}
		// User asked for a method spwn couldn't find. Fall through to
		// discovery order rather than failing hard — the dashboard will
		// surface the mismatch more usefully than a runtime error.
	}

	return detected[0]
}

// noneCredential is the Type=CredTypeNone placeholder we return
// instead of nil so callers don't have to nil-check. Source carries
// a human-readable hint ("not configured" vs "disabled") that the
// dashboard renders verbatim.
func noneCredential(p Provider, source string) *Credential {
	return &Credential{
		Provider: p,
		Type:     CredTypeNone,
		Source:   source,
	}
}
