package local

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type Invoking struct {
	// value: the names of the variables sudo sets for the account that
	// typed its line, its name, uid and gid; their owner declares them and
	// wire hands them here.
	User, UID, GID string
}

type invoker struct {
	name, home string
	uid, gid   int
	chown      func(path string, uid, gid int) error
}

// invariant: under sudo (euid 0 and SUDO_USER an account other than root)
// station acts for the owner who typed the line: their home, their name,
// and what it writes under that home is theirs, so an admin apply records
// into the owner's state, never root's. Any other run has no invoker.
func invokerOf(vars map[string]string, names Invoking, euid int, lookup func(name string) (*user.User, error)) *invoker {
	name := vars[names.User]
	if euid != 0 || name == "" || name == "root" {
		return nil
	}
	u, err := lookup(name)
	if err != nil || u.HomeDir == "" {
		return nil
	}
	uid, uerr := strconv.Atoi(cmpOr(vars[names.UID], u.Uid))
	gid, gerr := strconv.Atoi(cmpOr(vars[names.GID], u.Gid))
	if uerr != nil || gerr != nil {
		return nil
	}
	return &invoker{name: name, home: filepath.Clean(u.HomeDir), uid: uid, gid: gid, chown: os.Lchown}
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (o *invoker) owns(path string) bool {
	return o != nil && strings.HasPrefix(filepath.Clean(path), o.home+string(filepath.Separator))
}

func (o *invoker) give(paths ...string) error {
	for _, p := range paths {
		if o.owns(p) {
			if err := o.chown(p, o.uid, o.gid); err != nil {
				return err
			}
		}
	}
	return nil
}

// value: the folders MkdirAll would make for path, the outermost first.
func missing(path string) []string {
	var out []string
	for p := filepath.Clean(path); ; p = filepath.Dir(p) {
		if _, err := os.Lstat(p); err == nil {
			break
		}
		out = append([]string{p}, out...)
		if p == filepath.Dir(p) {
			break
		}
	}
	return out
}
