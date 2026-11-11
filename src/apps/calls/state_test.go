package calls_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/adapter/fake"
	"github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call/lanes"
)

type probed struct {
	output string
	code   int
}

type machine struct {
	files  map[string]bool
	paths  map[string]string
	probes map[string]probed
}

func (machine) Now() time.Time { return time.Unix(0, 0) }

func (machine) Sleep(ctx context.Context, _ time.Duration) error {
	<-ctx.Done()
	return ctx.Err()
}

func (machine) Home() string              { return "/home/u" }
func (machine) Glob(string) []string      { return nil }
func (m machine) Exists(path string) bool { return m.files[path] }
func (machine) Keep(string, string) error { return nil }
func (m machine) LookPath(name string) (string, bool) {
	where, ok := m.paths[name]
	return where, ok
}

func (m machine) RunBriefly(_ context.Context, name string, args []string, _ time.Duration) (output string, code int, ran bool) {
	p, ok := m.probes[strings.Join(append([]string{name}, args...), " ")]
	return p.output, p.code, ok
}

func reportOf(t *testing.T, app catalog.App, manager *fake.State, m machine) apps.Report {
	method := app.Installs[0].Install.Method()
	installers := contract.Installers{method: fake.New(method, manager)}
	reports, err := calls.NewShelf(m, catalog.Catalog{Apps: []catalog.App{app}}, nil, calls.ShelfOptions{ThisMachine: catalog.Mac, Root: "/", Installers: installers, Pool: lanes.NewPool(context.Background(), m, 16, 1)}).Inventory(context.Background())
	if err != nil || len(reports) != 1 {
		t.Fatalf("inventory: %v %v", reports, err)
	}
	return reports[0]
}

func java(output string, code int) machine {
	return machine{paths: map[string]string{"java": "/usr/bin/java"}, probes: map[string]probed{"/usr/bin/java -version": {output, code}}}
}

var javaApp = catalog.App{Name: "java", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac, catalog.Linux, catalog.MacOS}, Install: catalog.CaskInstall{Token: "temurin"}}}, Provides: []string{"java"}, Version: catalog.VersionRead{Args: []string{"java", "-version"}, Pattern: `version "([^"]+)"`}}

func TestStateAnAppPresentWithoutItsManagerIsDegraded(t *testing.T) {
	app := catalog.App{Name: "keynote", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "keynote"}}}, Bundle: "Keynote"}

	got := reportOf(t, app, &fake.State{Absent: true}, machine{files: map[string]bool{"/Applications/Keynote.app": true}})

	if got.State != catalog.StateDegraded || got.Detail != "installed, cask not installed" || got.Managed {
		t.Fatalf("report %+v", got)
	}
}

func TestStateAStockShimWhoseProbeFailsIsAbsent(t *testing.T) {
	got := reportOf(t, javaApp, &fake.State{}, java("The operation couldn't be completed. Unable to locate a Java Runtime.", 1))

	if got.State != catalog.StateAbsent {
		t.Fatalf("report %+v", got)
	}
}

func TestStateAShimThatAnswersIsDegraded(t *testing.T) {
	got := reportOf(t, javaApp, &fake.State{}, java(`openjdk version "21.0.5" 2024-10-15`, 0))

	if got.State != catalog.StateDegraded || got.Version != "21.0.5" || got.Detail != "on PATH, not managed by cask" {
		t.Fatalf("report %+v", got)
	}
}

func TestStateAManagerHoldsWhatNoAppOfTheCatalogInstalls(t *testing.T) {
	// invariant: what a manager's methods list beyond the catalog's own
	// apps is the owner's, by key; a list that cannot be read is unread.
	manager := catalog.App{Name: "casks", Kind: catalog.KindManager, Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.ScriptInstall{URL: "https://example.com/install.sh"}}}}
	tool := catalog.App{Name: "tool", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "tool"}}}}
	cat := catalog.Catalog{
		Apps:  []catalog.App{manager, tool},
		Items: []catalog.Placed{{Item: catalog.Item{Name: "casks", Kind: catalog.KindManager, Manages: []catalog.ManagerKey{catalog.ManagerKey(catalog.MethodCask)}}}, {Item: catalog.Item{Name: "tool"}}},
	}
	for _, c := range []struct {
		casks  *fake.State
		held   []string
		unread bool
	}{
		{&fake.State{Installed: map[string]string{"tool": "1.0.0", "editor": "2.0.0", "browser": "3.0.0"}}, []string{"browser", "editor"}, false},
		{&fake.State{Installed: map[string]string{"tool": "1.0.0"}}, nil, false},
		{&fake.State{Absent: true}, nil, true},
	} {
		installers := contract.Installers{
			catalog.MethodScript: fake.New(catalog.MethodScript, &fake.State{Installed: map[string]string{"https://example.com/install.sh": "1.0.0"}}),
			catalog.MethodCask:   fake.New(catalog.MethodCask, c.casks),
		}
		m := machine{}
		reports, err := calls.NewShelf(m, cat, nil, calls.ShelfOptions{ThisMachine: catalog.Mac, Root: "/", Installers: installers, Pool: lanes.NewPool(context.Background(), m, 16, 1)}).Inventory(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range reports {
			if r.App.Name == "tool" && (len(r.Holds) > 0 || r.HoldsUnread) {
				t.Errorf("an app that manages nothing holds %v, unread %v", r.Holds, r.HoldsUnread)
			}
			if r.App.Name == "casks" && (!slices.Equal(r.Holds, c.held) || r.HoldsUnread != c.unread) {
				t.Errorf("the manager holds %v, unread %v; want %v, unread %v", r.Holds, r.HoldsUnread, c.held, c.unread)
			}
		}
	}
}

func TestStateAnAppOfSeveralPackagesIsHeldOnlyWhileAllAre(t *testing.T) {
	app := catalog.App{Name: "foo", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.AptInstall{Packages: []string{"foo", "libfoo"}}}}}

	if got := reportOf(t, app, &fake.State{Installed: map[string]string{"foo": "1.0"}}, machine{}); got.State != catalog.StateAbsent {
		t.Errorf("foo without libfoo reads %+v; want absent", got)
	}
	if got := reportOf(t, app, &fake.State{Installed: map[string]string{"foo": "1.0", "libfoo": "2.0"}}, machine{}); got.State != catalog.StateOK || got.Version != "1.0" {
		t.Errorf("foo with libfoo reads %+v; want ok at foo's version", got)
	}
}
