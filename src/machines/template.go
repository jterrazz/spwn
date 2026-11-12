package machines

import (
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: the Machinefile a workspace is made from (create --from), as the
// create reads it on its Mac: its name (the image's), its system, what its
// vm says of how it runs (zero where it says nothing), the folders of the
// Mac it shares live, the accounts it is shared, and the files the
// guest's boot phase reads: its chain and its lock, by their slash path
// under the owner's folder.

type Template struct {
	Name     string
	OS       catalog.OS
	Server   bool
	Lifetime Lifetime
	Network  string
	Size     Resources
	Mounts   []Mount
	Shares   []TemplateShare
	Files    []TemplateFile
}

type TemplateShare struct {
	Account string
	For     time.Duration
}

type TemplateFile struct {
	Path string
	Body []byte
}

// value: what a workspace's first start did of its Machinefile: the boot
// phase's apply (Failed, its first line, when it did not end well), each
// account shared, and the image's drift from the Machinefile now, one
// line an item (hard case 17).

type FirstStart struct {
	Template string
	Failed   string
	Shares   []FirstStartShare
	Drift    []string
}

type FirstStartShare struct {
	Account string
	For     time.Duration
	// value: why it was not shared; empty when it was.
	Failed string
}

type ServerTemplateError struct{ Name string }

func (e *ServerTemplateError) Error() string {
	return e.Name + " is a server's Machinefile: station sets up nothing inside a server"
}
