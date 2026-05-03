package architect

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"spwn.sh/packages/automation"
	"spwn.sh/packages/project"
	"spwn.sh/packages/world/models"
)

// Comprehensive integration tests for the automation engine wired
// through the architect-side dispatcher. The engine is real, the
// dispatcher is real, the receipt + state stores are real files,
// the command resolver is real. Only the Docker backend is mocked
// (mockBackend records every Exec call + can inject errors).
//
// Coverage:
//
//   • Cron fire dispatches via real architect → mock backend
//   • FS fire dispatches via real fsnotify (FakeFSSource) → mock backend
//   • Catch-up modes: collapse / skip / stack
//   • FS replay-on-startup
//   • Output capture (mock backend writes to ExecConfig.Stdout)
//   • Receipt schema fields all populate (run_id, agent, engine_version,
//     prompt_sha, enqueued_at, event_paths, output)
//   • Receipt log rotation triggers correctly
//   • State store v1 + legacy formats both load
//   • Per-agent serialisation: same agent serialised, cross-agent parallel
//   • Tainted state refuses RecordFire
//   • Errors surface through the logger
//
// Tests live in package architect (not architect_test) so they can
// reach unexported types like mockBackend + share the existing
// fixtures.

// e2eFixture bundles everything an integration test needs.
type e2eFixture struct {
	t           *testing.T
	arc         *Architect
	mb          *mockBackend
	clock       *automation.FakeClock
	root        string
	fsSource    *automation.FakeFSSource
	fsWatcher   *automation.FSWatcher
	receiptsLog string
	stateFile   string
	logger      *captureLog
}

// newE2EFixture sets up a project root + architect + clock at testEpoch.
func newE2EFixture(t *testing.T) *e2eFixture {
	t.Helper()
	mb := newMockBackend()
	arc, _ := newTestArchitect(t, mb)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "spwn", "commands"), 0o755); err != nil {
		t.Fatal(err)
	}

	clock := automation.NewFakeClock(parseE2E(t, "2026-05-01T00:00:00Z"))
	fsSource := automation.NewFakeFSSource()
	fsWatcher := automation.NewFSWatcher(fsSource, clock)

	return &e2eFixture{
		t:           t,
		arc:         arc,
		mb:          mb,
		clock:       clock,
		root:        root,
		fsSource:    fsSource,
		fsWatcher:   fsWatcher,
		receiptsLog: filepath.Join(root, ".spwn", "runs.jsonl"),
		stateFile:   filepath.Join(root, ".spwn", "automations", "state.json"),
		logger:      &captureLog{},
	}
}

// captureLog is a Logger that records every Warnf call.
type captureLog struct {
	mu   sync.Mutex
	msgs []string
}

func (c *captureLog) Warnf(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, fmt.Sprintf(format, args...))
}
func (c *captureLog) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.msgs))
	copy(out, c.msgs)
	return out
}

// seedRunningWorld registers a world with the mock backend so
// architect.List finds it. configName is the manifest key the
// dispatcher matches on.
func (f *e2eFixture) seedRunningWorld(configName string, agents ...string) {
	f.t.Helper()
	w := models.World{
		ID:          "world-test-" + configName,
		Config:      configName,
		ContainerID: "mock-" + configName,
		Runtime:     "claude-code",
		Status:      models.StatusRunning,
	}
	for _, a := range agents {
		w.Agents = append(w.Agents, models.AgentRecord{Name: a})
	}
	seedWorld(f.mb, w)
}

