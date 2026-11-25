package devices

import "github.com/jterrazz/studio-station/apps/engine/core/fail"

const (
	BadName     fail.Code = "devices.bad-name"
	BadKey      fail.Code = "devices.bad-key"
	BadChannels fail.Code = "devices.bad-channels"
	Duplicate   fail.Code = "devices.duplicate"
	OwnKey      fail.Code = "devices.own-key"
)

func Refusal(t fail.Type, code fail.Code, subject string) *fail.Error {
	return fail.New(t, code, subject, nil)
}
