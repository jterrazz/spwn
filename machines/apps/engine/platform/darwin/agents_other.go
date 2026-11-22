//go:build !darwin || !cgo

package darwin

import "errors"

type Agents struct {
	// value: a build without cgo, or off a Mac, has no SMAppService: its
	// engine runs only as a bare job.
}

var errNoAppAgents = errors.New("SMAppService needs the release's darwin build, with cgo")

func (Agents) Register(string) error { return errNoAppAgents }

func (Agents) Unregister(string) error { return errNoAppAgents }

func (Agents) Registered(string) bool { return false }

func (Agents) ShowLoginItem(string) error { return errNoAppAgents }
