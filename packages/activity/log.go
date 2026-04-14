package activity

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"spwn.sh/packages/paths"
)

var mu sync.Mutex

// Log appends an event to the activity log.
// The ID and Timestamp are filled in if empty.
// Errors are best-effort: a failure to write must never block the caller.
func Log(e Event) {
	if e.ID == "" {
		e.ID = newID()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	if err := appendJSONL(paths.ActivityPath(), e); err != nil {
		fmt.Fprintf(os.Stderr, "activity: %v\n", err)
	}
}

func appendJSONL(path string, e Event) error {
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(e); err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	return nil
}

func newID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
