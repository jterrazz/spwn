package accounts_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
)

func TestTheTermRuleNamesTheLongestTerm(t *testing.T) {
	if accounts.LongestGrant != 7*24*time.Hour {
		t.Fatalf("the longest term is %s; the term rule says 7 days", accounts.LongestGrant)
	}
	if !strings.Contains(accounts.SentenceTermRule.Text(), "7 days") {
		t.Errorf("the term rule %q does not name 7 days", accounts.SentenceTermRule.Text())
	}
}

func TestTheRowFragmentsFillTheirName(t *testing.T) {
	if got := fmt.Sprintf(accounts.SentenceRowFromPassword.Text(), "GitHub deploy"); got != "from the password GitHub deploy" {
		t.Errorf("fragment = %q", got)
	}
}
