package automation

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

func receiptsPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".spwn", "runs.jsonl")
}

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

const maxReceiptLineBytes = 4 * 1024 * 1024

func decodeReceipts(r io.Reader) ([]receiptRow, error) {
	var out []receiptRow
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		trimmed := dropTrailingNewline(line)
		switch {
		case len(trimmed) == 0:
		case len(trimmed) > maxReceiptLineBytes:
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

func dropTrailingNewline(b []byte) []byte {
	if n := len(b); n > 0 && b[n-1] == '\n' {
		b = b[:n-1]
	}
	if n := len(b); n > 0 && b[n-1] == '\r' {
		b = b[:n-1]
	}
	return b
}

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
