package kit

import (
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

func setupFailure(err error) (*fail.Error, bool) {
	var mismatch *setup.NameMismatchError
	var bad *setup.BadNameError
	var step *setup.StepFailedError
	var different *setup.DifferentContentError
	var busy *setup.BusyError
	var exists *setup.DestinationExistsError
	var cross *setup.CrossDeviceError
	switch {
	case errors.As(err, &mismatch):
		return usage(mismatch.Error(), "run station init without --name"), true
	case errors.As(err, &bad):
		return usage(bad.Error(), "use lowercase letters, digits and -, 1 to 40 characters"), true
	case errors.Is(err, setup.ErrNeedsName):
		return usage(setup.ErrNeedsName.Error(), "station init --name <name>"), true
	case errors.Is(err, setup.ErrNameOnVM):
		return usage(setup.ErrNameOnVM.Error(), "run station init without --name"), true
	case errors.As(err, &step):
		hint := "fix it, then run station init again: it resumes"
		switch {
		case errors.As(err, &different):
			hint = "keep one of the two files, then run station init again"
		case errors.As(err, &busy):
			hint = "stop the persistent VMs and delete the ephemeral ones (station machines), then station init"
		case errors.As(err, &exists):
			hint = exists.Path + " already exists: move one of them by hand"
		case errors.As(err, &cross):
			hint = "station never copies: move it by hand, then station init"
		}
		return failed(step.Error(), hint, err), true
	}
	return nil, false
}
