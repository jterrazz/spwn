package machinefile

import (
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

const ID page.ID = "machinefile"

type ValueJSON struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Layer string `json:"layer"`
	// value: the Machinefile or the item that says it, and its file.
	From string `json:"from"`
	File string `json:"file,omitempty"`
}

type FileJSON struct {
	Name  string `json:"name"`
	Layer string `json:"layer"`
	File  string `json:"file"`
}

type ExplainJSON struct {
	Schema      string      `json:"schema"`
	Machinefile string      `json:"machinefile"`
	Files       []FileJSON  `json:"files"`
	Values      []ValueJSON `json:"values"`
}

func Contract(name string, files []machinefile.Origin, values []machinefile.Value) ExplainJSON {
	body := ExplainJSON{Schema: "station.machinefile.explain/v1", Machinefile: name, Files: []FileJSON{}, Values: []ValueJSON{}}
	for _, f := range files {
		body.Files = append(body.Files, FileJSON{Name: f.Name, Layer: string(f.Layer), File: f.File})
	}
	for _, v := range values {
		body.Values = append(body.Values, ValueJSON{Key: v.Key, Value: v.Text, Layer: string(v.Origin.Layer), From: v.Origin.Name, File: v.Origin.File})
	}
	return body
}

func Explain(name, key string, values []machinefile.Value, thisMac page.Context) page.Page {
	// value: one row per value, its key, the value, and where it was read;
	// a key asked explains it and every key under it.
	cols := []page.Column{{Key: "key", Kind: page.KindListing, Keep: 100}, {Key: "value", Kind: page.KindText, Keep: 40}, {Key: "layer", Kind: page.KindText, Keep: 60}, {Key: "from", Kind: page.KindText, Keep: 50}}
	items := make([]page.Item, 0, len(values))
	for _, v := range values {
		row := page.Row{ID: v.Key, Label: v.Key, Cells: []page.Value{page.Text(v.Text), page.Text(string(v.Origin.Layer)), page.Text(v.Origin.Name)}}
		items = append(items, page.Item{Row: &row})
	}
	if len(items) == 0 {
		items = append(items, page.Item{Empty: &page.Empty{Text: name + " carries nothing yet"}})
	}
	command := "explain --machinefile " + name
	if key != "" {
		command = "explain " + key + " --machinefile " + name
	}
	return page.Seal(page.Page{ID: ID, Command: command, Context: thisMac, Sections: []page.Section{{ID: "values", Title: name, Columns: cols, Items: items}}})
}
