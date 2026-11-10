package machine

type Kind string

const (
	Mac       Kind = "mac"
	Server    Kind = "server"
	Workspace Kind = "workspace"
)

type OS string

const (
	MacOS OS = "macos"
	Linux OS = "linux"
)

type System struct {
	OS      OS
	Version string
	Arch    string
}

type Ref struct {
	Name string
	// invariant: the name of the Mac a VM runs on, and empty for a Mac.
	On string
}

func (r Ref) Mac() string {
	if r.On == "" {
		return r.Name
	}
	return r.On
}

type Capabilities struct {
	// invariant: each capability is reported by the one contract that owns
	// the fact, never derived from a model name.
	Battery bool
	Lid     bool
	Thermal bool
	Jobs    bool
	Engine  bool
	// value: what provides each need of a call on this Mac, by its name and
	// how its presence is read; a need no entry names is not
	// judged here (a VM: its Mac provides it). Never sent: a far Mac judges
	// its own.
	Provides map[Need]Provider `json:"-"`
}

type Need string

const (
	// value: a Need is what a family of calls cannot run without on a Mac,
	// named by what it does, never by the app that does it; Hypervisor is
	// what makes, boots and deletes VMs and their images.
	Hypervisor Need = "hypervisor"
	// value: what carries a shared port to the owner's other devices.
	PrivateNetwork Need = "private-network"
	// value: the credential store an account is copied out of.
	Credentials Need = "credentials"
)

type Provider struct {
	// value: the program that provides the need, as the owner reads it.
	Name string
	// value: the catalogue item that installs it; empty when station
	// installs none.
	Item string
	// value: the presence read, asked only by the gate of a call that needs
	// it, at each such call: a call that declares no need reads nothing, and
	// a provider installed since, by this run or another process, is seen.
	Look func() bool `json:"-"`
}

func (c Capabilities) Lacks(n Need) (Provider, bool) {
	// invariant: a need no entry names is not judged here; one that names
	// it reads its presence now.
	p, judged := c.Provides[n]
	if !judged || p.Look == nil || p.Look() {
		return Provider{}, false
	}
	return p, true
}

type Hardware struct {
	// value: the model identifier the Mac reports ("Mac16,10"), its chip
	// ("Apple M4 Max"), each empty when it does not say.
	Model  string `json:",omitempty"`
	Chip   string `json:",omitempty"`
	Memory int64
	CPUs   int
	// value: the hardware address of the interface it answers on your LAN
	// with, what a Wake-on-LAN packet names; empty when it does not say.
	Ether string `json:",omitempty"`
}

type Profile struct {
	Ref      Ref
	Kind     Kind
	System   System
	Hardware Hardware
	Caps     Capabilities
}

type Verdict struct {
	OK  bool
	Why string
	// value: the command that makes the rule say yes, when one does.
	Fix string
}

var Yes = Verdict{OK: true}

func No(why string) Verdict { return Verdict{Why: why} }

type Applies func(Profile) Verdict

func All(rules ...Applies) Applies {
	return func(p Profile) Verdict {
		for _, rule := range rules {
			if v := rule(p); !v.OK {
				return v
			}
		}
		return Yes
	}
}

func Anywhere(Profile) Verdict {
	// value: the rule of a call every machine may ask; a call whose service
	// still refuses a machine in its own words carries it until its family
	// takes that rule.
	return Yes
}
