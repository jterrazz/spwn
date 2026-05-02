//go:build !windows

package automation

import (
	"fmt"
	"os"
	"syscall"
)

// acquireDaemonLock takes an exclusive flock on the lockfile,
// creating it if needed. Returns the open file handle; the caller
// keeps it alive for the daemon's lifetime — closing it (via
// releaseDaemonLock) releases the lock. Errors when another
// process already holds the lock so a second `spwn automation
// daemon` for the same project fails fast instead of interleaving
// receipts and clobbering the state cursor.
//
// flock is advisory and OS-released on process exit, so a
// SIGKILL'd daemon doesn't leave the lockfile in a stuck state —
// the next daemon's flock simply succeeds.
//
// Unix-only (Linux, macOS, BSD). Windows users would need
// LockFileEx; spwn is Docker-based and Docker on Windows runs in a
// Linux VM, so the daemon binary is Unix-only in practice.
func acquireDaemonLock(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		// Read the PID written by the holder for a friendlier hint.
		// Best effort — a torn write or a not-yet-written file is
		// fine, the user still sees the path and the action.
		if data, _ := os.ReadFile(path); len(data) > 0 {
			return nil, fmt.Errorf("another spwn automation daemon holds %s (pid %s); stop it before starting a new one", path, string(data))
		}
		return nil, fmt.Errorf("another spwn automation daemon holds %s; stop it before starting a new one", path)
	}
	// Write our PID for human debugging. flock is held while the
	// fd is open, so truncate/write here doesn't release it.
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	return f, nil
}

// releaseDaemonLock closes the lockfile handle, releasing the OS-
// level flock. Best-effort: errors logged but never propagated;
// the OS would release the lock on process exit anyway.
func releaseDaemonLock(f *os.File) {
	if f == nil {
		return
	}
	// Release explicitly so a long-lived daemon that calls Stop
	// without exiting (rare in production but possible in tests)
	// doesn't hold the lock past its useful life.
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}
