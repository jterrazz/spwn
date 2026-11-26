package calls

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/images"
)

// value: a build VM that takes every file and answers every line at once,
// the deadline of each line kept.
type boundVM struct{ until []time.Time }

func (*boundVM) Put(context.Context, Login, string, []byte, uint32) error { return nil }
func (*boundVM) Close() error                                             { return nil }
func (v *boundVM) Run(ctx context.Context, _ Login, _, _ string) (Ran, error) {
	until, _ := ctx.Deadline()
	v.until = append(v.until, until)
	return Ran{}, nil
}

// value: a failed apply over twelve packages as linux-bake.sh hands it
// back from the build VM: its page, then each line it marked failed after
// failedMark, and its stderr.
func applied(failed ...string) Ran {
	var out strings.Builder
	out.WriteString("# To install\n")
	for i := range 12 - len(failed) {
		out.WriteString("ok\tok-" + strconv.Itoa(i) + "\t1.0\tapt\n")
	}
	out.WriteString(failedMark + "\n")
	for _, p := range failed {
		out.WriteString(p + "\t\tapt\t~/.local/state/station/logs/" + p + ".log\n")
	}
	return Ran{Code: 1, Stdout: out.String(), Stderr: "Error: " + strconv.Itoa(len(failed)) + " of 12 packages failed\nHint: station machinefile apply\n"}
}

func TestAFailedImagePhaseNamesEachPackageItsApplyMarkedFailed(t *testing.T) {
	err := &images.BuildFailedError{Image: "base-ubuntu", Step: "the image phase", Cause: failedLines(applied("ripgrep", "linux-kernel", "build-essential", "playwright-deps"))}

	want := "image base-ubuntu: the image phase failed: 4 of 12 packages failed: ripgrep, linux-kernel, build-essential, playwright-deps\n" +
		"  ripgrep: ~/.local/state/station/logs/ripgrep.log\n" +
		"  linux-kernel: ~/.local/state/station/logs/linux-kernel.log\n" +
		"  build-essential: ~/.local/state/station/logs/build-essential.log\n" +
		"  playwright-deps: ~/.local/state/station/logs/playwright-deps.log"
	if err.Error() != want {
		t.Errorf("failed as\n%s\nwant\n%s", err.Error(), want)
	}
}

func TestAFailedImagePhaseNamesItsFirstEightAndCountsTheRest(t *testing.T) {
	many := make([]string, 0, 30)
	for i := range 30 {
		many = append(many, "pkg-"+strconv.Itoa(i))
	}
	out := applied(many...)
	out.Stderr = "Error: 30 of 42 packages failed\n"

	got := failedLines(out).Error()
	head, _, _ := strings.Cut(got, "\n")
	if head != "30 of 42 packages failed: pkg-0, pkg-1, pkg-2, pkg-3, pkg-4, pkg-5, pkg-6, pkg-7, and 22 more" || strings.Count(got, "\n") != 8 {
		t.Errorf("failed as\n%s\nwant the first 8 named, each on its line, and 22 counted", got)
	}
}

func TestAFailedImagePhaseThatMarkedNoLineIsToldByItsTail(t *testing.T) {
	if got := failedLines(Ran{Code: 2, Stderr: "Error: unknown phase\n"}); got != nil {
		t.Errorf("read %v; want no failed line, the tail kept", got)
	}
}

// value: a build VM whose guest answers the copy of the logs: each name
// with its log in base64, or a broken connection.
type loggedVM struct {
	boundVM
	logs   map[string]string
	broken error
	script string
}

func (v *loggedVM) Run(_ context.Context, _ Login, _, stdin string) (Ran, error) {
	v.script = stdin
	if v.broken != nil {
		return Ran{}, v.broken
	}
	var out strings.Builder
	for name, body := range v.logs {
		out.WriteString(name + " " + base64.StdEncoding.EncodeToString([]byte(body)) + "\n")
	}
	return Ran{Stdout: out.String()}, nil
}

// value: this Mac's files as a failed build writes them.
type notedMac struct {
	Machine
	files map[string]string
}

func (m *notedMac) Note(p string, data []byte) error {
	m.files[p] = string(data)
	return nil
}

func TestAFailedImagePhaseKeepsTheLogsOfWhatFailedOnThisMac(t *testing.T) {
	mac := &notedMac{files: map[string]string{}}
	s := &Builder{m: mac, o: BuilderOptions{Logs: "/home/you/.local/state/station/logs", Home: "/home/you"}}
	vm := &loggedVM{logs: map[string]string{"ripgrep": "E: ripgrep failed\n"}}
	err := failedLines(applied("ripgrep", "playwright-deps"))

	s.keepLogs(context.Background(), vm, "base-ubuntu", Login{User: "you"}, err)
	if mac.files["/home/you/.local/state/station/logs/station-build-base-ubuntu.ripgrep.log"] != "E: ripgrep failed\n" || len(mac.files) != 1 {
		t.Errorf("kept %v; want ripgrep's log alone, the one the guest held", mac.files)
	}
	if !strings.Contains(vm.script, "for p in 'ripgrep' 'playwright-deps'; do") || !strings.Contains(vm.script, "tail -c 65536") {
		t.Errorf("asked the guest\n%s\nwant the two failed lines' logs, each its last 64 KiB", vm.script)
	}
	if !strings.Contains(err.Error(), "\n  ripgrep: ~/.local/state/station/logs/station-build-base-ubuntu.ripgrep.log\n  playwright-deps: ~/.local/state/station/logs/playwright-deps.log") || !err.Kept() {
		t.Errorf("failed as\n%s\nwant ripgrep's copy on this Mac named, playwright-deps' own words kept", err.Error())
	}
}

func TestAGuestThatNoLongerAnswersLeavesTheBuildsOwnError(t *testing.T) {
	mac := &notedMac{files: map[string]string{}}
	s := &Builder{m: mac, o: BuilderOptions{Logs: "/home/you/.local/state/station/logs"}}
	err := failedLines(applied("ripgrep"))

	s.keepLogs(context.Background(), &loggedVM{broken: errors.New("the guest connection dropped")}, "base-ubuntu", Login{User: "you"}, err)
	if !strings.HasPrefix(err.Error(), "1 of 12 packages failed: ripgrep\n") || !strings.HasSuffix(err.Error(), "\n  the build VM's logs were not copied: the guest connection dropped") || err.Kept() || len(mac.files) != 0 {
		t.Errorf("failed as\n%s\nwant the failure named and the copy said not made", err.Error())
	}
}

func TestTheImagePhaseOutlastsItsPackagesAndItsLines(t *testing.T) {
	// invariant: the bake is given the packages' share and the run.image
	// lines' bound together, as a first start's boot phase is.
	vm := &boundVM{}
	s := &Builder{o: BuilderOptions{User: "admin"}}
	want := catalog.PackagesBound + catalog.RunBound
	ref, cancel := context.WithTimeout(context.Background(), want)
	defer cancel()
	due, _ := ref.Deadline()
	if _, err := s.bakeRun(context.Background(), vm, catalog.Image{Bake: "images/linux-bake.sh"}, images.Template{Name: "studio-linux"}, Login{User: "admin"}); err != nil {
		t.Fatal(err)
	}
	if off := vm.until[0].Sub(due); off < 0 || off > time.Minute {
		left := vm.until[0].Sub(due) + want
		t.Fatalf("the bake had %v; want %v, the packages' share and the lines' bound", left, want)
	}
}
