package settings

import "testing"

func TestClaudeTokenLine(t *testing.T) {
	cases := []struct {
		name   string
		export string
		want   string
		ok     bool
	}{
		{"spaces and dollar", `{"operator:claude:oauth-token":"a b $HOME"}`, `export CLAUDE_CODE_OAUTH_TOKEN='a b $HOME'`, true},
		{"other name only", `{"session:a":"x"}`, "", false},
		{"empty value", `{"operator:claude:oauth-token":""}`, "", false},
		{"not json", `error: no such credential`, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ClaudeTokenLine([]byte(c.export))
			if ok != c.ok || got != c.want {
				t.Fatalf("ClaudeTokenLine(%s) = %q, %v; want %q, %v", c.export, got, ok, c.want, c.ok)
			}
		})
	}
}
