package darwin

type Presence int

const (
	PresenceGiven Presence = iota
	PresenceRefused
	PresenceAbsent
)
