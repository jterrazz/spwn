package awake

import (
	"errors"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

const (
	Shortest = time.Minute
	Longest  = 24 * time.Hour
)

var (
	ErrOutOfBounds = errors.New("not between 1 min and 24 h")
	ErrNoLid       = errors.New("has no lid")
	ErrNoBattery   = errors.New("has no battery")
)

func For(words string, now time.Time, zone *time.Location) (time.Duration, error) {
	// invariant: a term is a duration (30m, 4h) or a clock time to stay
	// awake until (until 09:00), the next one to come; either lasts from 1
	// min to 24 h.
	if zone == nil {
		zone = time.UTC
	}
	clock, until := strings.CutPrefix(strings.TrimSpace(words), "until ")
	if until {
		at, err := time.ParseInLocation("15:04", strings.TrimSpace(clock), zone)
		if err != nil {
			return 0, ErrOutOfBounds
		}
		local := now.In(zone)
		end := time.Date(local.Year(), local.Month(), local.Day(), at.Hour(), at.Minute(), 0, 0, zone)
		if !end.After(local) {
			end = end.AddDate(0, 0, 1)
		}
		return Bounded(end.Sub(now))
	}
	d, err := time.ParseDuration(words)
	if err != nil {
		return 0, ErrOutOfBounds
	}
	return Bounded(d)
}

type Choice struct {
	Label string
	// value: the term as a length, in nanoseconds; empty when Until says it.
	For time.Duration `json:",omitempty"`
	// value: the term as the moment it ends, RFC 3339; empty when For says it.
	Until string `json:",omitempty"`
}

func UntilOffered(now time.Time, zone *time.Location) time.Time {
	// value: the end Until offers: the next 18:00 of the Mac's zone,
	// tomorrow's once today's has passed.
	if zone == nil {
		zone = time.UTC
	}
	local := now.In(zone)
	end := time.Date(local.Year(), local.Month(), local.Day(), 18, 0, 0, 0, zone)
	if !end.After(local) {
		end = end.AddDate(0, 0, 1)
	}
	return end
}

func Choices(now time.Time, zone *time.Location) []Choice {
	// invariant: the terms a page offers are the ones awake.keep takes, each
	// as keep would end it: a length, or the end UntilOffered gives.
	end := UntilOffered(now, zone)
	return []Choice{
		{Label: "1 hr", For: time.Hour},
		{Label: "4 hr", For: 4 * time.Hour},
		{Label: "Until " + end.Format("15:04"), Until: end.Format(time.RFC3339)},
	}
}

func Bounded(d time.Duration) (time.Duration, error) {
	if d < Shortest || d > Longest {
		return 0, ErrOutOfBounds
	}
	return d, nil
}

func Options(c machine.Capabilities, lid, battery bool) error {
	// invariant: --lid needs a lid, --battery a battery, as the Mac reports
	// them, never by its model.
	switch {
	case lid && !c.Lid:
		return ErrNoLid
	case battery && !c.Battery:
		return ErrNoBattery
	}
	return nil
}
