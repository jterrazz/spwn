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
	slices.SortStableFunc(items, byOrder)
	c := Catalog{Images: builtinImages}
	for _, p := range items {
		file := path.Join(root, p.folder, p.item.Name+".yaml")
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
	if err := c.whole(); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

// invariant: an ordered item comes before every unordered one, then by its
// order, then by name.
func byOrder(a, b placed) int {
	if (a.item.Order == 0) != (b.item.Order == 0) {
		if a.item.Order == 0 {
			return 1
		}
		return -1
	}
	return cmp.Or(cmp.Compare(a.item.Order, b.item.Order), cmp.Compare(a.item.Name, b.item.Name))
}

// invariant: the items judged together: each name is one item's, and every
// name an item gives (requires, then, provided-by) is an item of the right
// kind.
func (c Catalog) whole() error {
	owner := owners{}
	for _, it := range CheckedSettings {
		owner[it.Name] = "checked setting"
	}
	for _, t := range c.Apps {
		if err := owner.claim(t.Name, "app"); err != nil {
			return err
		}
	}
	for _, it := range c.Settings {
		if err := owner.claim(it.Name, "setting"); err != nil {
			return err
		}
	}
	for _, t := range c.Apps {
		for _, r := range t.Requires {
			if owner[r] != "app" {
				return fmt.Errorf("catalog: %s requires %q, which is no app", t.Name, r)
			}
		}
		for _, then := range t.Then {
			if owner[then] != "setting" {
				return fmt.Errorf("catalog: %s proposes %q, which is no setting", t.Name, then)
			}
		}
		for _, p := range t.Installs {
			if by, ok := p.Install.(ProvidedByInstall); ok && owner[by.Item] != "app" {
				return fmt.Errorf("catalog: %s is provided by %q, which is no app", t.Name, by.Item)
			}
		}
	}
	for _, it := range c.Settings {
		for _, r := range it.Needs() {
			if owner[r] != "app" {
				return fmt.Errorf("catalog: setting %s requires %q, which is no app", it.Name, r)
			}
		}
	}
	return nil
}

// value: what each name of the catalog names: an app, a setting, a checked
// setting.
type owners map[string]string

func (o owners) claim(name, what string) error {
	if prev, ok := o[name]; ok {
		return fmt.Errorf("catalog: %q names both a %s and a %s", name, prev, what)
	}
	o[name] = what
	return nil
}
