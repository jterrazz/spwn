package local

import (
	"io/fs"
	"strconv"
	"syscall"
)

type FileID string

func IdentityOf(info fs.FileInfo) (FileID, bool) {
	// value: a FileID is what tells one file of a Mac from every other, as
	// its file system keeps it: its device and inode, and its birth time
	// where the file system keeps one. A file deleted and made again under
	// the same path has another; a rename keeps it.
	// invariant: a file whose system says no identity has none, and is never
	// taken for another; a FileID a fake file system carries is its own.
	switch sys := info.Sys().(type) {
	case FileID:
		return sys, sys != ""
	case *syscall.Stat_t:
		return FileID(strconv.FormatUint(sys.Ino, 10) + "@" + device(sys) + "@" + born(sys)), true
	}
	return "", false
}
