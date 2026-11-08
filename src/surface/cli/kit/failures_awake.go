package kit

import (
	"errors"

	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"

	"github.com/jterrazz/studio-station/src/core/fail"
)

func awakeFailure(err error) (*fail.Error, bool) {
	var rng *awake.RangeError
	var off *awake.OffMacError
	var none *awake.UnsupportedError
	var guard *awake.GuardError
	var down *awake.EngineDownError
	var run *awake.RunError
	switch {
	case errors.As(err, &run):
		return failed(run.Error(), "station awake 4h", err), true
	case errors.As(err, &rng):
		if rng.Long {
			return usage(rng.Error(), "station awake 24h"), true
		}
		return usage(rng.Error(), "station awake 1h"), true
	case errors.As(err, &off):
		return usage(off.Error(), catalog.AwakeOnItsMac), true
	case errors.As(err, &none):
		return usage(none.Error(), "run it on a Mac"), true
	case errors.As(err, &guard):
		if guard.Guard == awake.ReasonLid {
			return failed(guard.Error(), "station awake 4h --lid --battery", err), true
		}
		return failed(guard.Error(), "station awake", err), true
	case errors.As(err, &down):
		if down.Installed {
			return failed(down.Error(), "station engine start", err), true
		}
		return failed(down.Error(), "station setup", err), true
	}
	return nil, false
}
