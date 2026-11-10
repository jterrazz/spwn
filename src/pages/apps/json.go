package apps

import (
	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
)

type AppStatusJSON struct {
	// value: alias of station.status/v1's row, kept byte for byte one release.
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Method   string   `json:"method"`
	LivesOn  []string `json:"places"`
	State    string   `json:"state"`
	Version  string   `json:"version"`
	Managed  bool     `json:"managed"`
	Provides []string `json:"provides"`
	Detail   string   `json:"detail"`
	Use      string   `json:"use"`
}

type AppJSON struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Method   string `json:"method"`
	State    string `json:"state"`
	// value: the version installed, null for an app that is not.
	Installed *string  `json:"installed"`
	Newest    string   `json:"newest,omitempty"`
	LivesOn   string   `json:"lives_on"`
	Provides  []string `json:"provides"`
	Detail    string   `json:"detail"`
}

type ListJSON struct {
	Schema   string         `json:"schema"`
	Findings []finding.JSON `json:"findings"`
	Apps     []AppJSON      `json:"apps"`
	// value: how many catalogue apps live on other machines.
	Elsewhere int `json:"not_for_this_machine"`
}

func ListContract(schema string, reports []apps.Report) ListJSON {
	// invariant: every array is built with make, so an empty one encodes as [].
	body := ListJSON{Schema: schema, Findings: finding.Doc(apps.Findings(reports, machine.Ref{})), Apps: make([]AppJSON, 0, len(reports)), Elsewhere: len(apps.Elsewhere(reports))}
	for _, r := range reports {
		detail := r.Detail
		if detail == "" {
			detail = r.Live
		}
		body.Apps = append(body.Apps, AppJSON{
			Name: r.App.Name, Category: string(r.App.Category), Method: string(r.Method),
			State: string(apps.StateOf(r)), Installed: installed(r.Version), Newest: r.Newest, LivesOn: apps.LivesOnToken(r.App.LivesOn()),
			Provides: append([]string{}, r.App.Provides...), Detail: detail,
		})
	}
	return body
}

func statusV1Category(c catalog.Category) string {
	// value: alias of v2's name of the command line category, which
	// station.status/v1 keeps byte for byte.
	if c == catalog.CategoryCLIApps {
		return "CLI Tools"
	}
	return string(c)
}

func AppObject(r apps.Report) AppStatusJSON {
	kinds := r.App.LivesOn()
	livesOn := make([]string, 0, len(kinds))
	for _, k := range kinds {
		livesOn = append(livesOn, catalog.StatusV1Kind(k))
	}
	detail := r.Detail
	if detail == "" {
		detail = r.Live
	}
	return AppStatusJSON{
		Name:     r.App.Name,
		Category: statusV1Category(r.App.Category),
		Method:   string(r.Method),
		LivesOn:  livesOn,
		State:    string(r.State),
		Version:  r.Version,
		Managed:  r.Managed,
		Provides: append([]string{}, r.App.Provides...),
		Detail:   detail,
		// value: the field status/v1's readers still parse; no app comes
		// with init, which installs none, so every row reads exploration.
		Use: "exploration",
	}
}

func installed(version string) *string {
	if version == "" {
		return nil
	}
	return &version
}
