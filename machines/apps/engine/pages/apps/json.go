package apps

import (
	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

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

func ListContract(schema string, reports []apps.Report, known []catalog.App) ListJSON {
	// invariant: every array is built with make, so an empty one encodes as [].
	body := ListJSON{Schema: schema, Findings: finding.Doc(apps.Findings(reports, machine.Ref{})), Apps: make([]AppJSON, 0, len(reports)), Elsewhere: len(apps.Elsewhere(known, reports))}
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

func installed(version string) *string {
	if version == "" {
		return nil
	}
	return &version
}
