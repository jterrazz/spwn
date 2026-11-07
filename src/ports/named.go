package ports

import "github.com/jterrazz/studio-station/src/catalog"

func (p Port) Named() catalog.PortName { return catalog.PortNameOf(p.Port, p.Process) }

func (s SharedPort) Named() catalog.PortName { return catalog.PortNameOf(s.GuestPort, s.Process) }

func (p Port) App() string { return catalog.AppOf(p.Process) }