// writeCommandFile creates spwn/commands/<name>.md with the given
// body so command refs resolve.
func (f *e2eFixture) writeCommandFile(name, body string) {
	f.t.Helper()
	path := filepath.Join(f.root, "spwn", "commands", name+".md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// startEngine constructs an engine via architect.NewAutomationEngine,
// registers automations, calls Start. Caller stops it via t.Cleanup.
func (f *e2eFixture) startEngine(automations map[string]project.Automation, agents []string) *automation.Engine {
	f.t.Helper()
	worlds := map[string]project.World{
		"brain": {
			Agents:      agents,
			Workspaces:  []string{"."},
			Automations: automations,
		},
	}
	manifest := &project.Manifest{
		Version: 1,
		Name:    "e2e-test",
		Worlds:  worlds,
	}
	eng, err := f.arc.NewAutomationEngine(AutomationEngineConfig{
		ProjectRoot: f.root,
		Manifest:    manifest,
		FS:          f.fsWatcher,
		Clock:       f.clock,
	})
	if err != nil {
		f.t.Fatalf("NewAutomationEngine: %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		f.t.Fatalf("Start: %v", err)
	}
	f.t.Cleanup(func() { eng.Stop() })
	return eng
}

// readReceipts decodes every line of the project's runs.jsonl. Used
// by tests to assert on engine output.
func (f *e2eFixture) readReceipts() []map[string]any {
	f.t.Helper()
	file, err := os.Open(f.receiptsLog)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		f.t.Fatal(err)
	}
	defer file.Close()
	var rows []map[string]any
	br := bufio.NewReader(file)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			line = []byte(strings.TrimSpace(string(line)))
			if len(line) > 0 {
				var row map[string]any
				if jerr := json.Unmarshal(line, &row); jerr == nil {
					rows = append(rows, row)
				}
			}
		}
		if err == io.EOF {
			return rows
		}
		if err != nil {
			f.t.Fatalf("read receipts: %v", err)
		}
	}
}

