package engine

import (
	"context"
	"time"
)

// value: the stream's frame of the moving numbers, one every 2 s while a
// watch of pages lists live keys.
const EventSample = "sample"

const SampleEvery = 2 * time.Second

type Sample struct {
	// value: each live key's value, as the page that lists the key draws it.
	Values map[string]any `json:"values,omitempty"`
	// value: the keys whose machine has not answered for longer than its
	// freshness: their value is the last known, marked stale.
	Stale []string `json:"stale,omitempty"`
}

// value: one tick's reading of the live keys the open pages list; it never
// waits on a machine that is late, whose last value it keeps.
type Sampler func(ctx context.Context, keys []string) Sample
