package calls_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/calls"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

// value: built by join, so no file holds a string a push protection reads
// as a token.
var token = "s" + "k-" + strings.Repeat("Ab3dE", 8)

var touched = catalog.Setting{Name: "dock-size", Effect: catalog.Touch{}}

type failing struct{ calls.Settings }

func (failing) Packages() []catalog.Setting { return []catalog.Setting{touched} }
func (failing) Lives(catalog.Setting) bool  { return true }
func (failing) Needs(catalog.Setting) string {
	return ""
}
func (failing) Admit(string, bool) error { return nil }
func (failing) Apply(context.Context, string, map[string]string, bool) (settings.Change, error) {
	return settings.Change{}, errors.New("defaults write com.apple.dock --token " + token + ": refused")
}

type kept struct{ contract.AppliedStore }

func (kept) Get(context.Context, string) (settings.Applied, error) { return settings.Applied{}, nil }

type undoing struct{ contract.Applier }

func (undoing) Kind() catalog.EffectKind { return catalog.EffectTouch }
func (undoing) Observe(context.Context, catalog.Setting) (settings.Observation, error) {
	return settings.Observation{Blind: true}, nil
}
func (undoing) Undo(context.Context, catalog.Setting) (settings.Change, error) {
	return settings.Change{}, errors.New("touch -t " + token + ": refused")
}

type prefs struct{}

func (prefs) Read(string, any) error  { return nil }
func (prefs) Write(string, any) error { return nil }

func deps() calls.Deps {
	return calls.Deps{
		Settings: failing{}, Applied: kept{}, OffThisMac: contract.OffHere(prefs{}),
		Appliers: contract.NewAppliers(undoing{}),
	}
}

func detailOf(t *testing.T, changes []settings.Change) string {
	t.Helper()
	if len(changes) != 1 || len(changes[0].Steps) != 1 {
		t.Fatalf("one failed step expected: %+v", changes)
	}
	return changes[0].Steps[0].Detail
}

func TestAFailedApplyAnswersItsToolsWordsScreened(t *testing.T) {
	changes, err := calls.Apply.Run(context.Background(), deps(), calls.ApplyIn{Names: []string{touched.Name}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := detailOf(t, changes); strings.Contains(got, token) || !strings.Contains(got, "defaults write com.apple.dock --token withheld") {
		t.Errorf("the failed step said %q", got)
	}
}

func TestAFailedOffAnswersItsToolsWordsScreened(t *testing.T) {
	changes, err := calls.Off.Run(context.Background(), deps(), calls.OffIn{Names: []string{touched.Name}, Confirm: "yes"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := detailOf(t, changes); strings.Contains(got, token) || !strings.Contains(got, "touch -t withheld") {
		t.Errorf("the failed step said %q", got)
	}
}
