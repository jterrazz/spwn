package local

import (
	"context"
	"encoding/binary"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	attrDirMountStatus = 0x8
	dirMountPoint      = 0x1
	objRegular         = 1
	objDir             = 2
)

type attrList struct {
	bitmapCount uint16
	reserved    uint16
	common      uint32
	volume      uint32
	dir         uint32
	file        uint32
	fork        uint32
}

func device(dev int32) uint64 {
	return uint64(uint32(dev)) //nolint:gosec // outside: dev_t is a 32-bit id on darwin; only its equality matters.
}

func lstat(path string) (k kind, bytes int64, dev uint64) {
	var st unix.Stat_t
	if unix.Lstat(path, &st) != nil {
		return kindOther, 0, 0
	}
	switch st.Mode & unix.S_IFMT {
	case unix.S_IFDIR:
		return kindDir, 0, device(st.Dev)
	case unix.S_IFREG:
		return kindOther, cost(st.Size, st.Blocks*512), 0
	}
	return kindOther, 0, 0
}

// outside: getattrlistbulk(2) packs one record per entry: its length, the
// attribute set it returned (5 words), the name's reference, the object type,
// then only the attributes that apply to that object: a directory's mount
// status, a file's allocated size and data length.
func readDir(ctx context.Context, dir string, dev uint64, buf []byte, sub func(string)) int64 {
	fd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return 0
	}
	defer func() { _ = unix.Close(fd) }()
	// hazard: a bulk entry reports a mount point as the directory it covers
	// (a simulator runtime under /Library/Developer/CoreSimulator/Volumes), so
	// another volume is caught on the opened directory's own device.
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil || device(st.Dev) != dev {
		return 0
	}
	attrs := attrList{
		bitmapCount: unix.ATTR_BIT_MAP_COUNT,
		common:      unix.ATTR_CMN_RETURNED_ATTRS | unix.ATTR_CMN_NAME | unix.ATTR_CMN_OBJTYPE,
		dir:         attrDirMountStatus,
		file:        unix.ATTR_FILE_ALLOCSIZE | unix.ATTR_FILE_DATALENGTH,
	}
	var total int64
	// invariant: a cancelled walk stops between two batches of one directory,
	// so a directory of 150 000 entries never holds a worker past cancel.
	for ctx.Err() == nil {
		n, _, errno := unix.Syscall6(unix.SYS_GETATTRLISTBULK, uintptr(fd), uintptr(unsafe.Pointer(&attrs)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, 0) //nolint:gosec,staticcheck // outside: x/sys wraps no getattrlistbulk; the raw call answers on every macOS station supports, and the list and buffer outlive it.
		if errno != 0 || n == 0 {
			return total
		}
		at := 0
		for range int(n) {
			rec := buf[at:]
			at += int(binary.LittleEndian.Uint32(rec))
			total += entry(dir, rec, sub)
		}
	}
	return total
}

func entry(dir string, rec []byte, sub func(string)) int64 {
	word := func(i int) uint32 { return binary.LittleEndian.Uint32(rec[i:]) }
	const returned, name, objType = 4, 24, 32
	dirAttrs, fileAttrs := word(returned+8), word(returned+12)
	field := objType + 4
	switch word(objType) {
	case objRegular:
		var alloc, data int64
		if fileAttrs&unix.ATTR_FILE_ALLOCSIZE != 0 {
			alloc = int64(binary.LittleEndian.Uint64(rec[field:])) //nolint:gosec // outside: the kernel's off_t, a signed size.
			field += 8
		}
		if fileAttrs&unix.ATTR_FILE_DATALENGTH != 0 {
			data = int64(binary.LittleEndian.Uint64(rec[field:])) //nolint:gosec // outside: the kernel's off_t, a signed size.
		}
		return cost(data, alloc)
	case objDir:
		if dirAttrs&attrDirMountStatus != 0 && word(field)&dirMountPoint != 0 {
			return 0
		}
		offset, length := int(int32(word(name))), int(word(name+4)) //nolint:gosec // outside: attrreference_t's offset is a signed int32.
		if length > 1 {
			sub(filepath.Join(dir, string(rec[name+offset:name+offset+length-1])))
		}
	}
	return 0
}
