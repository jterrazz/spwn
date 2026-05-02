package automation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func sampleReceipt(t *testing.T) Receipt {
	t.Helper()
	return Receipt{
		World:      "brain",
		Automation: "morning-brief",
		Trigger:    "cron",
		Scheduled:  mustParse(t, "2026-05-01T06:00:00Z"),
		Fired:      mustParse(t, "2026-05-01T06:00:01Z"),
		Finished:   mustParse(t, "2026-05-01T06:04:23Z"),
		DurationMS: 262000,
		OK:         true,
		Reason:     "on-time",
	}
}

// ── MemoryReceiptWriter ─────────────────────────────────────────────

func TestMemoryReceiptWriter_RoundTrip(t *testing.T) {
	w := NewMemoryReceiptWriter()
	r := sampleReceipt(t)
	must(t, w.Write(r))

	got := w.Receipts()
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Automation != r.Automation {
		t.Errorf("automation = %q, want %q", got[0].Automation, r.Automation)
	}
}

func TestMemoryReceiptWriter_AppendsInOrder(t *testing.T) {
	w := NewMemoryReceiptWriter()
	for i := 0; i < 5; i++ {
		r := sampleReceipt(t)
		r.Reason = "fire-" + string(rune('a'+i))
		must(t, w.Write(r))
	}
	got := w.Receipts()
	if len(got) != 5 {
		t.Fatalf("len = %d", len(got))
	}
	for i, r := range got {
		want := "fire-" + string(rune('a'+i))
		if r.Reason != want {
			t.Errorf("receipt[%d].Reason = %q, want %q", i, r.Reason, want)
		}
	}
}

// ── FileReceiptWriter ───────────────────────────────────────────────

func TestFileReceiptWriter_AppendsJSONLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runs.jsonl")
	w := NewFileReceiptWriter(path)
	must(t, w.Write(sampleReceipt(t)))

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Errorf("file should end with newline; got %q", string(data))
	}

	// Round-trip through JSON.
	var got Receipt
	if err := json.Unmarshal([]byte(strings.TrimRight(string(data), "\n")), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Automation != "morning-brief" {
		t.Errorf("automation = %q", got.Automation)
	}
}

func TestFileReceiptWriter_AppendsAcrossWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runs.jsonl")
	w := NewFileReceiptWriter(path)
	must(t, w.Write(sampleReceipt(t)))
	r2 := sampleReceipt(t)
	r2.Reason = "catchup"
	r2.Missed = 2
	must(t, w.Write(r2))

	// Read all lines, decode each, assert two rows.
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	var rows []Receipt
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var r Receipt
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatalf("decode: %v", err)
		}
		rows = append(rows, r)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[1].Reason != "catchup" || rows[1].Missed != 2 {
		t.Errorf("row 2 = %+v", rows[1])
	}
}

func TestFileReceiptWriter_OmitsEmptyOptionals(t *testing.T) {
	// On-time receipts shouldn't carry Missed: 0 / LastFired: zero
	// to disk — the json tags use omitempty for clarity in the
	// dashboard's grep.
	dir := t.TempDir()
	path := filepath.Join(dir, "runs.jsonl")
	w := NewFileReceiptWriter(path)
	must(t, w.Write(sampleReceipt(t)))

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	row := strings.TrimSpace(string(data))
	if strings.Contains(row, `"missed"`) {
		t.Errorf("on-time receipt should not include missed field: %s", row)
	}
	if strings.Contains(row, `"last_fired"`) {
		t.Errorf("on-time receipt should not include last_fired field: %s", row)
	}
	if strings.Contains(row, `"error"`) {
		t.Errorf("ok=true receipt should not include error field: %s", row)
	}
}

func TestFileReceiptWriter_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deep", "nested", "runs.jsonl")
	w := NewFileReceiptWriter(path)
	must(t, w.Write(sampleReceipt(t)))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist: %v", err)
	}
}

// ── Rotation ────────────────────────────────────────────────────────

func TestFileReceiptWriter_RotatesPastSizeThreshold(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runs.jsonl")
	// Tiny RotateSize so the test fires fast.
	w := &FileReceiptWriter{Path: path, RotateSize: 200, RotateKeep: 3}
	must(t, w.Write(sampleReceipt(t)))
	must(t, w.Write(sampleReceipt(t))) // ~360B total → next Write triggers
	must(t, w.Write(sampleReceipt(t)))

	// .1 should exist (the original got renamed before the third write).
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf(".1 should exist after rotation: %v", err)
	}
	// The active file now has only the third receipt.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read active log: %v", err)
	}
	count := strings.Count(string(data), "\n")
	if count != 1 {
		t.Errorf("active log should have 1 line after rotation, got %d:\n%s", count, string(data))
	}
}

func TestFileReceiptWriter_ShiftsHistoricalFiles(t *testing.T) {
	// Simulate three rotations and confirm .1 / .2 / .3 exist with the
	// expected ordering (most recent in .1).
	dir := t.TempDir()
	path := filepath.Join(dir, "runs.jsonl")
	w := &FileReceiptWriter{Path: path, RotateSize: 100, RotateKeep: 3}

	// Force four rotations.
	for i := 0; i < 5; i++ {
		must(t, w.Write(sampleReceipt(t)))
		must(t, w.Write(sampleReceipt(t)))
	}

	// .1, .2, .3 should exist; .4 must NOT (RotateKeep cap).
	for i := 1; i <= 3; i++ {
		p := fmt.Sprintf("%s.%d", path, i)
		if _, err := os.Stat(p); err != nil {
			t.Errorf(".%d should exist: %v", i, err)
		}
	}
	if _, err := os.Stat(path + ".4"); err == nil {
		t.Errorf(".4 should NOT exist (cap=3)")
	}
}

func TestFileReceiptWriter_RotationDisabledWhenSizeZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runs.jsonl")
	w := &FileReceiptWriter{Path: path, RotateSize: 0}
	for i := 0; i < 10; i++ {
		must(t, w.Write(sampleReceipt(t)))
	}
	if _, err := os.Stat(path + ".1"); err == nil {
		t.Errorf(".1 should NOT exist when rotation disabled")
	}
}

func TestFileReceiptWriter_ConcurrentSafe(t *testing.T) {
	// Many writers append concurrently. The mutex serialises file
	// opens — no torn lines, every receipt persisted.
	dir := t.TempDir()
	path := filepath.Join(dir, "runs.jsonl")
	w := NewFileReceiptWriter(path)

	const N = 50
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := sampleReceipt(t)
			r.Reason = "fire-" + string(rune('a'+i%26))
			r.Fired = mustParse(t, "2026-05-01T06:00:00Z").Add(time.Duration(i) * time.Second)
			must(t, w.Write(r))
		}(i)
	}
	wg.Wait()

	// Every line decodes cleanly — no torn writes.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	count := 0
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		var r Receipt
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatalf("decode line: %v\nline=%q", err, sc.Text())
		}
		count++
	}
	if count != N {
		t.Errorf("decoded %d lines, want %d (some torn?)", count, N)
	}
}
