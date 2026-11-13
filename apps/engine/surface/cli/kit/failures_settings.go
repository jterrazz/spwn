package kit

import (
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/words"
)

func settingsFailure(err error) (*fail.Error, bool) {
	var unknown *settings.UnknownPackageError
	var root *settings.NeedsRootError
	var livesOn *settings.WrongLivesOnError
	var irreversible *settings.NotReversibleError
	var terminal *settings.NeedsTerminalError
	var studio *settings.StudioMissingError
	switch {
	case errors.As(err, &unknown):
		return usage(unknown.Error(), "station settings"), true
	case errors.As(err, &root):
		verb := "apply"
		if root.Verb == "uninstall" {
			verb = "off"
		}
		return failed(root.Error(), words.SettingsAsRoot(verb, root.Package), err), true
	case errors.As(err, &livesOn):
		return usage(livesOn.Error(), "station settings apply "+livesOn.Package+" --on this-mac"), true
	case errors.As(err, &irreversible):
		return usage(irreversible.Error(), "station applies it but cannot undo it: undo it by hand"), true
	case errors.As(err, &terminal):
		return Translate(fail.New(fail.NeedsTerminal, words.NeedsTerminal, terminal.Package, nil), ""), true
	case errors.As(err, &studio):
		return failed(studio.Error(), "see docs/reference/accounts.md#claude-in-a-vm", err), true
	}
	return nil, false
}
