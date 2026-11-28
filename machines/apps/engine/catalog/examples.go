package catalog

import (
	"io/fs"
	"slices"
)

const projects = "projects"

func Examples() []string {
	// value: the projects the catalog makes a new one from, by the name
	// `--from` takes, in alphabetical order.
	entries, err := fs.ReadDir(builtin, projects)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	return names
}

func Example(name string) (fs.FS, bool) {
	// invariant: an example is the files of a project as they sit in its
	// folder: spwn.yaml, spwn.lock and the folders it lays out; the one named, or
	// false when the catalog holds none of that name.
	if !slices.Contains(Examples(), name) {
		return nil, false
	}
	sub, err := fs.Sub(builtin, projects+"/"+name)
	return sub, err == nil
}
