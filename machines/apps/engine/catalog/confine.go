package catalog

import (
	"fmt"
	"strings"
)

// value: the folders and files of the home a setting's `file` method never writes:
// station's own state and configuration in every mode; in mode merge or
// block also the files another part of Machines writes, `~/.zshrc` (the
// main block and the packages' shell blocks) and `~/.ssh` (the authorized
// keys and their folder).
var (
	stationsOwn = []string{"~/.local/state/station", "~/.config/station"}
	othersOwn   = []string{"~/.zshrc", "~/.ssh"}
)

func Confined(to string, mode WriteMode) string {
	// invariant: why a setting may not write the file `to` in mode, empty
	// when it may. A file a setting writes is named under the home, `~/`
	// then segments none of them empty, `.` or `..`, so no name climbs out
	// of it; it is in none of station's own folders, and a fragment goes
	// into no file another part of Machines writes. Names are compared
	// without case, as a macOS volume opens them.
	rest, ok := strings.CutPrefix(to, "~/")
	if !ok {
		return fmt.Sprintf("file %s is not under your home: a setting writes only a path that starts with ~/", to)
	}
	for _, segment := range strings.Split(rest, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Sprintf("file %s has an empty, `.` or `..` segment: a setting writes only a plain path under your home", to)
		}
	}
	if at := within(to, stationsOwn); at != "" {
		return fmt.Sprintf("file %s is in %s, station's own: no setting writes there", to, at)
	}
	if at := within(to, othersOwn); at != "" && (mode == WriteBlock || mode == WriteMerge) {
		return fmt.Sprintf("file %s is in %s: another part of Machines writes it, so no fragment goes into it", to, at)
	}
	return ""
}

func within(to string, kept []string) string {
	for _, p := range kept {
		if strings.EqualFold(to, p) || len(to) > len(p) && strings.EqualFold(to[:len(p)+1], p+"/") {
			return p
		}
	}
	return ""
}
