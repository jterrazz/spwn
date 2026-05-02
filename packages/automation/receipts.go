package automation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// receiptTimeFormat is the canonical timestamp format on disk. RFC3339
// with nanosecond precision keeps logs sortable as plain text (string
// comparison matches chronological order) and matches what the
// jterrazz-os occurrences.jsonl prior art emits.
const receiptTimeFormat = time.RFC3339Nano

// EngineVersion identifies the engine that wrote a receipt. Semver
// goes in the receipt schema (not git SHAs) so dashboards reading
// older logs can branch cleanly when fields are added or
// deprecated. Bump the minor when adding optional fields, the
// major when changing field meanings.
const EngineVersion = "spwn-automation/1"

// Receipt is one row of the runs.jsonl audit log. Schema-stable: the
// dashboard reads this format, and any addition should keep existing
// fields where they are so old logs continue to parse.
//
// Mirrors jterrazz-os' occurrences.jsonl shape (the launchd
// run-with-receipt.sh prior art) so tooling there can read spwn's
// log unchanged once the user migrates.
type Receipt struct {
	// World is the world key the automation belongs to.
	World string `json:"world"`
	// Automation is the automation name (the map key in spwn.yaml).
	Automation string `json:"automation"`
	// Agent is the target agent the engine woke. Saves the
	// dashboard a join through spwn.yaml when grouping by agent.
	Agent string `json:"agent"`
	// Trigger identifies the event source: "cron" or "fs".
	Trigger string `json:"trigger"`
	// RunID is a unique-per-fire opaque identifier. Lets dashboards
	// trace a single fire across structured-logger output and the
	// receipt log. Future correlation across multiple specs hit by
	// the same trigger event will reuse the same RunID; today each
	// fire gets its own. 16 hex chars from crypto/rand.
	RunID string `json:"run_id"`
	// EngineVersion is the schema generation that wrote this row.
	// Dashboards branch on this when the schema evolves.
	EngineVersion string `json:"engine_version"`
	// Scheduled is the cron grid slot this fire covers. For on-time
	// fires Scheduled ≈ Fired; for catch-up Scheduled is the missed
	// slot itself, Fired is when we got around to it.
	Scheduled time.Time `json:"scheduled,omitempty"`
	// Fired is when the engine started the dispatch.
	Fired time.Time `json:"fired"`
	// Finished is when Dispatch returned (success or error).
	Finished time.Time `json:"finished"`
	// DurationMS is Finished - Fired in milliseconds. Pre-computed so
	// dashboard queries don't reparse timestamps.
	DurationMS int64 `json:"duration_ms"`
	// OK is true iff Dispatch returned nil.
	OK bool `json:"ok"`
	// Reason is a short categorical label: "on-time" / "catchup" /
	// "create:foo.md" / etc. Free-form for fs to keep the path in.
	Reason string `json:"reason"`
	// Error is set when OK is false; the dispatcher's error string
	// verbatim.
	Error string `json:"error,omitempty"`
	// Missed is the count of catch-up slots collapsed into this fire.
	// Cron+catchup-collapse only.
	Missed int `json:"missed,omitempty"`
	// LastFired is the previous successful fire's scheduled time.
	// Used by the dashboard to render "ran 2h late, last good Sunday".
	LastFired time.Time `json:"last_fired,omitempty"`
	// EventPaths is the full path list a debounce burst coalesced
	// into one fire. Reason carries only the first basename ("create:
	// foo.md"); EventPaths preserves every path so downstream
	// queries ("which files did this fire process?") aren't lossy.
	// Only populated for fs triggers.
	EventPaths []string `json:"event_paths,omitempty"`
	// EventKind is the dominant op for an fs burst — create / write /
	// rename. Mirrors the kind embedded in Reason but exposed as a
	// separate field so dashboards filter without string parsing.
	EventKind string `json:"event_kind,omitempty"`
}

