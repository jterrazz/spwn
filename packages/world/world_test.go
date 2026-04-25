package world

import "testing"

func TestAutoWorkspaceName(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		index int
		want  string
	}{
		{name: "absolute slug basename", path: "/host/myproject", index: 0, want: "myproject"},
		{name: "lowercased", path: "/host/MyProject", index: 0, want: "myproject"},
		{name: "kebab survives", path: "/host/my-app", index: 2, want: "my-app"},
		{name: "non-slug → fallback", path: "/host/My Project", index: 0, want: "workspace0"},
		{name: "leading digit → fallback", path: "/host/123-app", index: 1, want: "workspace1"},
		{name: "empty basename → fallback", path: "/", index: 0, want: "workspace0"},
		{name: "underscore not slug → fallback", path: "/host/my_app", index: 3, want: "workspace3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AutoWorkspaceName(tt.path, tt.index); got != tt.want {
				t.Errorf("AutoWorkspaceName(%q, %d) = %q, want %q", tt.path, tt.index, got, tt.want)
			}
		})
	}
}
