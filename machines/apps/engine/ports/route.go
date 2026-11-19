package ports

type Verified struct {
	// value: the route's own evidence is there on the machine.
	Seen bool
	// value: what was looked for and not found, when it is not seen.
	Missing string
}
