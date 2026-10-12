//go:build windows

package automation

import (
	"fmt"
	"os"
)

func acquireDaemonLock(path string) (*os.File, error) {
	_ = path
	return nil, fmt.Errorf("automation daemon: cross-process lock not implemented on Windows")
}

func releaseDaemonLock(_ *os.File) {}
