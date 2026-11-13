package catalog

import (
	"cmp"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

type Catalog struct {
	Apps     []App
	Settings []Setting
	Images   []Image
	// value: every item as its file says it, and the layer it was read
	// from, in the lists' order: what the Machinefile's resolution reads.
	Items []Placed
}

// invariant: the built-in items and Machinefiles are embedded, so the
// binary reads no file of the clone: apps/engine/catalog/items/<category>/<name>.yaml,
// one file an item, and apps/engine/catalog/machinefiles/<name>.yaml.
//
//go:embed items machinefiles
var builtin embed.FS

// value: the layer every built-in file stands in, the weakest.

const BuiltinLayer = "built-in"

func Builtin() (Catalog, error) { return Load(builtin, "items") }

func BuiltinFiles() fs.FS {
	// value: the embedded files: items/ holds the catalog's, machinefiles/ the
	// built-in Machinefiles.
	return builtin
}

// value: an item, the file it was read from, its category folder and the
// layer it stands in.

type Placed struct {
	Item   Item
	Folder string
	File   string
	Layer  string
}

func (c Catalog) Item(name string) (Placed, bool) {
	for _, p := range c.Items {
		if p.Item.Name == name {
			return p, true
		}
	}
	return Placed{}, false
}

func (c Catalog) App(name string) (App, bool) {
	for _, t := range c.Apps {
		if t.Name == name {
			return t, true
		}
	}
	return App{}, false
}

func (c Catalog) Setting(name string) (Setting, bool) {
	for _, it := range c.Settings {
		if it.Name == name {
			return it, true
		}
	}
	return Setting{}, false
}

func Load(fsys fs.FS, root string) (Catalog, error) {
	items, err := Items(fsys, root, BuiltinLayer)
	if err != nil {
		return Catalog{}, err
	}
	return Assemble(items)
}

func Items(fsys fs.FS, root, layer string) ([]Placed, error) {
	// value: the items of one tree, `<root>/<category>/<name>.yaml`, each
	// decoded strictly, in the layer named.
	folders, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, err
	}
	var items []Placed
	for _, folder := range folders {
		if !folder.IsDir() {
			return nil, &FileError{File: path.Join(root, folder.Name()), Problem: "is no category folder"}
		}
		files, err := fs.ReadDir(fsys, path.Join(root, folder.Name()))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			file := path.Join(root, folder.Name(), f.Name())
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
				return nil, &FileError{File: file, Problem: "is no item file: an item is <name>.yaml"}
			}
			body, err := fs.ReadFile(fsys, file)
			if err != nil {
				return nil, err
			}
			p, err := Read(file, folder.Name(), layer, body)
			if err != nil {
				return nil, err
			}
			items = append(items, p)
		}
	}
	return items, nil
}

func Read(file, folder, layer string, body []byte) (Placed, error) {
	// value: one item file read, in its folder and layer.
	it, err := Decode(file, body)
	if err != nil {
		return Placed{}, err
	}
	return Placed{Item: it, Folder: folder, File: file, Layer: layer}, nil
}

func Over(layers ...[]Placed) []Placed {
	// invariant: the layers strongest first; an item of a stronger layer
	// replaces a same-name one of a weaker layer whole, never merged.
	var out []Placed
	seen := map[string]bool{}
	for _, layer := range layers {
		for _, p := range layer {
			if !seen[p.Item.Name] {
				seen[p.Item.Name] = true
				out = append(out, p)
			}
		}
	}
	return out
}

