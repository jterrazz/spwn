package linux

import (
	"bufio"
	"bytes"
	"strings"
)

func parseVersionID(osRelease []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(osRelease))
	for scanner.Scan() {
		value, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "VERSION_ID=")
		if ok {
			return strings.Trim(value, `"'`)
		}
	}
	return ""
}

func osVersion(read func(string) ([]byte, error)) string {
	for _, path := range []string{"/etc/os-release", "/usr/lib/os-release"} {
		if data, err := read(path); err == nil {
			return parseVersionID(data)
		}
	}
	return ""
}
