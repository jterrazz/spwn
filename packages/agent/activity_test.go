package agent

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMindActs_AppendTheirWholeEventToTheActivityLog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)

	if _, err := InitMind("neo"); err != nil {
		t.Fatalf("InitMind: %v", err)
	}
	if _, err := Fork("neo", "trinity", nil); err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if _, err := Sleep("neo"); err != nil {
		t.Fatalf("Sleep: %v", err)
	}
	if err := DeleteAgent("trinity"); err != nil {
		t.Fatalf("DeleteAgent: %v", err)
	}

	want := []string{
		`{"actor":"user","agent_id":"neo","phrase":"You created neo","target":"neo","type":"agent.created","verb":"created"}`,
		`{"actor":"user","agent_id":"trinity","metadata":{"layers":["playbooks","journal"],"source":"neo"},"phrase":"trinity forked from neo","target":"trinity","type":"agent.forked","verb":"forked"}`,
		`{"actor":"neo","agent_id":"neo","metadata":{"archived_playbooks":0,"pruned_sessions":0},"phrase":"neo slept","target":"neo","type":"agent.slept","verb":"slept"}`,
		`{"actor":"user","agent_id":"trinity","phrase":"trinity was deleted","target":"trinity","type":"agent.deleted","verb":"deleted"}`,
	}
	got := activityLines(t, filepath.Join(home, "activity.jsonl"))
	if len(got) != len(want) {
		t.Fatalf("activity.jsonl holds %d events, want %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d:\n got  %s\n want %s", i, got[i], want[i])
		}
	}
}

func activityLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	var lines []string
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		var event map[string]any
		if err := json.Unmarshal(scan.Bytes(), &event); err != nil {
			t.Fatalf("line %q: %v", scan.Text(), err)
		}
		for _, field := range []string{"id", "timestamp"} {
			if _, ok := event[field]; !ok {
				t.Fatalf("line %q has no %s", scan.Text(), field)
			}
			delete(event, field)
		}
		line, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(line))
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}
