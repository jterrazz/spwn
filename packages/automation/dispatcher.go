package automation

import (
	"context"
	"sync"
	"time"
)

// Dispatcher is what the engine calls when an automation fires. The
// architect-side adapter (added in a later phase) implements this by
// auto-spawning the world if cold, then exec'ing the agent's runtime
// with the rendered prompt. The engine itself has no Docker / runtime
// awareness — Dispatch is the contract that hides all of that.
//
// Implementations must:
//   - Return nil on successful delivery (the runtime accepted the
//     prompt; whether the agent's reasoning then errors is not the
//     dispatcher's concern).
//   - Return non-nil on any failure — receipt OK is set from this
//     return value verbatim.
//   - Honour ctx cancellation as a "stop trying" signal; the engine
//     cancels its context on Stop.
type Dispatcher interface {
	Dispatch(ctx context.Context, req DispatchRequest) error
}

// DispatchRequest is the payload the engine hands to a Dispatcher
// for one fire.
type DispatchRequest struct {
	// World is the world key from spwn.yaml#worlds.<key>.
	World string
	// Agent is the agent name to wake (one of the world's agents).
	Agent string
	// Prompt is the rendered prompt body. Already templated against
	// the trigger event — the dispatcher just delivers bytes.
	Prompt string
	// Source carries trigger-kind metadata (cron / fs, on-time /
	// catchup, scheduled time, missed count). Receipts and any
	// dispatcher-side telemetry can read from here without re-deriving
	// it from the rendered prompt.
	Source FireSource
}

// FireSource describes WHY an automation fired. Shared by the engine,
// the receipt writer, and the prompt renderer (template variables
// like {{ .Missed }} pull from here).
type FireSource struct {
	// Kind is "cron" or "fs".
	Kind string
	// Reason is a short categorical label for the receipt — "on-time",
	// "catchup", "create:foo.md", etc. Free-form for fs paths so the
	// dashboard can group by "create" / "write" without reparsing.
	Reason string
	// Now is the wall time at which the fire was decided (engine's
	// view of clock.Now()). Stable across the render+dispatch flow.
	Now time.Time

	// Cron-only fields.
	Scheduled time.Time
	Missed    int
	LastFired time.Time

	// FS-only fields (Phase 3).
	EventPaths []string
	EventKind  string
}

// MockDispatcher is a test Dispatcher that records every call. The
// engine uses sync.Map for agent locks; this struct uses a plain
// mutex because tests typically inspect the slice synchronously after
// each Advance.
//
// Use NewMockDispatcher in tests; the zero value is also valid.
type MockDispatcher struct {
	mu       sync.Mutex
	requests []DispatchRequest
	// Err, when non-nil, makes Dispatch return that error every call.
	// Tests assertions for the "dispatch failed → receipt OK=false"
	// path set this.
	Err error
	// Hold blocks each Dispatch on this channel before returning. Lets
	// tests drive concurrency: send N values for the first N calls to
	// proceed serially, or close to release everything. Nil → no hold.
	Hold chan struct{}
}

// NewMockDispatcher constructs a MockDispatcher with no error and no
// hold. Provided for parity with the other test helpers.
func NewMockDispatcher() *MockDispatcher { return &MockDispatcher{} }

// Dispatch records the request on entry, optionally blocks on Hold,
// then returns d.Err. Recording before Hold lets tests observe "the
// engine has reached Dispatch but Dispatch has not yet returned" via
// Count() — which is what serialisation checks need (e.g. only one
// request in flight per agent).
func (d *MockDispatcher) Dispatch(ctx context.Context, req DispatchRequest) error {
	d.mu.Lock()
	d.requests = append(d.requests, req)
	d.mu.Unlock()
	if d.Hold != nil {
		select {
		case <-d.Hold:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return d.Err
}

// Requests returns a snapshot of every recorded Dispatch call in
// arrival order.
func (d *MockDispatcher) Requests() []DispatchRequest {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]DispatchRequest, len(d.requests))
	copy(out, d.requests)
	return out
}

// Count is a convenience for "how many fires so far".
func (d *MockDispatcher) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.requests)
}
