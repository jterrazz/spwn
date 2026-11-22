package linux

import (
	"errors"
	"testing"
)

func TestTheVersionIDIsReadFromTheOSRelease(t *testing.T) {
	cases := map[string]struct{ file, want string }{
		"ubuntu quoted": {"NAME=\"Ubuntu\"\nVERSION_ID=\"24.04\"\nVERSION_CODENAME=noble\nID=ubuntu\n", "24.04"},
		"debian bare":   {"PRETTY_NAME=\"Debian GNU/Linux 13 (trixie)\"\nVERSION_ID=13\nID=debian\n", "13"},
		"single quoted": {"VERSION_ID='22.04'\n", "22.04"},
		"absent":        {"NAME=Arch\nID=arch\n", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := parseVersionID([]byte(c.file)); got != c.want {
				t.Fatalf("version = %q, want %q", got, c.want)
			}
		})
	}
}

func TestTheVersionFallsBackToTheUsrLibOSRelease(t *testing.T) {
	read := func(path string) ([]byte, error) {
		if path == "/usr/lib/os-release" {
			return []byte("VERSION_ID=\"24.04\"\n"), nil
		}
		return nil, errors.New("absent")
	}
	if got := osVersion(read); got != "24.04" {
		t.Fatalf("version = %q, want 24.04", got)
	}
}

func TestTheVersionIsEmptyWhenNoOSReleaseReads(t *testing.T) {
	read := func(string) ([]byte, error) { return nil, errors.New("absent") }
	if got := osVersion(read); got != "" {
		t.Fatalf("version = %q, want empty", got)
	}
}
