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
// phase's apply (Failed, its first line, when it did not run at all), the
// packages the apply of the whole file installed or set (Applied) and those
// it left unfinished, why that apply answered for none (Unapplied, its
// first line, when it could not run at all), each program
// granted, each sign-in left to the owner, and the image's drift from the
// Machinefile now, one line a package (hard case 17).

type FirstStart struct {
	Template   string
	Failed     string
	Applied    []FirstStartApp `json:",omitempty"`
	Unfinished []FirstStartApp `json:",omitempty"`
	Unapplied  string          `json:",omitempty"`
	Grants     []FirstStartGrant
	SignIns    []FirstStartSignIn
	Drift      []string
}

// value: a package of the Machinefile its first start's apply carried out:
// its name, whether it is a setting, and, when it was not installed or
// set, why and the log its run kept.

type FirstStartApp struct {
	App     string
	Setting bool   `json:",omitempty"`
	Reason  string `json:",omitempty"`
	Log     string `json:",omitempty"`
}

// value: why a package of a create's apply is unfinished when the run
// said no words of it: it asks the owner something, which a create has no
// terminal for, or the apply's bound ended before it was carried out.
const (
	AsksYou     = "it asks you something, on a terminal"
	NotFinished = "the apply's time ran out before it"
)

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
	return e.Name + " is a server's Machinefile: Machines sets up nothing inside a server"
}
