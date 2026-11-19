package engine

const (
	StateRunning    = "running"
	StateStopped    = "stopped"
	StateStale      = "stale"
	StateBroken     = "broken"
	GuestConnected  = "connected"
	GuestConnecting = "connecting"
	GuestAbsent     = "absent"
)

const GuestSession = "/usr/local/bin/station guest"
