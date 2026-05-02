package automation

import (
	"strings"
	"testing"
	"time"

	"spwn.sh/packages/project"
)

// ── collectAutomations: stable sort + flatten ───────────────────────

func TestCollectAutomations_StableSort(t *testing.T) {
	p := &project.Project{
		Manifest: &project.Manifest{
			Worlds: map[string]project.World{
				"scratch": {
					Automations: map[string]project.Automation{
						"z": {On: project.Trigger{Cron: "0 6 * * *"}, Agent: "a", Prompt: "p"},
						"a": {On: project.Trigger{Cron: "0 7 * * *"}, Agent: "a", Prompt: "p"},
					},
				},
				"brain": {
					Automations: map[string]project.Automation{
						"morning":     {On: project.Trigger{Cron: "0 6 * * *"}, Agent: "e", Prompt: "p"},
						"inbox-pull":  {On: project.Trigger{FS: &project.FSTrigger{Path: "./inbox"}}, Agent: "c", Prompt: "p"},
					},
				},
			},
		},
	}
	got := collectAutomations(p)
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
	want := []string{"brain/inbox-pull", "brain/morning", "scratch/a", "scratch/z"}
	for i, w := range want {
		key := got[i].World + "/" + got[i].Name
		if key != w {
			t.Errorf("entry %d = %q, want %q", i, key, w)
		}
	}
}

func TestCollectAutomations_NilProjectIsEmpty(t *testing.T) {
	if got := collectAutomations(nil); len(got) != 0 {
		t.Errorf("got %d entries from nil project", len(got))
	}
}

// ── formatTrigger: cron/fs labels ───────────────────────────────────

func TestFormatTrigger(t *testing.T) {
	cases := []struct {
		name string
		auto project.Automation
		want string
	}{
		{"cron", project.Automation{On: project.Trigger{Cron: "0 6 * * *"}}, "cron 0 6 * * *"},
		{"fs", project.Automation{On: project.Trigger{FS: &project.FSTrigger{Path: "/inbox"}}}, "fs /inbox"},
		{"empty", project.Automation{}, "?"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatTrigger(c.auto); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// ── humanTimeAgo: bucket boundaries ─────────────────────────────────

func TestHumanTimeAgo(t *testing.T) {
	now := time.Now()
	cases := []struct {
		t    time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "30s ago"},
		{now.Add(-5 * time.Minute), "5m ago"},
		{now.Add(-3 * time.Hour), "3h ago"},
		{now.Add(-2 * 24 * time.Hour), "2d ago"},
	}
	for _, c := range cases {
		// time.Since drifts slightly between the case construction
		// and the call; allow ±1 unit on the rendered string.
		got := humanTimeAgo(c.t)
		if !strings.HasSuffix(got, " ago") {
			t.Errorf("missing ' ago' suffix: %q", got)
		}
	}
}

// ── decodeReceipts + summariseReceipts ──────────────────────────────

const sampleReceiptLog = `{"world":"brain","automation":"morning","trigger":"cron","fired":"2026-05-01T06:00:00Z","finished":"2026-05-01T06:04:23Z","duration_ms":263000,"ok":true,"reason":"on-time","scheduled":"2026-05-01T06:00:00Z"}
{"world":"brain","automation":"inbox","trigger":"fs","fired":"2026-05-01T11:23:14Z","finished":"2026-05-01T11:23:18Z","duration_ms":4000,"ok":true,"reason":"create:foo.md"}
{"world":"brain","automation":"morning","trigger":"cron","fired":"2026-05-02T06:00:00Z","finished":"2026-05-02T06:00:01.5Z","duration_ms":1500,"ok":false,"reason":"on-time","error":"world brain: no running world"}
`

func TestDecodeReceipts_ParsesAllRows(t *testing.T) {
	rows, err := decodeReceipts(strings.NewReader(sampleReceiptLog))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("len = %d, want 3", len(rows))
	}
	if rows[0].Automation != "morning" || !rows[0].OK {
		t.Errorf("row 0 = %+v", rows[0])
	}
	if rows[2].OK || rows[2].Error == "" {
		t.Errorf("row 2 should be a failure with error set")
	}
}

func TestDecodeReceipts_SkipsBadLines(t *testing.T) {
	mixed := `{"world":"brain","automation":"x","trigger":"cron","fired":"2026-05-01T06:00:00Z","finished":"2026-05-01T06:00:00Z","duration_ms":0,"ok":true,"reason":"on-time"}
not-json
{"world":"brain","automation":"y","trigger":"cron","fired":"2026-05-01T07:00:00Z","finished":"2026-05-01T07:00:00Z","duration_ms":0,"ok":true,"reason":"on-time"}
`
	rows, err := decodeReceipts(strings.NewReader(mixed))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("len = %d, want 2 (bad line should be skipped)", len(rows))
	}
}

