package atom

import "testing"

func TestMiddleKeepsAWideNameWithinItsWidth(t *testing.T) {
	for _, s := range []string{
		"日本語日本語日本語日本語",
		"🚀🚀🚀🚀🚀🚀🚀🚀🚀🚀",
		"~/Library/日本語/アプリケーション/設定.plist",
		"a日b本c語d日e本f語",
	} {
		for width := 3; width <= Width(s); width++ {
			if got := Middle(s, width); Width(got) > width {
				t.Errorf("Middle(%q, %d) = %q is %d cells", s, width, got, Width(got))
			}
			if got := path(s, width); Width(got) > width {
				t.Errorf("path(%q, %d) = %q is %d cells", s, width, got, Width(got))
			}
		}
	}
}

func TestMiddleOfAsciiIsUnchanged(t *testing.T) {
	if got, want := Middle("abcdefghij", 7), "abc…hij"; got != want {
		t.Fatalf("Middle = %q, want %q", got, want)
	}
	if got, want := path("~/Library/LaunchAgents/watchman.plist", 16), "~/Libra…an.plist"; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestMiddleOfWideCharactersCountsCells(t *testing.T) {
	if got, want := Middle("日本語日本語日本語", 7), "日…語"; got != want {
		t.Fatalf("Middle = %q, want %q", got, want)
	}
}
