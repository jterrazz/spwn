package provider

import (
	"testing"
)

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
