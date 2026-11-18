package menubar

import (
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/page"
)

func TestAPageThatCouldNotBeReadSaysItsCauseWithheld(t *testing.T) {
	p := Unread("menu:home", page.Context{}, errors.New("read refused: Authorization: Bearer hunter2 sent"))
	if p.Error == nil || strings.Contains(p.Error.Message, "hunter2") || p.Error.Message != "read refused: Authorization: Bearer withheld sent" {
		t.Errorf("the unread page says %+v, want the token withheld and the rest kept", p.Error)
	}
}
