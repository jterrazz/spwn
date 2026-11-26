package molecule

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func TestAHintOfWordsHasNoArrowAndACommandFollowsOne(t *testing.T) {
	for _, tc := range []struct {
		hint page.Hint
		want string
	}{
		{page.Hint{Text: "move it away, then plan again"}, "move it away, then plan again"},
		{page.Hint{Command: "station project plan"}, token.Arrow + " station project plan"},
	} {
		if got := HintText(tc.hint); got != tc.want {
			t.Errorf("HintText(%+v) = %q, want %q", tc.hint, got, tc.want)
		}
	}
}
