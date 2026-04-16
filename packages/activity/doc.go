// Package activity provides a system-wide event log for spwn.
//
// Every meaningful lifecycle event (world spawned, agent joined,
// session ended, …) is appended as a JSONL record to
// ~/.spwn/activity.jsonl. Writes are best-effort: a failure to write
// must never block the caller.
//
// Use Log to append, Read (with ReadOpts for filtering) to query,
// and the Phrase* helpers to produce the human-readable summaries
// the web UI and `spwn logs` render.
package activity
