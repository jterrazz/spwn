package images

import "github.com/jterrazz/studio-station/apps/engine/catalog"

type BuildRequest struct {
	Images []string
	Binary string
}

type Step struct{ Label string }

// value: the Machinefile an image is made from, as a build reads it: its
// name (the image's), the image its disk starts from, its system, what it
// bakes, the files of its chain the build VM reads (station's built-in
// ones aside, which the binary embeds), the lock its build installs, and
// the hash of all of them.

type Template struct {
	Name  string
	From  string
	OS    catalog.OS
	Bake  []string
	Files []TemplateFile
	Lock  []byte
	SHA   string
}

// value: a file of a template's chain, by its slash path under the
// owner's folder (`machinefiles/<name>.yaml`).

type TemplateFile struct {
	Path string
	Body []byte
}
