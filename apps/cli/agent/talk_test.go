package agent

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatExecError_AuthenticationError(t *testing.T) {
	err := fmt.Errorf("exit status 1")
	output := []byte("Error: authentication_error - invalid credentials")
	result := formatExecError(err, output)

	if !strings.Contains(result.Error(), "spwn auth login") {
		t.Errorf("expected mention of 'spwn auth login', got: %s", result.Error())
	}
	if !strings.Contains(result.Error(), "authentication failed") {
		t.Errorf("expected mention of 'authentication failed', got: %s", result.Error())
	}
}

func TestFormatExecError_OAuthExpired(t *testing.T) {
	err := fmt.Errorf("exit status 1")
	output := []byte("OAuth token has expired, please refresh")
	result := formatExecError(err, output)

	if !strings.Contains(result.Error(), "refresh") {
		t.Errorf("expected mention of 'refresh', got: %s", result.Error())
	}
	if !strings.Contains(result.Error(), "expired") {
		t.Errorf("expected mention of 'expired', got: %s", result.Error())
	}
}

func TestFormatExecError_InvalidAPIKey(t *testing.T) {
	err := fmt.Errorf("exit status 1")
	output := []byte("Invalid API key provided")
	result := formatExecError(err, output)

	if !strings.Contains(result.Error(), "spwn auth") {
		t.Errorf("expected mention of 'spwn auth', got: %s", result.Error())
	}
	if !strings.Contains(result.Error(), "invalid API key") {
		t.Errorf("expected mention of 'invalid API key', got: %s", result.Error())
	}
}

func TestFormatExecError_InvalidXAPIKey(t *testing.T) {
	err := fmt.Errorf("exit status 1")
	output := []byte("invalid x-api-key header")
	result := formatExecError(err, output)

	if !strings.Contains(result.Error(), "spwn auth") {
		t.Errorf("expected mention of 'spwn auth', got: %s", result.Error())
	}
}

func TestFormatExecError_GenericWithOutput(t *testing.T) {
	err := fmt.Errorf("exit status 1")
	output := []byte("some unexpected error occurred")
	result := formatExecError(err, output)

	if !strings.Contains(result.Error(), "some unexpected error occurred") {
		t.Errorf("expected output in error message, got: %s", result.Error())
	}
	if !strings.Contains(result.Error(), "spwn auth check") {
		t.Errorf("expected hint about 'spwn auth check', got: %s", result.Error())
	}
}

func TestFormatExecError_GenericTruncatesLongOutput(t *testing.T) {
	err := fmt.Errorf("exit status 1")
	output := []byte(strings.Repeat("x", 600))
	result := formatExecError(err, output)

	if !strings.Contains(result.Error(), "...") {
		t.Errorf("expected truncated output with '...', got length: %d", len(result.Error()))
	}
}

func TestFormatExecError_EmptyOutput(t *testing.T) {
	err := fmt.Errorf("exit status 1")
	result := formatExecError(err, nil)

	if !strings.Contains(result.Error(), "spwn auth check") {
		t.Errorf("expected hint about 'spwn auth check', got: %s", result.Error())
	}
}

func TestFormatExecError_EmptyByteOutput(t *testing.T) {
	err := fmt.Errorf("exit status 1")
	result := formatExecError(err, []byte(""))

	if !strings.Contains(result.Error(), "spwn auth check") {
		t.Errorf("expected hint about 'spwn auth check', got: %s", result.Error())
	}
}

func TestFormatExecError_NetworkError(t *testing.T) {
	err := fmt.Errorf("exit status 1")
	output := []byte("Could not resolve host api.anthropic.com")
	result := formatExecError(err, output)

	if !strings.Contains(result.Error(), "network error") {
		t.Errorf("expected 'network error', got: %s", result.Error())
	}
}

func TestFormatExecError_RateLimit(t *testing.T) {
	err := fmt.Errorf("exit status 1")
	output := []byte("rate_limit exceeded, try again later")
	result := formatExecError(err, output)

	if !strings.Contains(result.Error(), "rate limited") {
		t.Errorf("expected 'rate limited', got: %s", result.Error())
	}
}
