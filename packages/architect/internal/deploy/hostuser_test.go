package deploy

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type hostUserFixture struct {
	root, etc, mountinfo, log, stubs string
}

func newHostUserFixture(t *testing.T, chownExit int) hostUserFixture {
	t.Helper()
	f := hostUserFixture{root: t.TempDir(), stubs: t.TempDir()}
	f.etc = filepath.Join(f.root, "etc")
	f.mountinfo = filepath.Join(f.stubs, "mountinfo")
	f.log = filepath.Join(f.stubs, "log")
	uid, gid := strconv.Itoa(os.Getuid()), strconv.Itoa(os.Getgid())

	write := func(path, content string, mode os.FileMode) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(f.etc, "passwd"), "root:x:0:0:root:/root:/bin/sh\nubuntu:x:1000:1000::/home/ubuntu:/bin/sh\nspwn:x:"+uid+":"+gid+"::/home/spwn:/bin/sh\n", 0o644)
	write(filepath.Join(f.etc, "group"), "root:x:0:\nubuntu:x:1000:\nspwn:x:"+gid+":\n", 0o644)
	write(filepath.Join(f.root, "home", "spwn", "file"), "", 0o644)
	write(filepath.Join(f.root, "workspaces", "ab", "own.txt"), "", 0o644)
	mounts := map[string]string{"my ws": `my\040ws`, "a*": "a*", "t\tb": `t\011b`, `b\c`: `b\134c`}
	info := "1 0 8:1 / / rw - ext4 /dev/vda1 rw\n"
	for name, encoded := range mounts {
		write(filepath.Join(f.root, "workspaces", name, "host.txt"), "", 0o644)
		info += "2 1 8:2 /src " + filepath.Join(f.root, "workspaces") + "/" + encoded + " rw - ext4 /dev/vdb rw\n"
	}
	write(f.mountinfo, info, 0o644)
	for _, tool := range []string{"chown", "chgrp"} {
		write(filepath.Join(f.stubs, tool), "#!/bin/sh\nprintf '"+tool+" %s\\n' \"$@\" >> '"+f.log+"'\nexit "+strconv.Itoa(chownExit)+"\n", 0o755)
	}
	return f
}

func (f hostUserFixture) run(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sh", "-c", followHostUserScript, "sh", "1000", "1000", f.etc, f.mountinfo, f.root)
	cmd.Env = append(os.Environ(), "PATH="+f.stubs+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	if err != nil {
		t.Fatalf("run the script: %v: %s", err, out)
	}
	return 0
}

func (f hostUserFixture) changed(t *testing.T, tool string) []string {
	t.Helper()
	raw, err := os.ReadFile(f.log)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, line := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		arg, ok := strings.CutPrefix(line, tool+" ")
		if ok && arg != "-h" && arg != "1000" {
			paths = append(paths, arg)
		}
	}
	sort.Strings(paths)
	return paths
}

func TestFollowHostUserScriptReownsTheImageAndPrunesEveryMountByItsLiteralName(t *testing.T) {
	f := newHostUserFixture(t, 0)

	if code := f.run(t); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}

	var want []string
	for _, rel := range []string{"", "etc", "etc/group", "etc/passwd", "home", "home/spwn", "home/spwn/file", "workspaces", "workspaces/ab", "workspaces/ab/own.txt"} {
		want = append(want, filepath.Join(f.root, rel))
	}
	sort.Strings(want)
	for _, tool := range []string{"chown", "chgrp"} {
		if got := f.changed(t, tool); !reflect.DeepEqual(got, want) {
			t.Errorf("%s paths:\n got  %q\n want %q", tool, got, want)
		}
	}
	passwd, _ := os.ReadFile(filepath.Join(f.etc, "passwd"))
	group, _ := os.ReadFile(filepath.Join(f.etc, "group"))
	if want := "spwn:x:1000:1000::/home/spwn:/bin/sh\nroot:x:0:0:root:/root:/bin/sh\nubuntu:x:1000:1000::/home/ubuntu:/bin/sh\n"; string(passwd) != want {
		t.Errorf("passwd:\n got  %q\n want %q", passwd, want)
	}
	if want := "spwn:x:1000:\nroot:x:0:\nubuntu:x:1000:\n"; string(group) != want {
		t.Errorf("group:\n got  %q\n want %q", group, want)
	}
}

func TestFollowHostUserScriptExitsWithTheRemapCodeWhenAChownFails(t *testing.T) {
	f := newHostUserFixture(t, 1)

	if code := f.run(t); code != followHostUserRemapFailed {
		t.Fatalf("exit %d, want %d", code, followHostUserRemapFailed)
	}
}

func TestFollowHostUserScriptExitsWithoutTheRemapCodeWhenTheImageHasNoSpwnUser(t *testing.T) {
	f := newHostUserFixture(t, 0)
	if err := os.WriteFile(filepath.Join(f.etc, "passwd"), []byte("root:x:0:0:root:/root:/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := f.run(t); code == 0 || code == followHostUserRemapFailed {
		t.Fatalf("exit %d, want a requirement failure", code)
	}
}
