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
type receiptRow struct {
	World      string    `json:"world"`
	Automation string    `json:"automation"`
	Trigger    string    `json:"trigger"`
	Scheduled  time.Time `json:"scheduled,omitempty"`
	Fired      time.Time `json:"fired"`
	Finished   time.Time `json:"finished"`
	DurationMS int64     `json:"duration_ms"`
	OK         bool      `json:"ok"`
	Reason     string    `json:"reason"`
	Error      string    `json:"error,omitempty"`
	Missed     int       `json:"missed,omitempty"`
	LastFired  time.Time `json:"last_fired,omitempty"`
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

// decodeReceipts is split out so tests can pass an in-memory reader
// without touching the filesystem.
func decodeReceipts(r io.Reader) ([]receiptRow, error) {
	var out []receiptRow
	sc := bufio.NewScanner(r)
	// Default token size is 64KB. Receipts are small, but a future
	// catch-up mode that includes long error strings could push
	// near. 1MB cap is plenty without inviting attacks.
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		var row receiptRow
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
			continue
		}
		out = append(out, row)
	}
	return out, sc.Err()
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
