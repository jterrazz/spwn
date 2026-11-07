package menubar

import (
	"strings"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/page"
)

// value: the measures a page reads live (SPEC §B3), each keyed
// `<measure>:<machine>`: CPU's share, the memory used and free, the disk
// used and the network's rate, in and out together.
const (
	LiveCPU    = "cpu"
	LiveMemory = "memory"
	LiveFree   = "free"
	LiveDisk   = "disk"
	LiveNet    = "net"
)

func LiveKey(measure, machine string) string { return measure + ":" + machine }

// value: the machine a live key reads; empty for a key of no machine.
func LiveMachine(key string) string {
	_, machine, _ := strings.Cut(key, ":")
	return machine
}

type measures struct {
	// value: each measure now, as its last two readings give it (one for
	// memory and disk), and its series over the readings kept: CPU's share,
	// memory's and disk's use of their total from 0 to 100, the network in
	// kB/s; a measure not read yet is the zero Value.
	cpu, memory, free, disk, net page.Value
	cpus, memories, disks, nets  []int
}

func measured(readings []activity.Reading) measures {
	var m measures
	for i, r := range readings {
		if i > 0 {
			rates := activity.Between(readings[i-1].Sample, r.Sample)
			m.cpus = append(m.cpus, int(rates.CPU))
			m.nets = append(m.nets, int((rates.NetIn+rates.NetOut)/1e3))
		}
		if s := r.Sample; s.MemTotal > 0 {
			m.memories = append(m.memories, int(min(s.MemUsed, s.MemTotal)*100/s.MemTotal))
		}
		if s := r.Sample; s.DiskTotal > 0 {
			m.disks = append(m.disks, int((s.DiskTotal-s.DiskFree)*100/s.DiskTotal))
		}
	}
	n := len(readings)
	if n == 0 {
		return m
	}
	last := readings[n-1].Sample
	if n > 1 {
		rates := activity.Between(readings[n-2].Sample, last)
		m.cpu = page.Percent(int64(rates.CPU * 10))
		m.net = page.Size(int64(rates.NetIn + rates.NetOut))
		m.net.Unit = "/s"
	}
	if last.MemTotal > 0 {
		m.memory = page.Ratio(min(last.MemUsed, last.MemTotal), last.MemTotal)
		m.free = page.Size(last.MemTotal - min(last.MemUsed, last.MemTotal))
	}
	if last.DiskTotal > 0 {
		m.disk = page.Percent((last.DiskTotal - last.DiskFree) * 1000 / last.DiskTotal)
	}
	return m
}

// value: each live key's value from its machine's readings, as the page
// that lists the key draws it: the measure now led into its series, the
// free memory alone; a key whose measure is not read yet is left out.
func LiveValues(keys []string, readings func(machine string) []activity.Reading) map[string]page.Value {
	out := map[string]page.Value{}
	of := map[string]measures{}
	for _, k := range keys {
		machine := LiveMachine(k)
		m, ok := of[machine]
		if !ok {
			m = measured(readings(machine))
			of[machine] = m
		}
		var v page.Value
		switch strings.TrimSuffix(k, ":"+machine) {
		case LiveCPU:
			v = led(m.cpu, m.cpus)
		case LiveMemory:
			v = led(m.memory, m.memories)
		case LiveFree:
			v = m.free
		case LiveDisk:
			v = led(m.disk, m.disks)
		case LiveNet:
			v = led(m.net, m.nets)
		}
		if v.Kind != "" {
			out[k] = v
		}
	}
	return out
}

func led(now page.Value, series []int) page.Value {
	if now.Kind == "" {
		return now
	}
	return page.Spark(series, now)
}
