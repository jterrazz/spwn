//go:build !darwin

package darwin

func basicCounters() (Counters, error) { return Counters{}, errUnsupported }
