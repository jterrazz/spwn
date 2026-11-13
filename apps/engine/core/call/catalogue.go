package call

import (
	"encoding/json"
	"slices"
	"strconv"
)

const CatalogueSchema = "station.calls/v1"

type Entry struct {
	Name string `json:"name"`
	// value: the schema of the call's output, `station.<name>/v<n>`.
	Schema  string   `json:"schema"`
	Section string   `json:"section,omitempty"`
	Where   string   `json:"where"`
	Confirm string   `json:"confirm,omitempty"`
	Errors  []string `json:"errors,omitempty"`
	In      string   `json:"in,omitempty"`
	Out     string   `json:"out,omitempty"`
	Watch   bool     `json:"watch,omitempty"`
}

type Catalogue struct {
	Schema string  `json:"schema"`
	Calls  []Entry `json:"calls"`
}

func schema(name string, document int) string {
	// value: the schema name of a call's output, after its registry name
	// and its document's version (SPEC §11.1): the one place a schema is
	// named.
	return "station." + name + "/v" + strconv.Itoa(max(document, 1))
}

func (r *Registry) Catalogue() ([]byte, error) {
	// value: calls.json, the registry's Meta as the API's catalogue: the
	// names a client calls, the menu bar's action ids, the palette's entries.
	c := Catalogue{Schema: CatalogueSchema, Calls: []Entry{}}
	for _, m := range r.Metas() {
		e := Entry{Name: m.Name, Schema: schema(m.Name, m.Document), Section: string(m.Section), Where: string(m.Where), Confirm: string(m.Confirm), In: m.In, Out: m.Out, Watch: m.Watch}
		for _, code := range m.Errors {
			e.Errors = append(e.Errors, string(code))
		}
		slices.Sort(e.Errors)
		c.Calls = append(c.Calls, e)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
