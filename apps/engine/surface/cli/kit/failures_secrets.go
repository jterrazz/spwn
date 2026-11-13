package kit

import (
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/accounts"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

func secretsFailure(err error) (*fail.Error, bool) {
	var notVM *accounts.NotAVMError
	var offMac *accounts.OffMacError
	var noStudio *accounts.NoStudioError
	var step *accounts.StepError
	switch {
	case errors.As(err, &notVM):
		return usage(notVM.Error(), "station machines"), true
	case errors.As(err, &offMac):
		return usage(offMac.Error(), "station accounts share <account> --with <vm> --on <mac>"), true
	case errors.As(err, &noStudio):
		return failed(noStudio.Error(), "see docs/reference/accounts.md#studio-in-a-vm", err), true
	case errors.As(err, &step):
		return failed(step.Error(), "check the VM: station machines "+step.VM, err), true
	}
	return nil, false
}
