package cliproject

import "golang.org/x/sys/unix"

const (
	ioctlGetTermios      = unix.TCGETS
	ioctlSetTermiosFlush = unix.TCSETSF
)
