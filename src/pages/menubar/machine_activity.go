package menubar

import (
	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

// value: Activity's three measures of the machine, as its last two
// readings give them (its disk as the share used, vision panels c and
// f), each led into its series over the readings kept (0 to 100: CPU's
// share, memory's and disk's use of their total), so a tile draws its bar
// meter; pending until they are read, none while the machine does not run.
func activitySection(readings []activity.Reading, running bool, title, name string) page.Section {
	cpu, memory, disk := page.Pending(), page.Pending(), page.Pending()
	if !running {
		cpu, memory, disk = page.None(), page.None(), page.None()
	}
	m := measured(readings)
	cpus, memories, disks := m.cpus, m.memories, m.disks
	if len(readings) > 0 {
		memory, disk = page.None(), page.None()
		if m.cpu.Kind != "" {
			cpu = m.cpu
		}
		if m.memory.Kind != "" {
			memory = m.memory
		}
		if m.disk.Kind != "" {
			disk = m.disk
		}
	}
	if running {
		cpu, memory, disk = live(cpu, engine.LiveKey(engine.LiveCPU, name)), live(memory, engine.LiveKey(engine.LiveMemory, name)), live(disk, engine.LiveKey(engine.LiveDisk, name))
	}
	measure := func(id, label string, now page.Value, series []int) page.Row {
		row := page.Row{ID: id, Label: label, Cells: []page.Value{now}}
		if len(series) > 1 {
			row.Cells = append(row.Cells, page.Spark(series, page.None()))
		}
		return row
	}
	rows := []page.Row{measure("cpu", "CPU", cpu, cpus), measure("memory", "Memory", memory, memories), measure("disk", "Disk", disk, disks)}
	cols := append(columns(), page.Column{Key: "graph", Kind: page.KindSpark, Keep: 50})
	return page.Section{ID: SectionActivity, Title: title, Role: page.Summary, Columns: cols, Items: items(rows)}
}

// value: a Mac's Activity segment: its three measures over the readings
// kept, until the Activity lot draws its charts.
func macActivity(m Machine, h model.MacView) page.Section {
	s := activitySection(m.Samples, true, "Activity", h.Name)
	s.Segment = SegmentActivity
	return s
}
