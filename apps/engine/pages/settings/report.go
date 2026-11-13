package settings

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
	settingsmodel "github.com/jterrazz/studio-station/apps/engine/settings"
)

type ItemJSON struct {
	// value: alias of station.status/v1's row, kept byte for byte one release.
	Name     string   `json:"name"`
	Category string   `json:"category"`
	LivesOn  []string `json:"places"`
	State    string   `json:"state"`
	Detail   string   `json:"detail"`
}

func ItemObject(r settingsmodel.Report) ItemJSON {
	livesOn := make([]string, len(r.Item.Kinds))
	for i, p := range r.Item.Kinds {
		livesOn[i] = catalog.StatusV1Kind(p)
	}
	return ItemJSON{Name: r.Item.Name, Category: string(r.Item.Category), LivesOn: livesOn, State: string(r.State), Detail: r.Detail}
}

func ReportColumns() []page.Column {
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100},
		{Key: "description", Kind: page.KindText, Keep: 20},
	}
}

const differs = "differs from station's"

func ReportRow(r settingsmodel.Report) page.Row {
	name := r.Item.Name
	row := page.Row{ID: name, Label: name, Mark: pages.MarkOf(r.State), Cells: []page.Value{page.Text(r.Item.Description)}}
	install := page.Cmd("station settings apply " + name)
	detail := r.Detail
	switch {
	case r.State == catalog.StatePending:
		row.Detail = page.Pending()
		return row
	case r.State == catalog.StateAbsent && detail == "" && !r.Applies:
		row.Reason, detail, row.Hint = page.Missing, "needs "+r.Item.NeedsSaid(), page.Cmd("station apps install "+r.Item.Requires)
	case r.State == catalog.StateAbsent:
		row.Hint = install
	case strings.HasPrefix(detail, differs):
		row.Reason, detail, row.Hint = page.Differs, differs, install
	case r.State == catalog.StateDegraded:
		row.Reason, row.Hint = page.Partial, install
	case r.State == catalog.StateUnknown:
		row.Reason = page.Late
	case r.State == catalog.StateBroken:
		row.Reason = page.Down
	case r.State == catalog.StateOK:
	}
	row.Detail = page.Text(detail)
	if (row.Mark == page.Attention || row.Mark == page.Broken) && row.Detail.IsNone() && row.Hint == nil {
		row.Detail = page.Text(string(r.State))
	}
	return row
}

func InputsForm(name string, inputs []catalog.Input, defaults map[string]string) page.Form {
	f := page.Form{Title: "Configure " + name}
	for _, in := range inputs {
		field := page.Field{Key: in.Name, Label: in.Label, Note: in.Help, Default: defaults[in.Name], Required: in.Required, Kind: page.FieldText}
		switch in.Kind {
		case catalog.InputPassword:
			field.Kind = page.FieldSecret
		case catalog.InputSelect:
			field.Kind = page.FieldSelect
			for _, o := range in.Options {
				field.Options = append(field.Options, page.Option{Value: o})
			}
		case catalog.InputConfirm:
			field.Kind = page.FieldConfirm
		case catalog.InputText:
		}
		f.Fields = append(f.Fields, field)
	}
	return f
}
