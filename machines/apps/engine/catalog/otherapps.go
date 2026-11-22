package catalog

import (
	"path"
	"strings"
)

var OtherAppsData = []string{"~/Library/Containers", "~/Library/Group Containers"} // value: the folders macOS keeps each app's own data in, from macOS 15 behind App Data Protection: a read by any other app shows the owner "Data Access Blocked" and answers nothing, so Machines never lists, sizes or erases what lies inside them.

func InOtherAppsData(pattern string) bool {
	// invariant: true when pattern, a `~/` path or glob, is one of
	// OtherAppsData, lies inside one, or matches a path that does, in any
	// letter case; an absolute path is first shown under `~/` by the caller.
	segments := strings.Split(strings.ToLower(path.Clean(pattern)), "/")
	for _, root := range OtherAppsData {
		guarded := strings.Split(strings.ToLower(root), "/")
		if len(segments) < len(guarded) {
			continue
		}
		inside := true
		for i, g := range guarded {
			if ok, _ := path.Match(segments[i], g); !ok {
				inside = false
				break
			}
		}
		if inside {
			return true
		}
	}
	return false
}
