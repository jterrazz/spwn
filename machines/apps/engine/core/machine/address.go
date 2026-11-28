package machine

import "strings"

func (r Ref) Address() string {
	// value: a machine's full address, `<vm>.<computer>` for a VM and its
	// bare name for a computer; a name is lower-case words joined by dashes,
	// never a dot, so the one dot of an address is the seam between the two.
	if r.On == "" {
		return r.Name
	}
	return r.Name + "." + r.On
}

type BadAddressError struct{ Address string }

func (e *BadAddressError) Error() string {
	return "\"" + e.Address + "\" is no machine's address: a machine is <name>, a VM of a computer <vm>.<computer>"
}

func ParseAddress(s string) (Ref, error) {
	// invariant: a bare name is a Ref with no computer, for the Router to
	// place; `<vm>.<computer>` is that VM of that computer; anything with an
	// empty half or a second dot is refused, never guessed.
	vm, on, dotted := strings.Cut(s, ".")
	switch {
	case !dotted && s != "":
		return Ref{Name: s}, nil
	case vm == "" || on == "" || strings.Contains(on, "."):
		return Ref{}, &BadAddressError{Address: s}
	}
	return Ref{Name: vm, On: on}, nil
}
