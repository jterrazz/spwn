package ports

import "strings"

type Listener struct {
	Port    int
	Bind    string
	Process string
	PID     int
}

func (l Listener) IPv6() bool { return strings.Contains(l.Bind, ":") }