// MarshalJSON emits the receipt with proper omission of zero-valued
// time fields. The standard `omitempty` JSON tag does not recognise
// time.Time{} as empty (it's a struct, not a nil-able primitive), so
// the dashboard's grep-friendly "no missed key for on-time fires"
// promise needs a hand-rolled marshaller. Encoding via a flat map
// keeps the field set obvious at the call site.
//
// Times are normalised to UTC at emit time. Without this, two
// daemon processes in different host time zones would write rows
// whose RFC3339Nano-with-offset strings sort incorrectly under
// plain text comparison ("2026-01-01T01:00:00+05:00" sorts BEFORE
// "2026-01-01T00:00:00Z" even though the UTC instant is later).
func (r Receipt) MarshalJSON() ([]byte, error) {
	out := map[string]any{
		"world":          r.World,
		"automation":     r.Automation,
		"agent":          r.Agent,
		"trigger":        r.Trigger,
		"run_id":         r.RunID,
		"engine_version": r.EngineVersion,
		"fired":          r.Fired.UTC().Format(receiptTimeFormat),
		"finished":       r.Finished.UTC().Format(receiptTimeFormat),
		"duration_ms":    r.DurationMS,
		"ok":             r.OK,
		"reason":         r.Reason,
	}
	if !r.Scheduled.IsZero() {
		out["scheduled"] = r.Scheduled.UTC().Format(receiptTimeFormat)
	}
	if !r.LastFired.IsZero() {
		out["last_fired"] = r.LastFired.UTC().Format(receiptTimeFormat)
	}
	if r.Error != "" {
		out["error"] = r.Error
	}
	if r.Missed > 0 {
		out["missed"] = r.Missed
	}
	if len(r.EventPaths) > 0 {
		out["event_paths"] = r.EventPaths
	}
	if r.EventKind != "" {
		out["event_kind"] = r.EventKind
	}
	return json.Marshal(out)
}

// ReceiptWriter is the engine's append-only sink for receipts.
// Production = FileReceiptWriter (project-relative .spwn/runs.jsonl).
// Tests = MemoryReceiptWriter, which keeps everything in a slice for
// easy assertion.
type ReceiptWriter interface {
	Write(r Receipt) error
}

// FileReceiptWriter persists receipts as JSON Lines to a file. One
// row per Write call, atomic at the line level (line-buffered append
// behind a mutex), but NOT crash-safe — a SIGKILL mid-write may
// leave a torn final line. Acceptable: the next Write replaces
// nothing, the dashboard's parser tolerates a trailing partial
// line, and the only loss is one fire's metadata (the dispatch
// itself was either successful or visibly failed elsewhere).
type FileReceiptWriter struct {
	Path string
	mu   sync.Mutex
}

// NewFileReceiptWriter constructs a writer rooted at path. The file
// is created on first Write — early construction failures (e.g.
// directory missing) surface there, not at New time, because the
// engine boot path constructs receipts before knowing whether any
// will actually fire.
func NewFileReceiptWriter(path string) *FileReceiptWriter {
	return &FileReceiptWriter{Path: path}
}

// Write appends the receipt as one JSON Lines row. Creates parent
// directories on first call. Errors propagate up so the engine can
// log them, but a write error never aborts dispatch — receipts are
// observability, not correctness.
func (w *FileReceiptWriter) Write(r Receipt) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(w.Path), 0o755); err != nil {
		return fmt.Errorf("receipt dir: %w", err)
	}
	f, err := os.OpenFile(w.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open receipt log: %w", err)
	}
	defer f.Close()
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode receipt: %w", err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append receipt: %w", err)
	}
	return nil
}

// MemoryReceiptWriter is a test ReceiptWriter that keeps every
// receipt in a slice. Goroutine-safe.
type MemoryReceiptWriter struct {
	mu       sync.Mutex
	receipts []Receipt
}

// NewMemoryReceiptWriter constructs an empty writer.
func NewMemoryReceiptWriter() *MemoryReceiptWriter { return &MemoryReceiptWriter{} }

// Write appends the receipt to the in-memory slice.
func (w *MemoryReceiptWriter) Write(r Receipt) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.receipts = append(w.receipts, r)
	return nil
}

// Receipts returns a snapshot of the recorded receipts.
func (w *MemoryReceiptWriter) Receipts() []Receipt {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]Receipt, len(w.receipts))
	copy(out, w.receipts)
	return out
}
