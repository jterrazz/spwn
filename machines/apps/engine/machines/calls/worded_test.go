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
		{&model.TooBigError{Mac: "mac-mini", AskedGB: 60, LeftGB: 36}, NotEnoughMemory, "mac-mini", map[string]string{"asked_gb": "60", "left_gb": "36"}},
		{&model.NoImageError{Image: "base-ubuntu", Mac: "mac-mini"}, NoImage, "base-ubuntu", map[string]string{"mac": "mac-mini", "fetched": ""}},
		{&model.NoImageError{Image: "base-macos", Mac: "mac-mini", Fetched: true}, NoImage, "base-macos", map[string]string{"mac": "mac-mini", "fetched": "yes"}},
		{&model.EngineDownError{Mac: "mac-mini", Installed: true}, EngineDown, "mac-mini", map[string]string{"installed": "yes"}},
		{&model.BadLifetimeError{Value: "forever"}, BadLifetime, "forever", nil},
		{ServerNeedsNameRule, ServerNeedsName, "", nil},
		{CheckForServersRule, CheckForServers, "", nil},
		{ServerBridgedRule, ServerBridged, "", nil},
		{&model.BadCheckError{Value: "nope"}, BadCheck, "nope", nil},
		{&model.BadCreatorError{Session: "Bad"}, BadCreator, "Bad", nil},
		{&model.UnknownImageError{Image: "linux"}, UnknownImage, "linux", nil},
		{&model.ServerTemplateError{Name: "infra"}, ServerTemplate, "infra", nil},
		{&model.TemplateNameError{Name: "base", Template: "linux-dev"}, TemplateName, "base", map[string]string{"template": "linux-dev"}},
		{&model.BadNetworkError{Value: "bridged", Bridge: true}, BadNetwork, "bridged", map[string]string{"bridge": "yes"}},
		{&model.NoEngineError{Hypervisor: "hv", Package: "hv-app"}, NoHypervisor, "hv", map[string]string{"package": "hv-app"}},
		{&model.NoEngineError{Hypervisor: "hv"}, NoHypervisor, "hv", map[string]string{"package": ""}},
		{&model.BadNameError{Name: "Bad_Name"}, BadName, "Bad_Name", nil},
		{&model.BadPurposeError{Purpose: "one\ntwo"}, BadPurpose, "one\ntwo", nil},
		{&model.BadPurposeError{Secret: true}, BadPurpose, "", map[string]string{"secret": "yes"}},
		{&model.TooLittleMemoryError{AskedBytes: 1 << 30}, TooLittle, "1073741824", nil},
		{&model.BeingCreatedError{Name: "linux-4", Mac: "mac-mini"}, BeingCreated, "linux-4", map[string]string{"mac": "mac-mini"}},
		{&model.NameTakenError{Name: "infra", VM: true, On: "mac-mini"}, NameTaken, "infra", map[string]string{"by": "vm", "on": "mac-mini"}},
		{&model.NameTakenError{Name: "mac-mini"}, NameTaken, "mac-mini", map[string]string{"by": "added", "on": ""}},
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
