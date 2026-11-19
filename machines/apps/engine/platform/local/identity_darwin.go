package local

import (
	"strconv"
	"syscall"
)

func device(st *syscall.Stat_t) string { return strconv.FormatInt(int64(st.Dev), 10) }

func born(st *syscall.Stat_t) string {
	return strconv.FormatInt(st.Birthtimespec.Sec, 10) + "." + strconv.FormatInt(st.Birthtimespec.Nsec, 10)
}
