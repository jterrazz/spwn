package dockerfile

import (
	"strings"
	"testing"
)

// The Dockerfile generator turns a per-tool Policy into a RUN line
// that materialises /etc/spwn/policy/<short>.json at image build
// time. Catalog tool wrappers read this file via spwn-policy-check
// to enforce per-agent allow/deny.
//
// Tests pin: (1) policy emission is gated on non-empty allow OR
// deny, (2) the emitted JSON is valid + ordered, (3) shell escaping
// handles single quotes in tool method names (X has none today,
// LinkedIn might tomorrow), (4) absent policy = no extra RUN.

func TestGenerate_EmitsPolicyJSONWhenSet(t *testing.T) {
	tools := []ToolInput{{
		Name:     "spwn:x",
		Commands: []string{"echo install x"},
		Policy: &Policy{
			Short: "x",
			Deny:  []string{"post-tweet", "reply-tweet"},
		},
	}}
	out := string(Generate([]byte("FROM scratch\n"), tools, ""))
	if !strings.Contains(out, "/etc/spwn/policy/x.json") {
		t.Errorf("missing policy file path in dockerfile:\n%s", out)
	}
	if !strings.Contains(out, `"deny":["post-tweet","reply-tweet"]`) {
		t.Errorf("missing/wrong deny JSON in dockerfile:\n%s", out)
	}
}

func TestGenerate_OmitsPolicyWhenEmpty(t *testing.T) {
	tools := []ToolInput{{
		Name:     "spwn:x",
		Commands: []string{"echo install x"},
		Policy:   &Policy{Short: "x"},
	}}
	out := string(Generate([]byte("FROM scratch\n"), tools, ""))
	if strings.Contains(out, "/etc/spwn/policy/") {
		t.Errorf("empty policy should not emit RUN:\n%s", out)
	}
}

func TestGenerate_OmitsPolicyWhenNil(t *testing.T) {
	tools := []ToolInput{{Name: "spwn:x", Commands: []string{"echo x"}}}
	out := string(Generate([]byte("FROM scratch\n"), tools, ""))
	if strings.Contains(out, "/etc/spwn/policy/") {
		t.Errorf("nil policy should not emit RUN:\n%s", out)
	}
}

func TestGenerate_EscapesSingleQuotesInPolicy(t *testing.T) {
	tools := []ToolInput{{
		Name:     "spwn:future",
		Commands: []string{"echo x"},
		Policy: &Policy{
			Short: "future",
			Deny:  []string{"a'b"}, // unrealistic but tests the escape path
		},
	}}
	out := string(Generate([]byte("FROM scratch\n"), tools, ""))
	// The shell escape produces '"'"' in place of '
	// We just verify no naked single quote inside our value broke
	// the surrounding shell quoting.
	if strings.Contains(out, `'a'b'`) {
		t.Errorf("naked single quote in dockerfile breaks shell quoting:\n%s", out)
	}
	if !strings.Contains(out, `'"'"'`) {
		t.Errorf("expected escape sequence not found:\n%s", out)
	}
}

func TestGenerate_PolicyEmittedAfterToolCommands(t *testing.T) {
	tools := []ToolInput{{
		Name:     "spwn:x",
		Commands: []string{"INSTALL_X_HERE"},
		Policy:   &Policy{Short: "x", Deny: []string{"post-tweet"}},
	}}
	out := string(Generate([]byte("FROM scratch\n"), tools, ""))
	idxInstall := strings.Index(out, "INSTALL_X_HERE")
	idxPolicy := strings.Index(out, "/etc/spwn/policy/x.json")
	if idxInstall == -1 || idxPolicy == -1 {
		t.Fatalf("missing markers in dockerfile:\n%s", out)
	}
	if idxPolicy < idxInstall {
		t.Errorf("policy RUN appeared before tool install — wrappers won't have spwn-policy-check yet")
	}
}
