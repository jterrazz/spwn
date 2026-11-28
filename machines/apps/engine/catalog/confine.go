package catalog

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// value: the folders and files of the home a setting's `file` method never writes:
// station's own state and configuration and the bare engine's launchd
// agent in every mode; in mode merge or block also the files another part
// of Machines writes, `~/.zshrc` (the main block and the packages' shell
// blocks) and `~/.ssh` (the authorized keys and their folder). Where a
// run's XDG variables move station's folders, its applier refuses those
// too.
var (
	stationsOwn = []string{"~/.local/state/station", "~/.config/station", EnginePlist("~")}
	othersOwn   = []string{"~/.zshrc", "~/.ssh"}
)

func Kept(mode WriteMode) []string {
	// value: the folders and files of the home Confined keeps a setting in
	// mode from, each as it names them under `~/`.
	if mode == WriteBlock || mode == WriteMerge {
		return append(slices.Clone(stationsOwn), othersOwn...)
	}
	return slices.Clone(stationsOwn)
}

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
	if mode == WriteBlock && !hashCommented(to) {
		names := make([]string, 0, len(hashNames))
		for _, n := range hashNames {
			names = append(names, strings.TrimPrefix(n, "/"))
		}
		return fmt.Sprintf("file %s is no file known to take `#` comments, which mark a block: mode block writes into a %s file, or one named %s", to, strings.Join(hashExtensions, ", "), strings.Join(names, ", "))
	}
	return ""
}

// value: the files whose format reads a line opening with `#` as a
// comment, by extension, then by name (each after its `/`): those alone
// take a block, whose markers are such lines.
var (
	hashExtensions = []string{".bash", ".cfg", ".conf", ".env", ".fish", ".ini", ".sh", ".toml", ".yaml", ".yml", ".zsh"}
	hashNames      = []string{"/.bash_profile", "/.bashrc", "/.curlrc", "/.editorconfig", "/.gitconfig", "/.gitignore", "/.inputrc", "/.npmrc", "/.profile", "/.wgetrc", "/.zprofile", "/.zshenv", "/config"}
)

func hashCommented(to string) bool {
	name := strings.ToLower(path.Base(to))
	return slices.Contains(hashNames, "/"+name) || slices.Contains(hashExtensions, path.Ext(name))
}

func within(to string, kept []string) string {
	for _, p := range kept {
		if strings.EqualFold(to, p) || len(to) > len(p) && strings.EqualFold(to[:len(p)+1], p+"/") {
			return p
		}
	}
	return ""
}
