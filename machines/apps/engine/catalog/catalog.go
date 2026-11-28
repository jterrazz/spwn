package catalog

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	catalogfiles "github.com/jterrazz/studio-station/catalog"
)

type Catalog struct {
	Apps     []App
	Settings []Setting
	Images   []Image
	// value: every package as its file says it, and the origin it was read
	// from, in the lists' order: what the Machinefile's resolution reads.
	Packages []Placed
}

// invariant: the built-in packages and Machinefiles are embedded by the catalog
// module, so the binary reads no file of the clone.
var builtin = catalogfiles.Files

// value: the origin every built-in file stands in, the weakest.

const BuiltinOrigin = "built-in"

func Builtin() (Catalog, error) {
	// value: the built-in catalog, compiled from the embedded files by `make
	// generate` (builtin_table.go), so a run decodes none of them; Load is
	// the strict read the table is compiled from and held to.
	return builtinCatalog()
}

func BuiltinFiles() fs.FS {
	// value: the embedded files: packages/ holds the catalog's, machines/ the
	// built-in Machinefiles.
	return builtin
}

// value: a package, the file it was read from, its category folder and the
// origin it stands in.

type Placed struct {
	Package Package
	Folder  string
	File    string
	Origin  string
}

func (c Catalog) Package(name string) (Placed, bool) {
	for _, p := range c.Packages {
		if p.Package.Name == name {
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
	pkgs, err := Packages(fsys, root, BuiltinOrigin)
	if err != nil {
		return Catalog{}, err
	}
	images, err := sizedImages(builtin)
	if err != nil {
		return Catalog{}, err
	}
	return assemble(pkgs, images)
}

func Packages(fsys fs.FS, root, origin string) ([]Placed, error) {
	// value: the packages of one tree, `<root>/<category>/<name>.yaml`, each
	// decoded strictly, in the origin named.
	folders, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, err
	}
	var pkgs []Placed
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
				return nil, &FileError{File: file, Problem: "is no package file: a package is <name>.yaml"}
			}
			body, err := fs.ReadFile(fsys, file)
			if err != nil {
				return nil, err
			}
			p, err := Read(file, folder.Name(), origin, body)
			if err != nil {
				return nil, err
			}
			pkgs = append(pkgs, p)
		}
	}
	return pkgs, nil
}

func Read(file, folder, origin string, body []byte) (Placed, error) {
	// value: one package file read, in its folder and origin.
	it, err := Decode(file, body)
	if err != nil {
		return Placed{}, err
	}
	return Placed{Package: it, Folder: folder, File: file, Origin: origin}, nil
}

func Over(sources ...[]Placed) []Placed {
	// invariant: the sources strongest first; a package of a stronger source
	// replaces a same-name one of a weaker source whole, never merged.
	var out []Placed
	seen := map[string]bool{}
	for _, origin := range sources {
		for _, p := range origin {
			if !seen[p.Package.Name] {
				seen[p.Package.Name] = true
				out = append(out, p)
			}
		}
	}
	return out
}

func Assemble(pkgs []Placed) (Catalog, error) {
	// value: the catalog of the packages given, judged whole, with the
	// built-in images.
	b, err := Builtin()
	if err != nil {
		return Catalog{}, err
	}
	return assemble(pkgs, b.Images)
}

