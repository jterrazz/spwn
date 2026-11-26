package devices

import "github.com/jterrazz/studio-station/apps/engine/core/fail"

const (
	BadName         fail.Code = "devices.bad-name"
	BadKey          fail.Code = "devices.bad-key"
	BadChannels     fail.Code = "devices.bad-channels"
	Duplicate       fail.Code = "devices.duplicate"
	OwnKey          fail.Code = "devices.own-key"
	AddHereOnly     fail.Code = "devices.add-here-only"
	PresenceRefused fail.Code = "devices.presence-refused"
	NoPresence      fail.Code = "devices.no-presence"
	Unknown         fail.Code = "devices.unknown"
)

// value: the finding a book this Mac's engine did not write is, its words
// on every surface: what happened, what is trusted now, the gesture that
// repairs.
const (
	ChangedOutside = "The device book changed outside Spwn"
	ChangedWhy     = "a book or a keychain head the engine did not write or could not parse, or a write of it cut short; nothing of it is trusted, and no VM is given its keys"
	ChangedRepair  = "add each device again with station devices add, you at this Mac, taking its key from the device itself, never from this list"
)

// value: the finding a book the engine could not check yet is: the login
// keychain unreadable since it started; it ends on its own once the
// keychain answers.
const (
	NotChecked       = "The device book is not checked yet"
	NotCheckedWhy    = "the login keychain could not be read since the engine started; nothing of the book is trusted, and no VM is given its keys, until it can"
	NotCheckedRepair = "unlock this Mac's login keychain; the book is checked again on its own"
)

func Refusal(t fail.Type, code fail.Code, subject string) *fail.Error {
	return fail.New(t, code, subject, nil)
}
