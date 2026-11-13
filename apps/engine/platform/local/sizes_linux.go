package local

import (
	"context"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func lstat(path string) (k kind, bytes int64, dev uint64) {
	var st unix.Stat_t
	if unix.Lstat(path, &st) != nil {
		return kindOther, 0, 0
	}
	switch st.Mode & unix.S_IFMT {
	case unix.S_IFDIR:
		return kindDir, 0, st.Dev
	case unix.S_IFREG:
		return kindOther, cost(st.Size, st.Blocks*512), 0
	}
	return kindOther, 0, 0
}

func readDir(ctx context.Context, dir string, dev uint64, _ []byte, sub func(string)) int64 {
	f, err := os.Open(dir)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()
	fd := int(f.Fd())
	var total int64
	for ctx.Err() == nil {
		names, _ := f.Readdirnames(1024)
		if len(names) == 0 {
			return total
		}
		total += entries(fd, dir, dev, names, sub)
	}
	return total
}

func entries(fd int, dir string, dev uint64, names []string, sub func(string)) int64 {
	var total int64
	for _, name := range names {
		var st unix.Stat_t
		if unix.Fstatat(fd, name, &st, unix.AT_SYMLINK_NOFOLLOW) != nil {
			continue
		}
		switch st.Mode & unix.S_IFMT {
		case unix.S_IFREG:
			total += cost(st.Size, st.Blocks*512)
		case unix.S_IFDIR:
			if st.Dev == dev {
				sub(filepath.Join(dir, name))
			}
		}
	}
	return total
}
