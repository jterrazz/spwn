//go:build !windows

package automation

import (
	"fmt"
	"os"
	"syscall"
)

func acquireDaemonLock(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if data, _ := os.ReadFile(path); len(data) > 0 {
			return nil, fmt.Errorf("another spwn automation daemon holds %s (pid %s); stop it before starting a new one", path, string(data))
		}
		return nil, fmt.Errorf("another spwn automation daemon holds %s; stop it before starting a new one", path)
	}
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	return f, nil
}

func releaseDaemonLock(f *os.File) {
	if f == nil {
		return
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}
