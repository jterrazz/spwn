package calls

import (
	"errors"
	"maps"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

func TestARefusalOfACreateCarriesItsCodeAndItsMac(t *testing.T) {
	for _, c := range []struct {
		err     error
		code    fail.Code
		subject string
		detail  map[string]string
	}{
		{&model.TooBigError{Mac: "jterrazz-mac-mini", AskedGB: 60, LeftGB: 36}, NotEnoughMemory, "jterrazz-mac-mini", map[string]string{"asked_gb": "60", "left_gb": "36"}},
		{&model.NoImageError{Image: "station-linux", Mac: "jterrazz-mac-mini"}, NoImage, "station-linux", map[string]string{"mac": "jterrazz-mac-mini", "fetched": ""}},
		{&model.NoImageError{Image: "macos", Mac: "jterrazz-mac-mini", Fetched: true}, NoImage, "macos", map[string]string{"mac": "jterrazz-mac-mini", "fetched": "yes"}},
		{&model.EngineDownError{Mac: "jterrazz-mac-mini", Installed: true}, EngineDown, "jterrazz-mac-mini", map[string]string{"installed": "yes"}},
		{&model.BadLifetimeError{Value: "forever"}, BadLifetime, "forever", nil},
		{ServerNeedsNameRule, ServerNeedsName, "", nil},
		{CheckForServersRule, CheckForServers, "", nil},
		{ServerBridgedRule, ServerBridged, "", nil},
		{&model.BadCheckError{Value: "nope"}, BadCheck, "nope", nil},
		{&model.BadCreatorError{Session: "Bad"}, BadCreator, "Bad", nil},
		{&model.UnknownImageError{Image: "linux"}, UnknownImage, "linux", nil},
		{&model.ServerTemplateError{Name: "infra"}, ServerTemplate, "infra", nil},
		{&model.BadNetworkError{Value: "bridged", Bridge: true}, BadNetwork, "bridged", map[string]string{"bridge": "yes"}},
		{&model.NoEngineError{Hypervisor: "hv"}, NoHypervisor, "hv", nil},
		{&model.BadNameError{Name: "Bad_Name"}, BadName, "Bad_Name", nil},
		{&model.BadPurposeError{Purpose: "one\ntwo"}, BadPurpose, "one\ntwo", nil},
		{&model.BeingCreatedError{Name: "linux-4", Mac: "jterrazz-mac-mini"}, BeingCreated, "linux-4", map[string]string{"mac": "jterrazz-mac-mini"}},
		{&model.NameTakenError{Name: "infra", VM: true, On: "jterrazz-mac-mini"}, NameTaken, "infra", map[string]string{"by": "vm", "on": "jterrazz-mac-mini", "agent": ""}},
		{&model.NameTakenError{Name: "docs-bot", Agent: "docs-bot", On: "linux-1"}, NameTaken, "docs-bot", map[string]string{"by": "agent", "on": "linux-1", "agent": "docs-bot"}},
		{&model.NameTakenError{Name: "jterrazz-mac-mini"}, NameTaken, "jterrazz-mac-mini", map[string]string{"by": "added", "on": "", "agent": ""}},
	} {
		var got *fail.Error
		if !errors.As(Worded(c.err), &got) {
			t.Fatalf("%v: no code", c.err)
		}
		if got.Code != c.code || got.Subject != c.subject || !maps.Equal(got.Detail, c.detail) || !errors.Is(got, c.err) {
			t.Errorf("%v: got %s %q %v", c.err, got.Code, got.Subject, got.Detail)
		}
	}
	var coded *fail.Error
	if errors.As(Worded(errors.New("plain")), &coded) {
		t.Error("an error of no code was given one")
	}
	if errors.As(Worded(&model.KindRuleError{Why: CheckForServersRule.Why, Fix: "an edit's own"}), &coded) {
		t.Error("an edit's kind rule, no create's, was given a create's code")
	}
}
