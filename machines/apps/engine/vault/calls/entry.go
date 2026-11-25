package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func EntryNamed(ctx context.Context, calls call.Router, mac machine.Ref, word string) (accounts.VaultEntry, error) {
	// value: the password of mac's vault a word names, by its label or its
	// id; a word that names none is the vault's own refusal.
	read, err := call.Read(ctx, calls, mac, VaultList, VaultListIn{}, nil)
	if err != nil {
		return accounts.VaultEntry{}, err
	}
	for _, e := range read.Entries {
		if e.Label == word || e.ID == word {
			return e, nil
		}
	}
	return accounts.VaultEntry{}, accounts.Refusal(fail.NotFound, accounts.VaultUnknown, word)
}
