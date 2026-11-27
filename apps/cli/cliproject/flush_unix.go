//go:build linux || darwin

package cliproject

import "golang.org/x/sys/unix"

// invariant: flushInput discards what was typed or pasted on the terminal before the ask, as tcsetattr(TCSAFLUSH) does, so the answer is only what is typed after the prompt.
func flushInput(fd int) error {
	t, err := unix.IoctlGetTermios(fd, ioctlGetTermios)
	if err != nil {
		return err
	}
	return unix.IoctlSetTermios(fd, ioctlSetTermiosFlush, t)
}
