package finding

type FixJSON struct {
	// invariant: SPEC §11.1: the three ways a finding is fixed are always
	// there, null when unset; args go with a call alone.
	Call    *string           `json:"call"`
	Args    map[string]string `json:"args,omitempty"`
	Command *string           `json:"command"`
	Path    *string           `json:"path"`
}

type JSON struct {
	// value: one Needs-you line as every document carries it; machine is
	// said only where a document spans machines.
	Machine  *string `json:"machine,omitempty"`
	Severity string  `json:"severity"`
	Section  string  `json:"section"`
	Subject  string  `json:"subject"`
	Text     string  `json:"text"`
	Fix      FixJSON `json:"fix"`
}

func (f Fix) JSON() FixJSON {
	out := FixJSON{Call: set(f.Call), Command: set(f.Command), Path: set(f.Path)}
	if f.Call != "" && len(f.Args) > 0 {
		out.Args = f.Args
	}
	return out
}

func (f Finding) JSON() JSON {
	return JSON{Severity: string(f.Severity), Section: string(f.Section), Subject: f.Subject, Text: f.Text, Fix: f.Fix.JSON()}
}

func Doc(fs []Finding) []JSON {
	// invariant: one machine's findings, ranked, never null.
	out := make([]JSON, 0, len(fs))
	for _, f := range Rank(fs) {
		out = append(out, f.JSON())
	}
	return out
}

func AllMachinesDoc(fs []Finding) []JSON {
	// invariant: findings of many machines, ranked, each naming its machine.
	out := make([]JSON, 0, len(fs))
	for _, f := range Rank(fs) {
		j := f.JSON()
		j.Machine = set(f.Machine.Name)
		out = append(out, j)
	}
	return out
}

func set(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
