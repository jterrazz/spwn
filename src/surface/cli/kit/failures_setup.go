package kit

import (
	"context"
	"errors"

	model "github.com/jterrazz/studio-station/src/machines"

	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/setup"
)

func setupFailure(err error) (*fail.Error, bool) {
	var mismatch *setup.NameMismatchError
	var bad *setup.BadNameError
	var step *setup.StepFailedError
	var different *setup.DifferentContentError
	var busy *setup.BusyError
	var exists *setup.DestinationExistsError
	var cross *setup.CrossDeviceError
	var batch *setup.AppsFailedError
	switch {
	case errors.Is(err, context.Canceled):
		// invariant: the report already says what was interrupted; the exit
		// is the interrupt's and nothing more is printed.
		return fail.Interruption(), true
	case errors.As(err, &mismatch):
		return usage(mismatch.Error(), "run station setup without --name"), true
	case errors.As(err, &bad):
		return usage(bad.Error(), "use lowercase letters, digits and -, 1 to 40 characters"), true
	case errors.Is(err, setup.ErrNeedsName):
		return usage(setup.ErrNeedsName.Error(), "station setup --name <name>"), true
	case errors.Is(err, setup.ErrNameOnVM):
		return usage(setup.ErrNameOnVM.Error(), "run station setup without --name"), true
	case errors.As(err, &step):
		hint := "fix it, then run station setup again: it resumes"
		var underAgent *model.RunningUnderAgentError
		switch {
		case errors.As(err, &underAgent):
			hint = "stop the permanent VMs and delete the throwaway ones (station machines), then station setup"
		case errors.As(err, &different):
			hint = "keep one of the two files, then run station setup again"
		case errors.As(err, &busy):
			hint = "stop the permanent VMs and delete the throwaway ones (station machines), then station setup"
		case errors.As(err, &exists):
			hint = exists.Path + " already exists: move one of them by hand"
		case errors.As(err, &batch):
			hint = "fix the apps above, then run station setup again: it resumes"
		case errors.As(err, &cross):
			hint = "station never copies: move it by hand, then station setup"
		}
		return failed(step.Error(), hint, err), true
	}
	return nil, false
}
