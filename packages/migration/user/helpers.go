package user

import "os"

// removeIfEmpty removes a directory only if it contains no entries.
// Best-effort — errors are swallowed so a migration can call this
// without guarding every call site.
func removeIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		_ = os.Remove(dir)
	}
}