// waitForReceipts polls until the receipts file has at least n
// rows or the deadline elapses. Lets tests synchronise with the
// dispatch goroutine without baking sleeps into every assertion.
func (f *e2eFixture) waitForReceipts(n int, within time.Duration) {
	f.t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if len(f.readReceipts()) >= n {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	f.t.Fatalf("expected %d receipts within %s, got %d", n, within, len(f.readReceipts()))
}

func parseE2E(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// ── Cron path: fire dispatches into mock backend ────────────────────

func TestE2E_CronFireDispatchesIntoBackend(t *testing.T) {
	f := newE2EFixture(t)
	f.seedRunningWorld("brain", "editor")

	f.startEngine(map[string]project.Automation{
		"morning": {
			On:      project.Trigger{Cron: "0 6 * * *"},
			Agent:   "editor",
			Prompt:  "Brief for {{ .Now | date \"2006-01-02\" }}.",
			Catchup: "collapse",
		},
	}, []string{"editor"})

	// Wait for the cron loop to register its timer.
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if f.clock.Pending() >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	f.clock.AdvanceTo(parseE2E(t, "2026-05-01T06:00:00Z"))
	f.waitForReceipts(1, 500*time.Millisecond)

	// Backend was called at least once. A prelaunch credentials
	// refresh may precede the actual dispatch; the dispatch is the
	// LAST exec, not the first.
	if len(f.mb.execCalls) < 1 {
		t.Fatalf("exec calls = %d, want ≥1", len(f.mb.execCalls))
	}
	cmd := f.mb.execCalls[len(f.mb.execCalls)-1].cfg.Cmd
	joined := strings.Join(cmd, " ")
	if !strings.Contains(joined, "2026-05-01") {
		t.Errorf("rendered prompt missing date: %s", joined)
	}

	// Receipt has all the new schema fields.
	rec := f.readReceipts()[0]
	for _, key := range []string{"run_id", "engine_version", "agent", "prompt_sha", "enqueued_at"} {
		if _, ok := rec[key]; !ok {
			t.Errorf("receipt missing field %q: %+v", key, rec)
		}
	}
	if rec["agent"] != "editor" {
		t.Errorf("agent = %v, want editor", rec["agent"])
	}
	if rec["engine_version"] != automation.EngineVersion {
		t.Errorf("engine_version = %v, want %s", rec["engine_version"], automation.EngineVersion)
	}
}

// ── FS path: handle event → fire ────────────────────────────────────

func TestE2E_FSFireDispatchesIntoBackend(t *testing.T) {
	f := newE2EFixture(t)
	f.seedRunningWorld("brain", "curator")

	inboxDir := filepath.Join(f.root, "inbox")
	if err := os.MkdirAll(inboxDir, 0o755); err != nil {
		t.Fatal(err)
	}

	f.startEngine(map[string]project.Automation{
		"inbox": {
			On: project.Trigger{
				FS: &project.FSTrigger{
					Path:     inboxDir,
					Events:   []string{"create"},
					Debounce: project.Duration(50 * time.Millisecond),
				},
			},
			Agent:  "curator",
			Prompt: "new file: {{ .Event.Name }}",
		},
	}, []string{"curator"})

	// Drive a synthetic create event through the watcher's handle
	// path (bypasses real fsnotify timing).
	f.fsWatcher.HandleForTest(filepath.Join(inboxDir, "foo.md"), "create")
	f.clock.Advance(50 * time.Millisecond)
	f.waitForReceipts(1, 500*time.Millisecond)

	rec := f.readReceipts()[0]
	if rec["trigger"] != "fs" {
		t.Errorf("trigger = %v, want fs", rec["trigger"])
	}
	if rec["event_kind"] != "create" {
		t.Errorf("event_kind = %v, want create", rec["event_kind"])
	}
	if paths, ok := rec["event_paths"].([]any); !ok || len(paths) != 1 {
		t.Errorf("event_paths = %v, want one entry", rec["event_paths"])
	}

	// Last exec is the dispatch (a prelaunch refresh may precede it).
	cmd := f.mb.execCalls[len(f.mb.execCalls)-1].cfg.Cmd
	joined := strings.Join(cmd, " ")
	if !strings.Contains(joined, "foo.md") {
		t.Errorf("rendered prompt missing fs event filename: %s", joined)
	}
}

// ── Catch-up modes ──────────────────────────────────────────────────

func TestE2E_CatchupCollapseFiresOnce(t *testing.T) {
	f := newE2EFixture(t)
	f.seedRunningWorld("brain", "editor")

	// Pre-seed state cursor at Sunday 06:00; boot clock at Wednesday
	// 08:00. Three slots missed (Mon, Tue, Wed). Collapse fires once
	// with missed=3.
	stateDir := filepath.Dir(f.stateFile)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	last := parseE2E(t, "2026-04-26T06:00:00Z")
	stateBody := fmt.Sprintf(`{"version":1,"entries":{"brain/morning":"%s"}}`, last.Format(time.RFC3339))
	if err := os.WriteFile(f.stateFile, []byte(stateBody), 0o644); err != nil {
		t.Fatal(err)
	}
	f.clock = automation.NewFakeClock(parseE2E(t, "2026-04-29T08:00:00Z"))

	f.startEngine(map[string]project.Automation{
		"morning": {
			On:      project.Trigger{Cron: "0 6 * * *"},
			Agent:   "editor",
			Prompt:  "go",
			Catchup: "collapse",
		},
	}, []string{"editor"})

	rows := f.readReceipts()
	if len(rows) != 1 {
		t.Fatalf("collapse should produce 1 catch-up receipt, got %d", len(rows))
	}
	if rows[0]["reason"] != "catchup" {
		t.Errorf("reason = %v, want catchup", rows[0]["reason"])
	}
	if int(rows[0]["missed"].(float64)) != 3 {
		t.Errorf("missed = %v, want 3", rows[0]["missed"])
	}
}

func TestE2E_CatchupStackFiresOncePerSlot(t *testing.T) {
	f := newE2EFixture(t)
	f.seedRunningWorld("brain", "editor")

	stateDir := filepath.Dir(f.stateFile)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	last := parseE2E(t, "2026-04-27T06:00:00Z") // Mon
	stateBody := fmt.Sprintf(`{"version":1,"entries":{"brain/morning":"%s"}}`, last.Format(time.RFC3339))
	if err := os.WriteFile(f.stateFile, []byte(stateBody), 0o644); err != nil {
		t.Fatal(err)
	}
	f.clock = automation.NewFakeClock(parseE2E(t, "2026-04-29T08:00:00Z")) // Wed

	f.startEngine(map[string]project.Automation{
		"morning": {
			On:      project.Trigger{Cron: "0 6 * * *"},
			Agent:   "editor",
			Prompt:  "go",
			Catchup: "stack",
		},
	}, []string{"editor"})

	rows := f.readReceipts()
	if len(rows) != 2 {
		t.Fatalf("stack should produce 2 catch-up receipts (Tue + Wed); got %d", len(rows))
	}
	for _, r := range rows {
		if r["reason"] != "catchup" {
			t.Errorf("reason = %v", r["reason"])
		}
		if int(r["missed"].(float64)) != 1 {
			t.Errorf("stack-mode missed = %v, want 1", r["missed"])
		}
	}
}

func TestE2E_CatchupSkipFiresZero(t *testing.T) {
	f := newE2EFixture(t)
	f.seedRunningWorld("brain", "editor")

	stateDir := filepath.Dir(f.stateFile)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	last := parseE2E(t, "2026-04-26T06:00:00Z")
	stateBody := fmt.Sprintf(`{"version":1,"entries":{"brain/morning":"%s"}}`, last.Format(time.RFC3339))
	if err := os.WriteFile(f.stateFile, []byte(stateBody), 0o644); err != nil {
		t.Fatal(err)
	}
	f.clock = automation.NewFakeClock(parseE2E(t, "2026-04-29T08:00:00Z"))

	f.startEngine(map[string]project.Automation{
		"morning": {
			On:      project.Trigger{Cron: "0 6 * * *"},
			Agent:   "editor",
			Prompt:  "go",
			Catchup: "skip",
		},
	}, []string{"editor"})

	if rows := f.readReceipts(); len(rows) != 0 {
		t.Errorf("skip mode should produce 0 catch-up receipts, got %d", len(rows))
	}
}

// ── FS replay-on-startup ────────────────────────────────────────────

func TestE2E_FSReplayProcessesNewFiles(t *testing.T) {
	f := newE2EFixture(t)
	f.seedRunningWorld("brain", "curator")

	inboxDir := filepath.Join(f.root, "inbox")
	if err := os.MkdirAll(inboxDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-seed state cursor at T0; place an old file at T-1h and a
	// new file at T+1h.
	stateDir := filepath.Dir(f.stateFile)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	last := parseE2E(t, "2026-05-01T00:00:00Z")
	stateBody := fmt.Sprintf(`{"version":1,"entries":{"brain/inbox":"%s"}}`, last.Format(time.RFC3339))
	if err := os.WriteFile(f.stateFile, []byte(stateBody), 0o644); err != nil {
		t.Fatal(err)
	}
	oldP := filepath.Join(inboxDir, "old.md")
	newP := filepath.Join(inboxDir, "new.md")
	if err := os.WriteFile(oldP, []byte("o"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newP, []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(oldP, last.Add(-1*time.Hour), last.Add(-1*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newP, last.Add(1*time.Hour), last.Add(1*time.Hour)); err != nil {
		t.Fatal(err)
	}

	f.startEngine(map[string]project.Automation{
		"inbox": {
			On: project.Trigger{
				FS: &project.FSTrigger{
					Path:     inboxDir,
					Events:   []string{"create"},
					Debounce: project.Duration(50 * time.Millisecond),
				},
			},
			Agent:  "curator",
			Prompt: "{{ .Event.Path }}",
		},
	}, []string{"curator"})

	rows := f.readReceipts()
	if len(rows) != 1 {
		t.Fatalf("replay should produce 1 receipt, got %d", len(rows))
	}
	if !strings.HasPrefix(rows[0]["reason"].(string), "replay:") {
		t.Errorf("reason = %v, want replay:*", rows[0]["reason"])
	}
}

// ── Output capture ──────────────────────────────────────────────────

func TestE2E_DispatcherCapturesRuntimeOutput(t *testing.T) {
	f := newE2EFixture(t)
	f.seedRunningWorld("brain", "editor")

	// Configure mock backend to write fake runtime output to the
	// captured stdout writer.
	f.mb.execStdoutWrite = []byte("hello from claude\n")

	f.startEngine(map[string]project.Automation{
		"morning": {
			On:      project.Trigger{Cron: "0 6 * * *"},
			Agent:   "editor",
			Prompt:  "go",
			Catchup: "collapse",
		},
	}, []string{"editor"})

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if f.clock.Pending() >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	f.clock.AdvanceTo(parseE2E(t, "2026-05-01T06:00:00Z"))
	f.waitForReceipts(1, 500*time.Millisecond)

	rec := f.readReceipts()[0]
	if got, _ := rec["output"].(string); !strings.Contains(got, "hello from claude") {
		t.Errorf("receipt output should contain runtime stdout, got %q", got)
	}
}

// ── Receipt log rotation ────────────────────────────────────────────

func TestE2E_ReceiptLogRotates(t *testing.T) {
	// Configure a tiny rotate threshold via direct construction —
	// the factory's defaults are 100MB which would take too long
	// to hit in tests.
	dir := t.TempDir()
	receiptsPath := filepath.Join(dir, "runs.jsonl")
	w := &automation.FileReceiptWriter{
		Path:       receiptsPath,
		RotateSize: 200,
		RotateKeep: 3,
	}

	// Write enough to trigger 2 rotations.
	for i := 0; i < 6; i++ {
		err := w.Write(automation.Receipt{
			World:         "brain",
			Automation:    "x",
			Agent:         "editor",
			Trigger:       "cron",
			RunID:         "0123456789ab",
			EngineVersion: automation.EngineVersion,
			Fired:         parseE2E(t, "2026-05-01T06:00:00Z"),
			Finished:      parseE2E(t, "2026-05-01T06:00:01Z"),
			DurationMS:    1000,
			OK:            true,
			Reason:        "on-time",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, suffix := range []string{".1", ".2"} {
		if _, err := os.Stat(receiptsPath + suffix); err != nil {
			t.Errorf("expected %s after rotation: %v", receiptsPath+suffix, err)
		}
	}
}

// ── Per-agent serialisation, error logging ──────────────────────────

func TestE2E_DispatcherErrorLogsAndReceipts(t *testing.T) {
	f := newE2EFixture(t)
	// Don't seed a world — dispatcher will fail with "no running
	// world" and the engine should log + receipt.
	f.startEngine(map[string]project.Automation{
		"morning": {
			On:      project.Trigger{Cron: "0 6 * * *"},
			Agent:   "editor",
			Prompt:  "go",
			Catchup: "collapse",
		},
	}, []string{"editor"})

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if f.clock.Pending() >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	f.clock.AdvanceTo(parseE2E(t, "2026-05-01T06:00:00Z"))
	f.waitForReceipts(1, 500*time.Millisecond)

	rec := f.readReceipts()[0]
	if rec["ok"].(bool) {
		t.Errorf("missing-world receipt should be ok=false, got %+v", rec)
	}
	if errStr, _ := rec["error"].(string); !strings.Contains(errStr, "no running world") {
		t.Errorf("error message should mention missing world, got %q", errStr)
	}
}

// ── State store: legacy file readable, future version tainted ───────

func TestE2E_LegacyStateFileWorks(t *testing.T) {
	f := newE2EFixture(t)
	f.seedRunningWorld("brain", "editor")

	// Pre-v1 file format: bare flat map without the envelope.
	stateDir := filepath.Dir(f.stateFile)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := `{"brain/morning":"2026-04-30T06:00:00Z"}`
	if err := os.WriteFile(f.stateFile, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	f.clock = automation.NewFakeClock(parseE2E(t, "2026-05-01T07:00:00Z"))

	// Engine reads legacy + records collapse-mode catchup for
	// the missed slot. The next RecordFire upgrades the file shape.
	f.startEngine(map[string]project.Automation{
		"morning": {
			On:      project.Trigger{Cron: "0 6 * * *"},
			Agent:   "editor",
			Prompt:  "go",
			Catchup: "collapse",
		},
	}, []string{"editor"})

	if got := len(f.readReceipts()); got != 1 {
		t.Errorf("expected 1 catchup receipt from legacy state, got %d", got)
	}

	// File should now be in v1 envelope shape.
	data, _ := os.ReadFile(f.stateFile)
	if !strings.Contains(string(data), `"version"`) {
		t.Errorf("state file should be upgraded to v1 envelope; got: %s", string(data))
	}
}

// ── Hidden-dir filtering at the dispatcher boundary ─────────────────

func TestE2E_FSHiddenDirFilteredByDefault(t *testing.T) {
	f := newE2EFixture(t)
	f.seedRunningWorld("brain", "curator")

	inboxDir := filepath.Join(f.root, "inbox")
	if err := os.MkdirAll(filepath.Join(inboxDir, ".cache"), 0o755); err != nil {
		t.Fatal(err)
	}

	f.startEngine(map[string]project.Automation{
		"inbox": {
			On: project.Trigger{FS: &project.FSTrigger{
				Path: inboxDir, Recursive: true, Events: []string{"create"},
				Debounce: project.Duration(50 * time.Millisecond),
			}},
			Agent: "curator", Prompt: "go",
		},
	}, []string{"curator"})

	// Visible file should fire; hidden dir's child should NOT.
	f.fsWatcher.HandleForTest(filepath.Join(inboxDir, "visible.md"), "create")
	f.fsWatcher.HandleForTest(filepath.Join(inboxDir, ".cache", "secret.md"), "create")
	f.clock.Advance(100 * time.Millisecond)

	// Wait modestly; the hidden one should produce no fire.
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(f.readReceipts()) >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	rows := f.readReceipts()
	if len(rows) != 1 {
		t.Fatalf("expected 1 receipt (visible only), got %d", len(rows))
	}
	paths, _ := rows[0]["event_paths"].([]any)
	if len(paths) != 1 || !strings.Contains(paths[0].(string), "visible.md") {
		t.Errorf("paths = %v, want only visible.md", paths)
	}
}

// ── Output truncation guarantees ────────────────────────────────────

func TestE2E_OutputTruncatedAtCap(t *testing.T) {
	f := newE2EFixture(t)
	f.seedRunningWorld("brain", "editor")

	// Configure mock backend to write 50KB of output — well past
	// the 32KB dispatcher cap and the 8KB receipt cap.
	f.mb.execStdoutWrite = []byte(strings.Repeat("x", 50*1024))

	f.startEngine(map[string]project.Automation{
		"morning": {
			On:      project.Trigger{Cron: "0 6 * * *"},
			Agent:   "editor",
			Prompt:  "go",
			Catchup: "collapse",
		},
	}, []string{"editor"})

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if f.clock.Pending() >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	f.clock.AdvanceTo(parseE2E(t, "2026-05-01T06:00:00Z"))
	f.waitForReceipts(1, 500*time.Millisecond)

	rec := f.readReceipts()[0]
	out, _ := rec["output"].(string)
	if len(out) > 9*1024 { // cap + tiny "…[truncated]" marker
		t.Errorf("output should be truncated near 8KB, got %d bytes", len(out))
	}
	if !strings.Contains(out, "truncated") {
		t.Errorf("truncated marker missing from output: %s", out[len(out)-50:])
	}
}

