package ports

import "strconv"

type VMFull struct {
	VM  string
	Max int
}

func (e *VMFull) Error() string {
	return e.VM + " has " + strconv.Itoa(e.Max) + " shared ports, the most a VM may"
}

type RangeFull struct{ Low, High int }

func (e *RangeFull) Error() string {
	return "every reserved port " + strconv.Itoa(e.Low) + "-" + strconv.Itoa(e.High) + " is taken"
}

func Reserve(p Policy, taken []Reservation, vm string, guestPort int, busy func(int) bool) (Reservation, error) {
	// invariant: a reservation is sticky on (vm, guest port): one already
	// taken is returned as it is; a new one takes the LOWEST port of the range
	// no reservation holds and nothing on the Mac listens on, at most
	// MaxPerVM per VM.
	held, mine := map[int]bool{}, 0
	for _, r := range taken {
		if r.VM == vm && r.GuestPort == guestPort {
			return r, nil
		}
		held[r.MacPort] = true
		if r.VM == vm {
			mine++
		}
	}
	if mine >= p.MaxPerVM {
		return Reservation{}, &VMFull{VM: vm, Max: p.MaxPerVM}
	}
	for hp := p.Low; hp <= p.High; hp++ {
		if held[hp] || (busy != nil && busy(hp)) {
			continue
		}
		return Reservation{VM: vm, GuestPort: guestPort, MacPort: hp}, nil
	}
	return Reservation{}, &RangeFull{Low: p.Low, High: p.High}
}
