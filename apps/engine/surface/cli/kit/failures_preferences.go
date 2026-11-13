package kit

import (
	"errors"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
)

const preferencesFile = "preferences.json"

func preferencesFailure(err error) (*fail.Error, bool) {
	var unreadable *store.UnreadableError
	var unwritable *store.UnwritableError
	var vault *accounts.VaultError
	switch {
	case errors.As(err, &unreadable) && strings.HasSuffix(unreadable.Path, "/"+preferencesFile):
		// invariant: a preferences file that does not read stops the command and
		// is never rewritten, so the owner's hand edit is never lost.
		where := unreadable.Cause.Error()
		if unreadable.Line > 0 {
			where = "line " + strconv.Itoa(unreadable.Line) + ": " + where
		}
		return failed(unreadable.Path+" cannot be read: "+where, "fix that line: station never rewrites a file it cannot read", err), true
	case errors.As(err, &unreadable):
		return failed("cannot read \""+unreadable.Path+"\": "+unreadable.Cause.Error(), "repair the JSON of \""+unreadable.Path+"\", or move it away to start afresh", err), true
	case errors.As(err, &unwritable):
		return failed("cannot write \""+unwritable.Path+"\": "+unwritable.Cause.Error(), "check the permissions of \""+unwritable.Path+"\"", err), true
	case errors.Is(err, accounts.ErrNoVault):
		return usage(accounts.ErrNoVault.Error(), "run it on its Mac: station <command> --on <mac>"), true
	case errors.As(err, &vault):
		return failed("keychain account \""+vault.Name+"\": "+vault.Cause.Error(), "open Keychain Access and check the station items", err), true
	}
	return nil, false
}
