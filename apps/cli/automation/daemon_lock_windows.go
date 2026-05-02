//go:build windows

package automation

import (
	"fmt"
	"os"
)

// acquireDaemonLock is a no-op stub on Windows. spwn is Docker-
// based and the daemon binary is Unix-only in practice; Windows
// builds compile but skip the cross-process safety check. Users
// running multiple daemons on Windows are on their own.
func acquireDaemonLock(path string) (*os.File, error) {
	_ = path
	return nil, fmt.Errorf("automation daemon: cross-process lock not implemented on Windows")
}

// releaseDaemonLock is a no-op on Windows.
func releaseDaemonLock(_ *os.File) {}
