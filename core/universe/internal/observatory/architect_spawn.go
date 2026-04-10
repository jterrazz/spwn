package observatory

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"
)

// ArchitectSpawnOpts mirrors universe.StartArchitectDaemonOpts but
// lives in the observatory package so we don't create an import cycle
// (the universe public package imports observatory). The cli wires
// universe.StartArchitectDaemonWithOpts in via Server.SpawnArchitect.
type ArchitectSpawnOpts struct {
	ImageOverride string
	LogWriter     io.Writer
	OnProgress    func(event, detail string)
}

// ArchitectSpawnFunc is the signature the parent universe package
// implements. The observatory holds it as a function value injected
// at server construction time.
type ArchitectSpawnFunc func(ctx context.Context, opts ArchitectSpawnOpts) (string, error)

// architectSpawn is the in-memory record of an in-progress (or
// just-completed) architect daemon spawn. The observatory exposes its
// fields via /api/architect/status so the desktop app can render
// real-time progress instead of guessing from elapsed time.
//
// Lifecycle:
//   - nil               → no spawn has been attempted in this process
//   - InProgress=true   → goroutine is running, Event/Detail tick
//   - InProgress=false  → finished; Error is set on failure, ContainerID
//                          on success. Frontend can use this to render
//                          a final state until the next spawn is started.
type architectSpawn struct {
	StartedAt   time.Time
	FinishedAt  time.Time
	InProgress  bool
	Event       string // last progress event key (e.g. "image_building")
	Detail      string // last progress detail (free-form)
	Error       string // populated if the spawn failed
	ContainerID string // populated on success
	// Build log tail. Capped to keep the JSON response small.
	logBuffer *boundedBuffer
}

// snapshot returns a JSON-serialisable view of the spawn state. Safe
// to call from any goroutine.
func (s *architectSpawn) snapshot() map[string]interface{} {
	out := map[string]interface{}{
		"event":          s.Event,
		"detail":         s.Detail,
		"inProgress":     s.InProgress,
		"startedAt":      s.StartedAt.Format(time.RFC3339),
		"elapsedSeconds": int(time.Since(s.StartedAt).Seconds()),
	}
	if !s.FinishedAt.IsZero() {
		out["finishedAt"] = s.FinishedAt.Format(time.RFC3339)
	}
	if s.Error != "" {
		out["error"] = s.Error
	}
	if s.ContainerID != "" {
		out["containerId"] = s.ContainerID
	}
	if s.logBuffer != nil {
		out["logTail"] = s.logBuffer.String()
	}
	return out
}

// startArchitectAsync launches the daemon spawn in a goroutine and
// updates the server's spawn tracker as the universe code emits
// progress events. Returns immediately. Calling it again while a
// spawn is already in progress is a no-op (the existing spawn keeps
// running and the new caller subscribes to its progress via the
// status endpoint).
func (s *Server) startArchitectAsync(imageOverride string) error {
	if s.spawnArchitectFn == nil {
		return errSpawnNotWired
	}
	s.architectMu.Lock()
	if s.architectSpawnState != nil && s.architectSpawnState.InProgress {
		s.architectMu.Unlock()
		return nil
	}
	state := &architectSpawn{
		StartedAt:  time.Now(),
		InProgress: true,
		Event:      "queued",
		Detail:     "preparing to start architect",
		logBuffer:  newBoundedBuffer(64 * 1024), // 64 KB tail is plenty
	}
	s.architectSpawnState = state
	s.architectMu.Unlock()

	go func() {
		ctx := context.Background()
		_, err := s.spawnArchitectFn(ctx, ArchitectSpawnOpts{
			ImageOverride: imageOverride,
			LogWriter:     state.logBuffer,
			OnProgress: func(event, detail string) {
				s.architectMu.Lock()
				state.Event = event
				state.Detail = detail
				s.architectMu.Unlock()
			},
		})
		s.architectMu.Lock()
		state.InProgress = false
		state.FinishedAt = time.Now()
		if err != nil {
			state.Error = err.Error()
		}
		s.architectMu.Unlock()
	}()
	return nil
}

var errSpawnNotWired = errSpawn("architect spawn function not wired into observatory")

type errSpawn string

func (e errSpawn) Error() string { return string(e) }

// boundedBuffer is a process-local writer that keeps the most recent
// N bytes written to it. It's used to capture the architect build log
// tail without growing unbounded.
type boundedBuffer struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	limit int
}

func newBoundedBuffer(limit int) *boundedBuffer {
	if limit <= 0 {
		limit = 16 * 1024
	}
	return &boundedBuffer{limit: limit}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, err := b.buf.Write(p); err != nil {
		return 0, err
	}
	if b.buf.Len() > b.limit {
		// Trim to keep the most recent <limit> bytes.
		excess := b.buf.Len() - b.limit
		// Discard from the head by reading and dropping.
		_ = b.buf.Next(excess)
	}
	return len(p), nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
