package cliproject

import "golang.org/x/sys/unix"

const (
	ioctlGetTermios      = unix.TIOCGETA
	ioctlSetTermiosFlush = unix.TIOCSETAF
)
