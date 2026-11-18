package accounts_test

import (
	"fmt"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
)

func TestTheRowFragmentsFillTheirName(t *testing.T) {
	if got := fmt.Sprintf(accounts.SentenceRowFromPassword.Text(), "GitHub deploy"); got != "from the password GitHub deploy" {
		t.Errorf("fragment = %q", got)
	}
}