func assemble(pkgs []Placed, images []Image) (Catalog, error) {
	pkgs = byCategory(pkgs)
	c := Catalog{Images: images, Packages: pkgs}
	files := map[string]string{}
	for _, p := range pkgs {
		files[p.Package.Name] = p.File
		if p.Package.Kind == KindSetting {
			category, ok := settingCategory(p.Folder)
			if !ok {
				return Catalog{}, &FileError{File: p.File, Problem: "folder " + p.Folder + " is no category of settings"}
			}
			c.Settings = append(c.Settings, p.Package.setting(category))
			continue
		}
		category, ok := appCategory(p.Folder)
		if !ok {
			return Catalog{}, &FileError{File: p.File, Problem: "folder " + p.Folder + " is no category of apps"}
		}
		t := p.Package.app(category)
		t.Installs = c.placed(p.Package)
		c.Apps = append(c.Apps, t)
	}
	if err := whole(pkgs, files); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

// invariant: the apps and managers stand before the settings, each by
// category, in the order of the categories' tables, then by name; a folder
// that is no category sorts last, refused by the load. Each package's rank
// is read once, never per comparison.
func byCategory(pkgs []Placed) []Placed {
	apps, settings := make([]string, 0, len(Categories)), make([]string, 0, len(SettingCategories))
	for _, c := range Categories {
		apps = append(apps, c.Folder())
	}
	for _, c := range SettingCategories {
		settings = append(settings, c.Folder())
	}
	order := make([]ranked, len(pkgs))
	for i, p := range pkgs {
		folders := apps
		setting := p.Package.Kind == KindSetting
		if setting {
			folders = settings
		}
		rank := slices.Index(folders, p.Folder)
		if rank < 0 {
			rank = len(folders)
		}
		order[i] = ranked{p: p, setting: setting, rank: rank}
	}
	slices.SortFunc(order, byRank)
	out := make([]Placed, len(order))
	for i, r := range order {
		out[i] = r.p
	}
	return out
}

type ranked struct {
	p       Placed
	setting bool
	rank    int
}

func byRank(a, b ranked) int {
	if a.setting != b.setting {
		if a.setting {
			return 1
		}
		return -1
	}
	return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(a.p.Package.Name, b.p.Package.Name))
}

// invariant: the packages judged together, each refusal naming the file that
// says it: each name is one package's, and every name a package gives (requires,
// after, then, provided-by, renamed-from) is a package of the right kind.
func whole(pkgs []Placed, files map[string]string) error {
	owner := owners{}
	for _, it := range CheckedSettings {
		owner[it.Name] = "checked setting"
	}
	for _, p := range pkgs {
		what := "app"
		if p.Package.Kind == KindSetting {
			what = "setting"
		}
		if err := owner.claim(p.Package.Name, what); err != nil {
			return &FileError{File: files[p.Package.Name], Problem: err.Error()}
		}
	}
	if err := managers(pkgs, files); err != nil {
		return err
	}
	renamed := map[string]string{}
	for _, p := range pkgs {
		it := p.Package
		if problem := judged(it, owner, renamed); problem != "" {
			return &FileError{File: files[it.Name], Problem: problem}
		}
	}
	return nil
}

func judged(it Package, owner owners, renamed map[string]string) string {
	for _, old := range it.RenamedFrom {
		if owner[old] != "" {
			return fmt.Sprintf("%s is renamed from %q, which names a package still", it.Name, old)
		}
		if prev, ok := renamed[old]; ok {
			return fmt.Sprintf("%s and %s are both renamed from %q", prev, it.Name, old)
		}
		renamed[old] = it.Name
	}
	if d := it.Deprecated; d != nil && d.Use != "" && owner[d.Use] != "app" && owner[d.Use] != "setting" {
		return fmt.Sprintf("%s says to use %q, which is no package", it.Name, d.Use)
	}
	for _, a := range it.After {
		if owner[a] != "app" && owner[a] != "setting" {
			return fmt.Sprintf("%s comes after %q, which is no package", it.Name, a)
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
	for i, m := range it.Methods {
		if m.ProvidedBy != "" && owner[m.ProvidedBy] != "app" {
			return fmt.Sprintf("%s is provided by %q, which is no app", it.Name, m.ProvidedBy)
		}
		for _, r := range m.Requires {
			if owner[r] != "app" {
				return fmt.Sprintf("%s's method %d requires %q, which is no app", it.Name, i+1, r)
			}
		}
	}
	return ""
}

// invariant: a method key has one manager, and a system's priority one
// manager at each rank, so the order a system installs by is declared once.
func managers(pkgs []Placed, files map[string]string) error {
	by := map[ManagerKey]string{}
	type slot struct {
		system string
		at     int
	}
	at := map[slot]string{}
	for _, p := range pkgs {
		it := p.Package
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

// value: a port a package serves on, and the package that says so.

type PackagePort struct {
	Package string
	PortHint
}

func (c Catalog) Ports() []PackagePort {
	// value: every port the catalog's packages serve on, in the packages' order:
	// the label a listening port is shown with, and what sharing it sets.
	var out []PackagePort
	for _, p := range c.Packages {
		for _, h := range p.Package.Ports {
			out = append(out, PackagePort{Package: p.Package.Name, PortHint: h})
		}
	}
	return out
}
