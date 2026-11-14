package accountscmd

import (
	"errors"
	"testing"

	pageaccounts "github.com/jterrazz/studio-station/apps/engine/pages/accounts"
)

func TestAMacWhoseAccountsWereNotReadSaysWhyWithItsTokenWithheld(t *testing.T) {
	b := faulted(pageaccounts.MachineBlock{}, errors.New("fetch https://bot:hunter2@proxy.example/x refused"))
	if b.Note != "fetch https://withheld@proxy.example/x refused" {
		t.Errorf("note = %q; want its credential withheld", b.Note)
	}
}
