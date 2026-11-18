package local

import (
	"strconv"
	"syscall"
)

func device(st *syscall.Stat_t) string { return strconv.FormatUint(st.Dev, 10) }

// hazard: Linux's stat keeps no birth time, so a file there is told by its
// device and inode alone, and an inode freed may be given again at once;
// station's hypervisor runs on a Mac only.
func born(*syscall.Stat_t) string { return "" }