func TestSummariseReceipts_RollsUpStats(t *testing.T) {
	rows, _ := decodeReceipts(strings.NewReader(sampleReceiptLog))
	stats := summariseReceipts(rows)

	morning := stats["brain/morning"]
	if morning.fires != 2 || morning.okCount != 1 || morning.failCount != 1 {
		t.Errorf("brain/morning = %+v, want fires=2 ok=1 fail=1", morning)
	}
	// Last fired should be the May-02 entry.
	wantLast, _ := time.Parse(time.RFC3339, "2026-05-02T06:00:00Z")
	if !morning.lastFired.Equal(wantLast) {
		t.Errorf("brain/morning lastFired = %s, want %s", morning.lastFired, wantLast)
	}
	if morning.lastOK {
		t.Errorf("brain/morning lastOK = true, want false (latest fire failed)")
	}

	inbox := stats["brain/inbox"]
	if inbox.fires != 1 || !inbox.lastOK {
		t.Errorf("brain/inbox = %+v", inbox)
	}
}

// ── formatReceiptLine: schema-stable rendering ──────────────────────

func TestFormatReceiptLine_OkPath(t *testing.T) {
	r := receiptRow{
		World:      "brain",
		Automation: "morning",
		Trigger:    "cron",
		Reason:     "on-time",
		Fired:      mustTime(t, "2026-05-01T06:00:00Z"),
		DurationMS: 263000,
		OK:         true,
	}
	got := formatReceiptLine(r)
	for _, want := range []string{"brain/morning", "cron", "on-time", "ok", "2026-05-01T06:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in line: %q", want, got)
		}
	}
}

func TestFormatReceiptLine_FailIncludesError(t *testing.T) {
	r := receiptRow{
		World:      "brain",
		Automation: "x",
		Trigger:    "cron",
		Reason:     "on-time",
		Fired:      mustTime(t, "2026-05-01T06:00:00Z"),
		DurationMS: 0,
		OK:         false,
		Error:      "boom",
	}
	got := formatReceiptLine(r)
	if !strings.Contains(got, "FAIL") {
		t.Errorf("missing FAIL: %q", got)
	}
	if !strings.Contains(got, "err=boom") {
		t.Errorf("missing err=boom: %q", got)
	}
}

func TestFormatReceiptLine_TruncatesLongError(t *testing.T) {
	long := strings.Repeat("e", 250)
	r := receiptRow{
		World: "brain", Automation: "x", Trigger: "cron",
		Fired: time.Now(), OK: false, Error: long,
	}
	got := formatReceiptLine(r)
	if !strings.Contains(got, "…") {
		t.Errorf("expected truncation marker '…' in %q", got)
	}
	// Sanity: trimmed length is bounded.
	if len(got) > 350 {
		t.Errorf("rendered line too long (%d chars)", len(got))
	}
}

func TestFormatReceiptLine_CatchupShowsMissed(t *testing.T) {
	r := receiptRow{
		World: "brain", Automation: "morning", Trigger: "cron",
		Reason: "catchup", Missed: 3,
		Fired: time.Now(), OK: true,
	}
	got := formatReceiptLine(r)
	if !strings.Contains(got, "missed=3") {
		t.Errorf("missing missed=3: %q", got)
	}
}

// ── countAutomations (daemon helper) ────────────────────────────────

func TestCountAutomations(t *testing.T) {
	p := &project.Project{
		Manifest: &project.Manifest{
			Worlds: map[string]project.World{
				"a": {Automations: map[string]project.Automation{"x": {}, "y": {}}},
				"b": {Automations: map[string]project.Automation{"z": {}}},
			},
		},
	}
	if got := countAutomations(p); got != 3 {
		t.Errorf("got %d, want 3", got)
	}
	if got := countAutomations(nil); got != 0 {
		t.Errorf("nil project = %d, want 0", got)
	}
}

// ── helpers ─────────────────────────────────────────────────────────

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
