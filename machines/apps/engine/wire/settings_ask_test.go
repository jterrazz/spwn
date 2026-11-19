package wire

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagesettings "github.com/jterrazz/studio-station/apps/engine/pages/settings"
)

type answers struct {
	form   page.Form
	values map[string]string
}

func (a *answers) Confirm(page.Confirm) (bool, error) { return false, nil }

func (a *answers) Form(f page.Form) (map[string]string, error) {
	a.form = f
	return a.values, nil
}

func TestASecretFieldCarriesNoDefaultButABlankAnswerTakesIt(t *testing.T) {
	inputs := []catalog.Input{
		{Name: "user", Label: "User", Kind: catalog.InputText},
		{Name: "password", Label: "Password", Kind: catalog.InputPassword},
	}
	defaults := map[string]string{"user": "agent", "password": "hunter2"}
	form := pagesettings.InputsForm("autologin", inputs, defaults)
	if got := form.Fields[0].Default; got != "agent" {
		t.Errorf("the text field's default is %q, want agent", got)
	}
	if got := form.Fields[1].Default; got != "" {
		t.Errorf("the secret field carries the default %q on a page", got)
	}
	if err := page.Check(page.Page{Actions: []page.Action{{ID: "configure", Form: &form}}}); err != nil {
		t.Errorf("the form is refused as a page: %v", err)
	}

	ask := &answers{values: map[string]string{"user": "agent", "password": ""}}
	got, err := askInputs(ask, "autologin", inputs, defaults)
	if err != nil || got["password"] != "hunter2" {
		t.Errorf("a blank secret answered %q, %v, want its default", got["password"], err)
	}
	if ask.form.Fields[1].Default != "" {
		t.Errorf("the asker was drawn the secret's default")
	}
	ask.values = map[string]string{"user": "agent", "password": "typed"}
	if got, _ := askInputs(ask, "autologin", inputs, defaults); got["password"] != "typed" {
		t.Errorf("a typed secret answered %q, want what was typed", got["password"])
	}
}
