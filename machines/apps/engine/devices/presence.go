package devices

import (
	"context"
	"slices"
	"strings"
)

type Answer int

const (
	// value: how a Mac's presence check answered: the owner confirmed,
	// declined, or this Mac had no way to ask.
	Given Answer = iota
	Refused
	Absent
)

type Presence func(ctx context.Context, reason string) Answer

func Reason(d Device) string {
	// value: the sentence the presence sheet shows: the device by its name
	// and its key's fingerprint, and what it may open.
	channels := make([]string, 0, len(d.Channels))
	for _, c := range d.Channels {
		channels = append(channels, string(c))
	}
	slices.Sort(channels)
	return "let " + d.Name + " (" + Fingerprint(d.Key) + ") reach this Mac by " + strings.Join(channels, ", ")
}
