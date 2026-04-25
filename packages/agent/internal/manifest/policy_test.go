package manifest

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// agent.yaml supports two shapes for each dependencies entry: a
// scalar string ("spwn:x") for the no-policy case, or a mapping
// ({name: spwn:x, deny: [...], allow: [...]}) when a policy is
// declared. The unmarshaller must accept both, mixed in any order,
// and round-trip cleanly so `spwn agent inspect` doesn't lose the
// allow/deny lists.

func TestManifest_UnmarshalYAML_PoliciedDeps(t *testing.T) {
	src := `
name: marketer
dependencies:
  - spwn:unix
  - name: spwn:x
    deny: [post-tweet, reply-tweet]
  - name: spwn:notion
    allow: [search, fetch_page]
`
	var m Manifest
	if err := yaml.Unmarshal([]byte(src), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	wantDeps := []string{"spwn:unix", "spwn:x", "spwn:notion"}
	if strings.Join(m.Deps, ",") != strings.Join(wantDeps, ",") {
		t.Errorf("Deps = %v, want %v", m.Deps, wantDeps)
	}
	if x := m.DepPolicies["spwn:x"]; len(x.Deny) != 2 || x.Deny[0] != "post-tweet" {
		t.Errorf("spwn:x deny = %v", x.Deny)
	}
	if n := m.DepPolicies["spwn:notion"]; len(n.Allow) != 2 || n.Allow[1] != "fetch_page" {
		t.Errorf("spwn:notion allow = %v", n.Allow)
	}
	// Plain scalar entry should NOT have a policy entry.
	if _, ok := m.DepPolicies["spwn:unix"]; ok {
		t.Errorf("spwn:unix should have no policy")
	}
}

func TestManifest_UnmarshalYAML_MissingNameRejected(t *testing.T) {
	src := `
dependencies:
  - allow: [foo]
`
	var m Manifest
	err := yaml.Unmarshal([]byte(src), &m)
	if err == nil {
		t.Errorf("missing name in mapping entry: want error, got Deps=%v", m.Deps)
	}
}

func TestManifest_UnmarshalYAML_BadTypeRejected(t *testing.T) {
	// dependencies entry that's neither scalar nor mapping (here a
	// nested sequence).
	src := `
dependencies:
  - [nope, this, fails]
`
	var m Manifest
	err := yaml.Unmarshal([]byte(src), &m)
	if err == nil {
		t.Errorf("nested-seq entry: want error, got Deps=%v", m.Deps)
	}
}

func TestManifest_RoundtripPreservesPolicy(t *testing.T) {
	orig := Manifest{
		Name: "alfred",
		Deps: []string{"spwn:unix", "spwn:x"},
		DepPolicies: map[string]DepPolicy{
			"spwn:x": {Deny: []string{"post-tweet"}},
		},
	}
	out, err := yaml.Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}
	var back Manifest
	if err := yaml.Unmarshal(out, &back); err != nil {
		t.Fatalf("re-unmarshal: %v\nwire:\n%s", err, out)
	}
	if strings.Join(back.Deps, ",") != "spwn:unix,spwn:x" {
		t.Errorf("Deps lost: %v", back.Deps)
	}
	pol, ok := back.DepPolicies["spwn:x"]
	if !ok || len(pol.Deny) != 1 || pol.Deny[0] != "post-tweet" {
		t.Errorf("policy lost: ok=%v pol=%+v", ok, pol)
	}
}

func TestManifest_EmptyDepPolicy_NotMaterialized(t *testing.T) {
	// Empty {allow: [], deny: []} mapping — should still skip the
	// DepPolicies map entry (no filter to apply).
	src := `
dependencies:
  - name: spwn:x
`
	var m Manifest
	if err := yaml.Unmarshal([]byte(src), &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Deps) != 1 || m.Deps[0] != "spwn:x" {
		t.Errorf("Deps = %v", m.Deps)
	}
	if _, ok := m.DepPolicies["spwn:x"]; ok {
		t.Errorf("empty policy mapping should not populate DepPolicies")
	}
}
