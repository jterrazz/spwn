//go:build !darwin || !cgo

package darwin

import "time"

func AskPresence(string, time.Duration) Presence { return PresenceAbsent }
