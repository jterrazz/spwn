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
	VMs     bool
	Engine  bool
	// value: the hypervisor a Mac runs its VMs with, by its adapter's name,
	// installed or not (VMs says which).
	Hypervisor string
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
