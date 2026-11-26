package wire

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	devicesjsonfile "github.com/jterrazz/studio-station/apps/engine/devices/adapter/jsonfile"
)

func doorMac(p planted, s *devicesjsonfile.Store, kind coremachine.Kind) machine {
	key := &heldKey{}
	key.once.Do(func() { key.key = p.key })
	return machine{profile: coremachine.Profile{Kind: kind, Ref: coremachine.Ref{Name: "mini"}}, stores: &stores{devices: s}, key: key}
}

func engineWith(t *testing.T, p planted, entries ...devices.Device) *devicesjsonfile.Store {
	t.Helper()
	s := p.store(nil)
	s.Sole(t.Context())
	if err := s.Change(t.Context(), func(b *devices.Book) error { b.Devices = append(b.Devices, entries...); return nil }); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestTheEngineLetsInTheDeviceOfTheKeyOfferedAlone(t *testing.T) {
	p := bookWithLaptop(t)
	m := doorMac(p, engineWith(t, p, p.laptop), coremachine.Computer)
	if name, err := m.admits(t.Context(), p.laptop.Key); err != nil || name != "laptop" {
		t.Fatalf("laptop's own key: %q %v", name, err)
	}
	own, err := p.key.Kept(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	line, err := devices.LineOf(own)
	if err != nil {
		t.Fatal(err)
	}
	for what, key := range map[string]string{"a stranger's key": "ssh-ed25519 AAAAstranger", "this Mac's own device key": line, "no key": ""} {
		if name, err := m.admits(t.Context(), key); !errors.Is(err, errNotInBook) {
			t.Errorf("%s: let in as %q, %v", what, name, err)
		}
	}
}

func TestAProcessThatIsNotTheEngineLetsNobodyIn(t *testing.T) {
	p := bookWithLaptop(t)
	engineWith(t, p, p.laptop)
	m := doorMac(p, p.store(nil), coremachine.Computer)
	if name, err := m.admits(t.Context(), p.laptop.Key); !errors.Is(err, errBookNotHeld) {
		t.Fatalf("a process that never took the engine's start read let laptop in: %q %v", name, err)
	}
}

func TestAMachineThatIsNotAMacOpensNoDoor(t *testing.T) {
	p := bookWithLaptop(t)
	m := doorMac(p, engineWith(t, p, p.laptop), coremachine.Workspace)
	if name, err := m.admits(t.Context(), p.laptop.Key); !errors.Is(err, errDoorNotHere) {
		t.Fatalf("a workspace: %q %v", name, err)
	}
}

func TestAnEntryWhoseSealFailsInTheHeldBookLetsItsKeyNowhere(t *testing.T) {
	p := bookWithLaptop(t)
	forged := p.laptop
	forged.Seal = slices.Clone(forged.Seal)
	forged.Seal[len(forged.Seal)-1] ^= 1
	s := engineWith(t, p, forged)
	if _, standing, ok := s.Held(t.Context()); !ok || standing != devices.Kept {
		t.Fatalf("the engine's own write stands %q %v, want kept", standing, ok)
	}
	m := doorMac(p, s, coremachine.Computer)
	if name, err := m.admits(t.Context(), p.laptop.Key); !errors.Is(err, errNotInBook) {
		t.Fatalf("an entry whose seal fails let its key in as %q, %v", name, err)
	}
}

// invariant: every Standing the devices package declares is read from its
// source, so a Standing added there fails here until the door decides it.
func standings(t *testing.T) []devices.Standing {
	t.Helper()
	dir := filepath.Join("..", "devices")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []devices.Standing
	fset := token.NewFileSet()
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			spec, ok := n.(*ast.ValueSpec)
			if !ok {
				return true
			}
			if typ, ok := spec.Type.(*ast.Ident); !ok || typ.Name != "Standing" {
				return true
			}
			for _, v := range spec.Values {
				lit, ok := v.(*ast.BasicLit)
				if !ok {
					t.Fatalf("a Standing whose value is not a literal: %T", v)
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatal(err)
				}
				out = append(out, devices.Standing(s))
			}
			return true
		})
	}
	return out
}

func TestTheDoorDecidesEveryStandingAndOpensOnKeptAlone(t *testing.T) {
	decided := map[devices.Standing]bool{devices.Kept: true, devices.Changed: false, devices.Unchecked: false, devices.Unreadable: false, devices.Newer: false}
	declared := standings(t)
	if len(declared) == 0 {
		t.Fatal("no Standing read from the devices package")
	}
	for _, s := range declared {
		opens, ok := decided[s]
		if !ok {
			t.Errorf("the door has not decided the standing %q", s)
			continue
		}
		if err := bookTrusted(s); (err == nil) != opens {
			t.Errorf("standing %q: %v, want open %v", s, err, opens)
		}
	}
	if len(declared) != len(decided) {
		t.Errorf("the devices package declares %v, the door decides %d", declared, len(decided))
	}
	if err := bookTrusted(devices.Standing("bogus")); err == nil {
		t.Error("a standing the devices package never declared opens the door")
	}
}
