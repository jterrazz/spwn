package menubar

import (
	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

type measures struct {
	// value: each measure now, as its last two readings give it (one for
	// memory and disk), and its series over the readings kept: CPU's share,
	// memory's and disk's use of their total from 0 to 100, the network in
	// kB/s; a measure not read yet is the zero Value.
	cpu, memory, disk, net      page.Value
	cpus, memories, disks, nets []int
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
	}
	if last.DiskTotal > 0 {
		m.disk = page.Percent((last.DiskTotal - last.DiskFree) * 1000 / last.DiskTotal)
	}
	return m
}

func LiveValue(key string, readings []activity.Reading) page.Value {
	// value: a live key's value from its machine's readings, as the page
	// that lists the key draws it: the measure now led into its series, the
	// free memory in the gauge's own words; the zero Value while the measure is not read yet.
	m := measured(readings)
	measure, subject := engine.LiveOf(key)
	switch measure {
	case engine.LiveCPUMac:
		return ofTheMac(subject, readings)
	case engine.LiveCPU:
		return led(m.cpu, m.cpus)
	case engine.LiveMemory:
		return led(m.memory, m.memories)
	case engine.LiveFree:
		return freeWords(readings)
	case engine.LiveDisk:
		return led(m.disk, m.disks)
	case engine.LiveNet:
		return led(m.net, m.nets)
	}
	return page.Value{}
}

// value: SPEC B5.3: a VM's CPU as a share of the whole Mac, from that
// Mac's readings, led into its series; the zero Value while the Mac's last
// reading does not carry the VM.
func ofTheMac(vm string, readings []activity.Reading) page.Value {
	var series []int
	now := page.Value{}
	for _, r := range readings {
		now = page.Value{}
		for _, u := range r.VMs {
			if u.VM == vm {
				series = append(series, int(u.CPU))
				now = page.Percent(int64(u.CPU * 10))
			}
		}
	}
	return led(now, series)
}

func led(now page.Value, series []int) page.Value {
	if now.Kind == "" {
		return now
	}
	return page.Spark(series, now)
}
