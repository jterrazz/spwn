//go:build !darwin || !cgo

package darwin

type Kernel struct{}

func (Kernel) Counters() (Counters, error) { return basicCounters() }

func (Kernel) Processes() (Procs, error) { return Procs{}, errUnsupported }

func (Kernel) Process(int) (Procs, error) { return Procs{}, errUnsupported }

func (Kernel) Files(int) ([]string, error) { return nil, errUnsupported }
