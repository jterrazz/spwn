package engine

import (
	"path"
	"strings"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

type Hello struct {
	Version    string    `json:"version"`
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"started_at"`
	Executable string    `json:"executable,omitempty"`
	// value: the names of the Registry's calls this engine serves, never its own.
	Calls []string `json:"calls,omitempty"`
	// value: the abilities of its own calls it serves (engine.Abilities);
	// none from an engine older than the field.
	Abilities []string `json:"abilities,omitempty"`
	Page      int      `json:"page,omitempty"`
	// value: how its Mac stands with macOS's Local Network approval of
	// station, as the engine last read it; empty where it cannot tell.
	LocalNetwork model.LocalNetwork `json:"local_network,omitempty"`
}

type VMView struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	RunPID int    `json:"run_pid"`
	// value: the process that holds its guest on this Mac, 0 while none does.
	Holder int    `json:"holder,omitempty"`
	Guest  string `json:"guest"`
	IP     string `json:"ip"`
	Detail string `json:"detail"`
	// value: its ssh connection opened, whether station's guest answers on
	// it or not; a server's restarts and health check.
	SSH      bool              `json:"ssh,omitempty"`
	Restarts model.Restarts    `json:"restarts,omitzero"`
	Check    model.CheckStatus `json:"check,omitzero"`
	// value: when its run began, while it runs.
	BootAt time.Time `json:"boot_at,omitzero"`
	// value: a server's last check runs (SPEC B5.4), and the health step its
	// restart runs now, nil when none runs (B5.11).
	Latency []model.LatencyPoint `json:"latency,omitempty"`
	Restart *model.RestartWalk   `json:"restart,omitempty"`
	// value: why a server is down, and the last lines its console kept
	// then (SPEC B5.5); empty while it is up.
	Cause     string   `json:"cause,omitempty"`
	LastLines []string `json:"last_lines,omitempty"`
	// value: when the create that made it made it ready (SPEC B5.10).
	ReadyAt time.Time `json:"ready_at,omitzero"`
	// value: how a running workspace stands with its file, as the drift
	// job last read it since its boot; nil before that reading.
	Drift *model.KeptDrift `json:"drift,omitempty"`
	// value: what refused its guest's last connection, while nothing has
	// connected since; empty when nothing did.
	Blocked model.GuestBlock `json:"blocked,omitempty"`
}

func SocketPath(state string) string { return path.Join(state, "engine.sock") }

func LogPath(state string) string { return path.Join(state, "logs", "engine.log") }

const (
	// value: what a read of the engine's log keeps: its last lines, from the
	// last bytes of its file, which launchd's output only grows.
	LogLines = 500
	LogBytes = 256 * 1024
	// value: what the engine says when its log holds no line.
	LogEmpty = "The engine has written no line here yet."
)

type Log struct {
	Lines []string
	// value: why Lines is empty, in the engine's words; absent otherwise.
	Empty string `json:",omitempty"`
}

func LogOf(tail []byte, cut bool) Log {
	// invariant: the last LogLines non-blank lines of the tail; a tail cut
	// inside a line (the file is longer than LogBytes) drops its first,
	// partial line.
	lines := strings.Split(string(tail), "\n")
	if cut && len(lines) > 0 {
		lines = lines[1:]
	}
	kept := make([]string, 0, min(len(lines), LogLines))
	for _, line := range lines {
		if line = strings.TrimRight(line, "\r"); strings.TrimSpace(line) != "" {
			kept = append(kept, line)
		}
	}
	if len(kept) > LogLines {
		kept = kept[len(kept)-LogLines:]
	}
	if len(kept) == 0 {
		return Log{Lines: []string{}, Empty: LogEmpty}
	}
	return Log{Lines: kept}
}

func ChildrenDir(state string) string { return path.Join(state, "children") }

type AwakeIn struct {
	Lid     bool `json:"lid,omitempty"`
	Battery bool `json:"battery,omitempty"`
	// value: seconds.
	For int    `json:"for,omitempty"`
	By  string `json:"by,omitempty"`
	// value: the end asked (until 17:00), which wins over For.
	Until *time.Time `json:"until,omitempty"`
}
