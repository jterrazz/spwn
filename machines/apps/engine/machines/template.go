package machines

import (
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

// value: the Machinefile a workspace is made from (create --from), as the
// create reads it on its Mac: its name (the image's), its system, what its
// vm says of how it runs (zero where it says nothing), the folders of the
// Mac it shares live, the accounts it signs in to and is granted, and the files the
// guest's boot phase reads: its chain and its lock, by their slash path
// under the owner's folder.

type Template struct {
	Name     string
	OS       catalog.OS
	Server   bool
	Lifetime Lifetime
	Network  catalog.Network
	Size     Resources
	Mounts   []Mount
	// value: the programs it signs in to, and those this Mac's vault grants
	// it, by their wire names.
	SignIns, Grants []string
	Files           []TemplateFile
	// value: the names of the Machinefiles its chain reads, itself among
	// them: a VM named like one would read that file's overlay as its own.
	Chain []string
}

type TemplateFile struct {
	Path string
	Body []byte
}

// value: what a workspace's first start did of its Machinefile: the boot
// phase's apply (Failed, its first line, when it did not end well), each
// program granted, each sign-in left to the owner, and the image's drift
// from the Machinefile now, one line a package (hard case 17).

type FirstStart struct {
	Template string
	Failed   string
	Grants   []FirstStartGrant
	SignIns  []FirstStartSignIn
	Drift    []string
}

type FirstStartGrant struct {
	Program string
	// value: why it was not granted, the refusal's code and its subject,
	// worded by the binary that prints them; none when it was granted.
	Code    fail.Code `json:",omitempty"`
	Subject string    `json:",omitempty"`
}

// value: a program the Machinefile signs in to that the workspace is not
// signed in to, and the command that signs it in.

type FirstStartSignIn struct {
	Program, Command string
}

// value: a workspace named like a Machinefile its template's chain reads:
// its own changes, `<name>.local.yaml`, would change that file for every
// machine that follows it.

type TemplateNameError struct{ Name, Template string }

func (e *TemplateNameError) Error() string {
	return e.Name + " names a Machinefile of " + e.Template + "'s chain: a workspace's own changes, " + e.Name + ".local.yaml, would be those of every machine that follows " + e.Template
}

type ServerTemplateError struct{ Name string }

func (e *ServerTemplateError) Error() string {
	return e.Name + " is a server's Machinefile: station sets up nothing inside a server"
}
