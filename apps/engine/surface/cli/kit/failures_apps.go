package kit

import (
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

func appsFailure(err error) (*fail.Error, bool) {
	var irreversible *apps.NotReversibleError
	var manual *apps.ManualInstallError
	var noUpgrade *apps.NoUpgradePathError
	var appStore *apps.AppStoreError
	switch {
	case errors.As(err, &irreversible):
		return usage(irreversible.Error(), "station installs it but cannot remove it: remove it by hand"), true
	case errors.As(err, &manual):
		return usage(manual.Error(), manual.Hint), true
	case errors.As(err, &appStore):
		return usage(appStore.Error(), "remove it from /Applications"), true
	case errors.As(err, &noUpgrade):
		return usage(noUpgrade.Error(), "reinstall it by hand: station apps install "+noUpgrade.App), true
	}
	return nil, false
}
