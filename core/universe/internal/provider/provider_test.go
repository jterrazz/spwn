package provider

import (
	"testing"
)

// --- Provider interface compliance ---

// Provider is the interface all providers must satisfy.
type Provider interface {
	Name() string
	RequiredEnvVars() []string
}

func allProviders() []Provider {
	return []Provider{
		NewAnthropic(),
		NewOpenAI(),
	}
}

func TestAllProviders_HaveValidNames(t *testing.T) {
	for _, p := range allProviders() {
		name := p.Name()
		if name == "" {
			t.Errorf("provider has empty name")
		}
		// Names should be lowercase identifiers
		for _, c := range name {
			if c >= 'A' && c <= 'Z' {
				t.Errorf("provider name %q contains uppercase characters", name)
				break
			}
			if c == ' ' {
				t.Errorf("provider name %q contains spaces", name)
				break
			}
		}
	}
}

func TestAllProviders_DeclareRequiredEnvVars(t *testing.T) {
	for _, p := range allProviders() {
		vars := p.RequiredEnvVars()
		if len(vars) == 0 {
			t.Errorf("provider %q declares no required env vars", p.Name())
		}
		for _, v := range vars {
			if v == "" {
				t.Errorf("provider %q has empty env var name", p.Name())
			}
		}
	}
}

func TestAllProviders_UniqueNames(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range allProviders() {
		name := p.Name()
		if seen[name] {
			t.Errorf("duplicate provider name: %q", name)
		}
		seen[name] = true
	}
}

// --- Anthropic provider ---

func TestAnthropicName(t *testing.T) {
	a := NewAnthropic()
	if a.Name() != "anthropic" {
		t.Errorf("Name() = %q, want %q", a.Name(), "anthropic")
	}
}

func TestAnthropicRequiredEnvVars(t *testing.T) {
	a := NewAnthropic()
	vars := a.RequiredEnvVars()
	if len(vars) != 1 || vars[0] != "ANTHROPIC_API_KEY" {
		t.Errorf("RequiredEnvVars() = %v, want [ANTHROPIC_API_KEY]", vars)
	}
}

func TestAnthropicEnvVar_Format(t *testing.T) {
	a := NewAnthropic()
	for _, v := range a.RequiredEnvVars() {
		// Env vars should be UPPER_SNAKE_CASE
		for _, c := range v {
			if c >= 'a' && c <= 'z' {
				t.Errorf("env var %q should be uppercase", v)
				break
			}
		}
	}
}

// --- OpenAI provider ---

func TestOpenAIName(t *testing.T) {
	o := NewOpenAI()
	if o.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", o.Name(), "openai")
	}
}

func TestOpenAIRequiredEnvVars(t *testing.T) {
	o := NewOpenAI()
	vars := o.RequiredEnvVars()
	if len(vars) != 1 || vars[0] != "OPENAI_API_KEY" {
		t.Errorf("RequiredEnvVars() = %v, want [OPENAI_API_KEY]", vars)
	}
}

func TestOpenAIEnvVar_Format(t *testing.T) {
	o := NewOpenAI()
	for _, v := range o.RequiredEnvVars() {
		for _, c := range v {
			if c >= 'a' && c <= 'z' {
				t.Errorf("env var %q should be uppercase", v)
				break
			}
		}
	}
}

// --- Provider lookup ---

func TestProviderLookupByName(t *testing.T) {
	registry := map[string]Provider{}
	for _, p := range allProviders() {
		registry[p.Name()] = p
	}

	tests := []struct {
		name    string
		wantOK  bool
	}{
		{"anthropic", true},
		{"openai", true},
		{"unknown", false},
		{"", false},
		{"ANTHROPIC", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := registry[tt.name]
			if ok != tt.wantOK {
				t.Errorf("lookup(%q) = %v, want %v", tt.name, ok, tt.wantOK)
			}
		})
	}
}