func Assemble(items []Placed) (Catalog, error) {
	// value: the catalog of the items given, judged whole.
	items = slices.Clone(items)
	slices.SortFunc(items, byCategory)
	c := Catalog{Images: builtinImages, Items: items}
	files := map[string]string{}
	for _, p := range items {
		files[p.Item.Name] = p.File
		if p.Item.Kind == KindSetting {
			category, ok := settingCategory(p.Folder)
			if !ok {
				return Catalog{}, &FileError{File: p.File, Problem: "folder " + p.Folder + " is no category of settings"}
			}
			c.Settings = append(c.Settings, p.Item.setting(category))
			continue
		}
		category, ok := appCategory(p.Folder)
		if !ok {
			return Catalog{}, &FileError{File: p.File, Problem: "folder " + p.Folder + " is no category of apps"}
		}
		t := p.Item.app(category)
		t.Installs = c.placed(p.Item)
		c.Apps = append(c.Apps, t)
	}
	if err := whole(items, files); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

// invariant: the apps and managers stand before the settings, each by
// category, in the order of the categories' tables, then by name; a folder
// that is no category sorts last, refused by the load.
func byCategory(a, b Placed) int {
	if (a.Item.Kind == KindSetting) != (b.Item.Kind == KindSetting) {
		if a.Item.Kind == KindSetting {
			return 1
		}
		return -1
	}
	return cmp.Or(cmp.Compare(a.rank(), b.rank()), cmp.Compare(a.Item.Name, b.Item.Name))
}

func (p Placed) rank() int {
	folders := make([]string, 0, len(Categories))
	if p.Item.Kind == KindSetting {
		for _, c := range SettingCategories {
			folders = append(folders, c.Folder())
		}
	} else {
		for _, c := range Categories {
			folders = append(folders, c.Folder())
		}
	}
	if i := slices.Index(folders, p.Folder); i >= 0 {
		return i
	}
	return len(folders)
}

// invariant: the items judged together, each refusal naming the file that
// says it: each name is one item's, and every name an item gives (requires,
// after, then, provided-by, renamed-from) is an item of the right kind.
func whole(items []Placed, files map[string]string) error {
	owner := owners{}
	for _, it := range CheckedSettings {
		owner[it.Name] = "checked setting"
	}
	for _, p := range items {
		what := "app"
		if p.Item.Kind == KindSetting {
			what = "setting"
		}
		if err := owner.claim(p.Item.Name, what); err != nil {
			return &FileError{File: files[p.Item.Name], Problem: err.Error()}
		}
	}
	if err := managers(items, files); err != nil {
		return err
	}
	renamed := map[string]string{}
	for _, p := range items {
		it := p.Item
		if problem := judged(it, owner, renamed); problem != "" {
			return &FileError{File: files[it.Name], Problem: problem}
		}
	}
	return nil
}

func judged(it Item, owner owners, renamed map[string]string) string {
	for _, old := range it.RenamedFrom {
		if owner[old] != "" {
			return fmt.Sprintf("%s is renamed from %q, which names an item still", it.Name, old)
		}
		if prev, ok := renamed[old]; ok {
			return fmt.Sprintf("%s and %s are both renamed from %q", prev, it.Name, old)
		}
		renamed[old] = it.Name
	}
	if d := it.Deprecated; d != nil && d.Use != "" && owner[d.Use] != "app" && owner[d.Use] != "setting" {
		return fmt.Sprintf("%s says to use %q, which is no item", it.Name, d.Use)
	}
	for _, a := range it.After {
		if owner[a] != "app" && owner[a] != "setting" {
			return fmt.Sprintf("%s comes after %q, which is no item", it.Name, a)
		}
	}
	if it.Kind == KindSetting {
		for _, r := range slices.Concat(it.Requires, it.RequiresOneOf) {
			if owner[r] != "app" {
				return fmt.Sprintf("setting %s requires %q, which is no app", it.Name, r)
			}
		}
		return ""
	}
	for _, r := range it.Requires {
		if owner[r] != "app" {
			return fmt.Sprintf("%s requires %q, which is no app", it.Name, r)
		}
	}
	for _, then := range it.Then {
		if owner[then] != "setting" {
			return fmt.Sprintf("%s proposes %q, which is no setting", it.Name, then)
		}
	}
	for _, m := range it.Methods {
		if m.ProvidedBy != "" && owner[m.ProvidedBy] != "app" {
			return fmt.Sprintf("%s is provided by %q, which is no app", it.Name, m.ProvidedBy)
		}
	}
	return ""
}

// invariant: a method key has one manager, and a system's priority one
// manager at each rank, so the order a system installs by is declared once.
func managers(items []Placed, files map[string]string) error {
	by := map[ManagerKey]string{}
	type slot struct {
		system string
		at     int
	}
	at := map[slot]string{}
	for _, p := range items {
		it := p.Item
		for _, k := range it.Manages {
			if prev, ok := by[k]; ok {
				return &FileError{File: files[it.Name], Problem: fmt.Sprintf("%s and %s both manage %s", prev, it.Name, k)}
			}
			by[k] = it.Name
		}
		if it.Priority == nil {
			continue
		}
		for _, s := range []slot{{"macos", it.Priority.MacOS}, {"ubuntu", it.Priority.Ubuntu}, {"debian", it.Priority.Debian}} {
			if s.at == 0 {
				continue
			}
			if prev, ok := at[s]; ok {
				return &FileError{File: files[it.Name], Problem: fmt.Sprintf("%s and %s both stand at %d on %s", prev, it.Name, s.at, s.system)}
			}
			at[s] = it.Name
		}
	}
	return nil
}

// value: what each name of the catalog names: an app, a setting, a checked
// setting.
type owners map[string]string

func (o owners) claim(name, what string) error {
	if prev, ok := o[name]; ok {
		return fmt.Errorf("%q names both a %s and a %s", name, prev, what)
	}
	o[name] = what
	return nil
}

// value: a port an item serves on, and the item that says so.

type ItemPort struct {
	Item string
	PortHint
}

func (c Catalog) Ports() []ItemPort {
	// value: every port the catalog's items serve on, in the items' order:
	// the label a listening port is shown with, and what sharing it sets.
	var out []ItemPort
	for _, p := range c.Items {
		for _, h := range p.Item.Ports {
			out = append(out, ItemPort{Item: p.Item.Name, PortHint: h})
		}
	}
	return out
}
