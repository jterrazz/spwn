package api

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"
)

// ArchitectSpawnOpts mirrors architect.StartDaemonOpts but
// lives in the api package so we don't create an import cycle.
// The CLI wires architect.StartDaemonWithOpts in via
// Server.SpawnArchitect.
type ArchitectSpawnOpts struct {
	ImageOverride string
	LogWriter     io.Writer
	OnProgress    func(event, detail string)
}

// ArchitectSpawnFunc is the signature the parent world package
// implements. The the API holds it as a function value injected
// at server construction time.
type ArchitectSpawnFunc func(ctx context.Context, opts ArchitectSpawnOpts) (string, error)

type architectSpawn struct {
	StartedAt   time.Time
	FinishedAt  time.Time
	InProgress  bool
	Event       string
	Detail      string
	Error       string
	ContainerID string
	logBuffer *boundedBuffer
}

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
		logBuffer:  newBoundedBuffer(64 * 1024),
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

var errSpawnNotWired = errSpawn("architect spawn function not wired into the api server")

type errSpawn string

func (e errSpawn) Error() string { return string(e) }

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
		excess := b.buf.Len() - b.limit
		_ = b.buf.Next(excess)
	}
	return len(p), nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
