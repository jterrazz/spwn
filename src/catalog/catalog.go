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
	Profiles []Profile
}

// invariant: the built-in items are embedded, so the binary reads no file of
// the clone: src/catalog/items/<category>/<name>.yaml, one file an item.
//
//go:embed items
var builtin embed.FS

func Builtin() (Catalog, error) { return Load(builtin, "items") }

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

type placed struct {
	item   Item
	folder string
}

func Load(fsys fs.FS, root string) (Catalog, error) {
	folders, err := fs.ReadDir(fsys, root)
	if err != nil {
		return Catalog{}, err
	}
	var items []placed
	for _, folder := range folders {
		if !folder.IsDir() {
			return Catalog{}, &FileError{File: path.Join(root, folder.Name()), Problem: "is no category folder"}
		}
		files, err := fs.ReadDir(fsys, path.Join(root, folder.Name()))
		if err != nil {
			return Catalog{}, err
		}
		for _, f := range files {
			file := path.Join(root, folder.Name(), f.Name())
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
				return Catalog{}, &FileError{File: file, Problem: "is no item file: an item is <name>.yaml"}
			}
			body, err := fs.ReadFile(fsys, file)
			if err != nil {
				return Catalog{}, err
			}
			it, err := Decode(file, body)
			if err != nil {
				return Catalog{}, err
			}
			items = append(items, placed{item: it, folder: folder.Name()})
		}
	}
	slices.SortFunc(items, byCategory)
	c := Catalog{Images: builtinImages, Profiles: builtinProfiles}
	files := map[string]string{}
	for _, p := range items {
		file := path.Join(root, p.folder, p.item.Name+".yaml")
		files[p.item.Name] = file
		if p.item.Kind == KindSetting {
			category, ok := settingCategory(p.folder)
			if !ok {
				return Catalog{}, &FileError{File: file, Problem: "folder " + p.folder + " is no category of settings"}
			}
			c.Settings = append(c.Settings, p.item.setting(category))
			continue
		}
		category, ok := appCategory(p.folder)
		if !ok {
			return Catalog{}, &FileError{File: file, Problem: "folder " + p.folder + " is no category of apps"}
		}
		c.Apps = append(c.Apps, p.item.app(category))
	}
	if err := whole(items, files); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

// invariant: the apps and managers stand before the settings, each by
// category, in the order of the categories' tables, then by name; a folder
// that is no category sorts last, refused by the load.
func byCategory(a, b placed) int {
	if (a.item.Kind == KindSetting) != (b.item.Kind == KindSetting) {
		if a.item.Kind == KindSetting {
			return 1
		}
		return -1
	}
	return cmp.Or(cmp.Compare(a.rank(), b.rank()), cmp.Compare(a.item.Name, b.item.Name))
}

func (p placed) rank() int {
	folders := make([]string, 0, len(Categories))
	if p.item.Kind == KindSetting {
		for _, c := range SettingCategories {
			folders = append(folders, c.Folder())
		}
	} else {
		for _, c := range Categories {
			folders = append(folders, c.Folder())
		}
	}
	if i := slices.Index(folders, p.folder); i >= 0 {
		return i
	}
	return len(folders)
}

// invariant: the items judged together, each refusal naming the file that
// says it: each name is one item's, and every name an item gives (requires,
// after, then, provided-by, renamed-from) is an item of the right kind.
func whole(items []placed, files map[string]string) error {
	owner := owners{}
	for _, it := range CheckedSettings {
		owner[it.Name] = "checked setting"
	}
	for _, p := range items {
		what := "app"
		if p.item.Kind == KindSetting {
			what = "setting"
		}
		if err := owner.claim(p.item.Name, what); err != nil {
			return &FileError{File: files[p.item.Name], Problem: err.Error()}
		}
	}
	if err := managers(items, files); err != nil {
		return err
	}
	renamed := map[string]string{}
	for _, p := range items {
		it := p.item
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
func managers(items []placed, files map[string]string) error {
	by := map[ManagerKey]string{}
	type slot struct {
		system string
		at     int
	}
	at := map[slot]string{}
	for _, p := range items {
		it := p.item
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
