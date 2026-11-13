package contracttest

import (
	"context"
	"encoding/json"
	"maps"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
)

type SecretSourceSubject struct {
	Source contract.SecretSource
	// value: the secrets the other tool keeps, by name.
	Kept map[string]string
}

func SecretSource(t *testing.T, subject func(t *testing.T) SecretSourceSubject) {
	ctx := context.Background()

	t.Run("the secrets asked are exported at once, one JSON object of name to value", func(t *testing.T) {
		s := subject(t)
		names := []string{"session:a", "operator:claude:oauth-token"}
		data, err := s.Source.Export(ctx, names)
		if err != nil {
			t.Fatalf("Export: %v", err)
		}
		var got map[string]string
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Export answered %q: not a JSON object: %v", data, err)
		}
		want := map[string]string{}
		for _, n := range names {
			want[n] = s.Kept[n]
		}
		if !maps.Equal(got, want) {
			t.Fatalf("Export = %v; want %v", got, want)
		}
	})

	t.Run("a name the program does not keep fails the export", func(t *testing.T) {
		s := subject(t)
		if _, err := s.Source.Export(ctx, []string{"session:a", "session:never"}); err == nil {
			t.Fatal("Export of a name not kept succeeded")
		}
	})
}
