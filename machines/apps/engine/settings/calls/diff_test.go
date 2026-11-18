package calls_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/calls"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type drifted struct{ contract.Applier }

func (drifted) Kind() catalog.EffectKind { return catalog.EffectTouch }
func (drifted) Observe(context.Context, catalog.Setting) (settings.Observation, error) {
	return settings.Observation{
		State:  catalog.StateDegraded,
		Diff:   "-  \"key\": \"set\"\n+  \"key\": \"" + token + "\"",
		Before: []settings.KeyValue{{Key: "api", Value: "https://bot:" + "hunter2@host/x"}},
		After:  []settings.KeyValue{{Key: "api", Value: token}},
	}, nil
}

func TestADiffAnswersTheMachinesValuesAndFileScreened(t *testing.T) {
	d := deps()
	d.Appliers = contract.NewAppliers(drifted{})
	out, err := calls.Diff.Run(context.Background(), d, calls.DiffIn{Name: touched.Name}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.Diff, token) || !strings.Contains(out.Diff, "+  \"key\": \"withheld\"") || !strings.Contains(out.Diff, "-  \"key\": \"set\"\n") {
		t.Errorf("the diff = %q; want its lines kept and the token withheld", out.Diff)
	}
	if len(out.Before) != 1 || out.Before[0].Value != "https://withheld@host/x" || len(out.After) != 1 || out.After[0].Value != "withheld" {
		t.Errorf("before %+v, after %+v; want each value screened", out.Before, out.After)
	}
}
