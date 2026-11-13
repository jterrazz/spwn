package api_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/remote/api"
)

func TestAPeerBeforeTheNewWordsIsOlder(t *testing.T) {
	own := "v3.1.0-4-gabcdef0"
	for version, older := range map[string]bool{
		"":                  true,
		"v3.0.0":            true,
		"v2.0.0-250":        true,
		"v3.0.0-75-g2174a1": true,
		api.WordsSince:      false,
		"v3.1.2":            false,
		"v4.0.0":            false,
		"dev":               false,
		"2174a13d":          false,
		own:                 false,
	} {
		if got := api.Older(version, own); got != older {
			t.Fatalf("a peer at %q is older: %v, want %v", version, got, older)
		}
	}
	if api.Older("v3.0.0", "v3.0.0") {
		t.Fatal("a peer that runs this very build reads its words")
	}
}

func TestAnOlderPeerIsWrittenTheOlderWords(t *testing.T) {
	in := `{"Request":{"Lifetime":"persistent","Size":{"MemoryBytes":9007199254740993}},"Head":{"Kind":"computer"},"vms":[{"lifetime":"ephemeral","kind":"workspace"}],"detail":{"kind":"computer"},"name":"computer"}`
	want := `{"Head":{"Kind":"mac"},"Request":{"Lifetime":"permanent","Size":{"MemoryBytes":9007199254740993}},"detail":{"kind":"mac"},"name":"computer","vms":[{"kind":"workspace","lifetime":"throwaway"}]}`
	if got := string(api.OlderWords([]byte(in))); got != want {
		t.Fatalf("written\n%s\nwant\n%s", got, want)
	}
	for _, same := range []string{`{"Request":{"Lifetime":"always-on"}}`, `not json "persistent"`, `null`} {
		if got := string(api.OlderWords([]byte(same))); got != same {
			t.Fatalf("%s written %s", same, got)
		}
	}
}
