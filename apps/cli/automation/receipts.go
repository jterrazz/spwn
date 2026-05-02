package automation

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

// receiptsPath is the canonical location of the project-scoped
// runs.jsonl. Mirrors what the engine factory configures in the
// architect's NewAutomationEngine — keeping the constant here
// (rather than re-importing from packages/automation) means the CLI
// package doesn't drag fsnotify + cron deps into the install
// graph for users running `spwn automation ls` on a tiny project.
func receiptsPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".spwn", "runs.jsonl")
}

// receiptRow is the shape we read off disk. Mirrors the on-wire JSON
// from packages/automation/receipts.go's MarshalJSON. We don't import
// the engine's Receipt type because:
//
//   - The CLI runs without the engine in-process; coupling its
//     binary size to fsnotify/cron is unnecessary.
//   - On-disk schema stability is the contract; the CLI is just a
//     reader, and an explicit local struct with json tags makes that
//     contract visible.
//
// Forward-compat: every new optional field carries omitempty here
// AND in the engine's Receipt. A CLI from yesterday reading a row
// written by today's engine simply ignores unknown fields (json
// unmarshal default behaviour); a CLI from tomorrow reading
// yesterday's row sees zero values for the new fields, which is
// what the renderer must tolerate.
type receiptRow struct {
	World         string    `json:"world"`
	Automation    string    `json:"automation"`
	Agent         string    `json:"agent,omitempty"`
	Trigger       string    `json:"trigger"`
	RunID         string    `json:"run_id,omitempty"`
	EngineVersion string    `json:"engine_version,omitempty"`
	Scheduled     time.Time `json:"scheduled,omitempty"`
	Fired         time.Time `json:"fired"`
	Finished      time.Time `json:"finished"`
	DurationMS    int64     `json:"duration_ms"`
	OK            bool      `json:"ok"`
	Reason        string    `json:"reason"`
	Error         string    `json:"error,omitempty"`
	Missed        int       `json:"missed,omitempty"`
	LastFired     time.Time `json:"last_fired,omitempty"`
	EventPaths    []string  `json:"event_paths,omitempty"`
	EventKind     string    `json:"event_kind,omitempty"`
}

// readReceipts decodes every row in the receipts log. Bad lines are
// skipped silently — a torn final line from a SIGKILL mid-write is a
// known acceptable loss documented at the engine's
// FileReceiptWriter.
func readReceipts(projectRoot string) ([]receiptRow, error) {
	path := receiptsPath(projectRoot)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	return decodeReceipts(f)
}

// maxReceiptLineBytes caps how large a single receipt line we'll
// json-decode. Lines longer than this are skipped (same contract as
// bad-JSON lines). A 4MB cap covers a panic stack trace plus padding;
// pathological dispatcher errors that produce 100MB lines are best
// dropped from the dashboard view rather than blocking the whole log.
const maxReceiptLineBytes = 4 * 1024 * 1024

// decodeReceipts is split out so tests can pass an in-memory reader
// without touching the filesystem.
//
// We use bufio.Reader.ReadBytes('\n') rather than bufio.Scanner
// because Scanner aborts the entire read on a token-size violation
// (returning bufio.ErrTooLong from Err()), which would silently drop
// every row after the first oversized one. ReadBytes returns the
// long line and lets us decide per-line whether to keep, skip, or
// stop. Skipping oversized matches the bad-JSON behaviour.
func decodeReceipts(r io.Reader) ([]receiptRow, error) {
	var out []receiptRow
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		// Trim newline + leading/trailing whitespace before deciding
		// what to do with the line. ReadBytes returns a non-empty
		// slice even on the final line without a trailing '\n',
		// hence handling the line BEFORE checking err.
		trimmed := dropTrailingNewline(line)
		switch {
		case len(trimmed) == 0:
			// blank line — fall through to err handling
		case len(trimmed) > maxReceiptLineBytes:
			// oversized — skip silently, same as bad-JSON
		default:
			var row receiptRow
			if jerr := json.Unmarshal(trimmed, &row); jerr == nil {
				out = append(out, row)
			}
		}
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return out, err
		}
	}
}

// dropTrailingNewline returns the slice without a single trailing
// '\n' or '\r\n'. ReadBytes preserves the delimiter; we don't.
func dropTrailingNewline(b []byte) []byte {
	if n := len(b); n > 0 && b[n-1] == '\n' {
		b = b[:n-1]
	}
	if n := len(b); n > 0 && b[n-1] == '\r' {
		b = b[:n-1]
	}
	return b
}

// readLastFiredFromReceipts collapses the receipt log into a
// "<world>/<name>" → most-recent-fire map. Used by `ls` to render
// the LAST FIRED column without consulting the architect.
//
// Missing receipts file is not an error — the map just comes back
// empty.
func readLastFiredFromReceipts(projectRoot string) (map[string]time.Time, error) {
	rows, err := readReceipts(projectRoot)
	if err != nil {
		return nil, err
	}
	out := map[string]time.Time{}
	for _, r := range rows {
		key := r.World + "/" + r.Automation
		if existing, ok := out[key]; !ok || r.Fired.After(existing) {
			out[key] = r.Fired
		}
	}
	return out, nil
}
