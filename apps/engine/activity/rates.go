package activity

type Rates struct {
	// value: CPU is the busy share of the whole machine (0 to 100); the
	// others are bytes per second.
	CPU                                float64
	DiskRead, DiskWrite, NetIn, NetOut float64
}

func Between(a, b Sample) Rates {
	// invariant: a counter that went backwards (a reboot between a and b)
	// gives a zero rate, never a negative one.
	seconds := b.At.Sub(a.At).Seconds()
	if seconds <= 0 {
		return Rates{}
	}
	perSecond := func(from, to uint64) float64 {
		if to < from {
			return 0
		}
		return float64(to-from) / seconds
	}
	r := Rates{
		DiskRead: perSecond(a.DiskRead, b.DiskRead), DiskWrite: perSecond(a.DiskWritten, b.DiskWritten),
		NetIn: perSecond(a.NetIn, b.NetIn), NetOut: perSecond(a.NetOut, b.NetOut),
	}
	if b.CPUTotal > a.CPUTotal && b.CPUBusy >= a.CPUBusy {
		r.CPU = min(100, 100*float64(b.CPUBusy-a.CPUBusy)/float64(b.CPUTotal-a.CPUTotal))
	}
	return r
}
