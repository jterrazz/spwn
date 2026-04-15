package source

import (
	"strings"
	"testing"
)

func TestResolveRuntime_tableCases(t *testing.T) {
	type want struct {
		runtime string
		errSub  string // if non-empty, expect error containing this substring
	}
	cases := []struct {
		name     string
		src      *ProjectSource
		override string
		want     want
	}{
		{
			name:     "nil source falls back to claude-code",
			src:      nil,
			override: "",
			want:     want{runtime: "claude-code"},
		},
		{
			name:     "empty source falls back to claude-code",
			src:      &ProjectSource{},
			override: "",
			want:     want{runtime: "claude-code"},
		},
		{
			name: "single agent with @spwn/claude-code is canonicalised",
			src: &ProjectSource{Agents: []AgentSource{
				{Name: "neo", Config: AgentConfig{Runtime: RuntimeConfig{Backend: "@spwn/claude-code"}}},
			}},
			want: want{runtime: "claude-code"},
		},
		{
			name: "single agent with already-canonical claude-code",
			src: &ProjectSource{Agents: []AgentSource{
				{Name: "neo", Config: AgentConfig{Runtime: RuntimeConfig{Backend: "claude-code"}}},
			}},
			want: want{runtime: "claude-code"},
		},
		{
			name:     "override wins over agents",
			override: "claude-code",
			src: &ProjectSource{Agents: []AgentSource{
				{Name: "neo", Config: AgentConfig{Runtime: RuntimeConfig{Backend: "codex"}}},
			}},
			want: want{runtime: "claude-code"},
		},
		{
			name:     "override gets canonicalised",
			override: "@spwn/claude-code",
			src:      nil,
			want:     want{runtime: "claude-code"},
		},
		{
			name: "two agents, same runtime: picked",
			src: &ProjectSource{Agents: []AgentSource{
				{Name: "neo", Config: AgentConfig{Runtime: RuntimeConfig{Backend: "@spwn/claude-code"}}},
				{Name: "morpheus", Config: AgentConfig{Runtime: RuntimeConfig{Backend: "claude-code"}}},
			}},
			want: want{runtime: "claude-code"},
		},
		{
			name: "two agents, conflicting runtimes: error",
			src: &ProjectSource{Agents: []AgentSource{
				{Name: "neo", Config: AgentConfig{Runtime: RuntimeConfig{Backend: "@spwn/claude-code"}}},
				{Name: "morpheus", Config: AgentConfig{Runtime: RuntimeConfig{Backend: "@spwn/codex"}}},
			}},
			want: want{errSub: "conflicting runtimes"},
		},
		{
			name: "agent without runtime declared: ignored",
			src: &ProjectSource{Agents: []AgentSource{
				{Name: "neo", Config: AgentConfig{}},
				{Name: "morpheus", Config: AgentConfig{Runtime: RuntimeConfig{Backend: "@spwn/claude-code"}}},
			}},
			want: want{runtime: "claude-code"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveRuntime(tc.src, tc.override)
			if tc.want.errSub != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got %q", tc.want.errSub, got)
				}
				if !strings.Contains(err.Error(), tc.want.errSub) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.want.errSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want.runtime {
				t.Fatalf("got %q want %q", got, tc.want.runtime)
			}
		})
	}
}
