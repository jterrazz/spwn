package accounts_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
)

func TestCheckTermHoldsTheSpanAndDefaultsZero(t *testing.T) {
	for term, want := range map[time.Duration]bool{
		0:                          true,
		accounts.ShortestGrant:     true,
		accounts.LongestGrant:      true,
		accounts.ShortestGrant - 1: false,
		accounts.LongestGrant + 1:  false,
		-time.Hour:                 false,
		31 * 365 * 24 * time.Hour:  false,
	} {
		got, ok := accounts.CheckTerm(term)
		if ok != want {
			t.Fatalf("CheckTerm(%s) ok = %v; want %v", term, ok, want)
		}
		if term == 0 && got != accounts.DefaultGrantTerm {
			t.Fatalf("CheckTerm(0) = %s; want the default %s", got, accounts.DefaultGrantTerm)
		}
	}
}
