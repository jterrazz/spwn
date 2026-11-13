package activity

import (
	"slices"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

// value: the sections a page adds when it asks for a window: one line per
// measure with its graph, and the busiest processes one per row.
const (
	Graphs    = "graphs"
	Processes = "processes"
)

func graphs(v View) []page.Section {
	var out []page.Section
	if lines := measures(v); len(lines) > 0 {
		out = append(out, page.Section{ID: Graphs, Uncounted: true, Columns: graphColumns(), Items: lines})
	}
	if procs := processRows(v); len(procs) > 0 {
		out = append(out, page.Section{ID: Processes, Title: "Top processes", Uncounted: true, Columns: processColumns(), Items: procs})
	}
	return out
}

func graphColumns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindText, Keep: 100}, {Key: "graph", Kind: page.KindSpark, Keep: 50}}
}

func measures(v View) []page.Item {
	// invariant: SPEC §6.4: a sparkline for CPU, memory, swap, disk read and
	// write, network in and out, each point 0 to 100 of its own scale, and a
	// bar for the disk's free space. A measure with no points has no graph,
	// never a flat line of zeros; a VM read from its Mac has its process's
	// CPU and memory only.
	r := v.Report
	s := r.Last.Sample
	if r.Silent {
		return nil
	}
	lines := series(r.Graph)
	line := func(id, label string, words ...string) page.Item {
		row := page.Row{ID: id, Label: label, Cells: []page.Value{page.Texts(words...)}}
		if len(lines[id]) > 0 {
			row.Cells = append(row.Cells, page.Spark(lines[id], page.None()))
		}
		return page.Item{Row: &row}
	}
	if run := r.Last.Run; run != nil {
		return []page.Item{
			line("cpu", "CPU", activity.Percent(run.CPU)+" of "+v.On),
			line("memory", "Memory", activity.Memory(run.Memory)+" resident"),
		}
	}
	rated := r.Window > 0 && counted(s)
	rate := func(perSecond float64) string {
		if !rated {
			return "?"
		}
		return activity.Rate(perSecond)
	}
	cpu := "?"
	if rated {
		cpu = activity.Percent(r.Rates.CPU)
	}
	out := []page.Item{
		line("cpu", "CPU", cpu+" of "+strconv.Itoa(s.CPUs)+" CPU", "load "+activity.Load(s.Load)),
		line("memory", "Memory", memory(s), "pressure "+pressure(s.Pressure)),
		line("swap", "Swap", activity.Memory(s.SwapUsed)+" used"),
		line("disk-read", "Disk read", rate(r.Rates.DiskRead), activity.Bytes(s.DiskRead)+" since boot"),
		line("disk-write", "Disk write", rate(r.Rates.DiskWrite), activity.Bytes(s.DiskWritten)+" since boot"),
		line("net-in", "Net in", rate(r.Rates.NetIn), activity.Bytes(s.NetIn)+" since boot"),
		line("net-out", "Net out", rate(r.Rates.NetOut), activity.Bytes(s.NetOut)+" since boot"),
	}
	if s.DiskTotal > 0 {
		free := page.Row{ID: "disk-free", Label: "Disk free", Cells: []page.Value{
			page.Text(activity.Bytes(s.DiskFree) + " free of " + activity.Bytes(s.DiskTotal)), page.Ratio(s.DiskTotal-s.DiskFree, s.DiskTotal),
		}}
		out = append(out, page.Item{Row: &free})
	}
	return out
}

func series(points []activity.Sample) map[string][]int {
	out := map[string][]int{}
	if len(points) < 2 {
		return out
	}
	rates := make([]activity.Rates, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		rates = append(rates, activity.Between(points[i-1], points[i]))
	}
	scaled := func(of func(activity.Rates) float64) []int {
		top := 0.0
		for _, r := range rates {
			top = max(top, of(r))
		}
		line := make([]int, 0, len(rates))
		for _, r := range rates {
			if top <= 0 {
				line = append(line, 0)
				continue
			}
			line = append(line, int(100*of(r)/top))
		}
		return line
	}
	for _, r := range rates {
		out["cpu"] = append(out["cpu"], int(r.CPU))
	}
	out["disk-read"] = scaled(func(r activity.Rates) float64 { return r.DiskRead })
	out["disk-write"] = scaled(func(r activity.Rates) float64 { return r.DiskWrite })
	out["net-in"] = scaled(func(r activity.Rates) float64 { return r.NetIn })
	out["net-out"] = scaled(func(r activity.Rates) float64 { return r.NetOut })
	for _, x := range points[1:] {
		if x.MemTotal > 0 {
			out["memory"] = append(out["memory"], int(100*x.MemUsed/x.MemTotal))
			out["swap"] = append(out["swap"], int(100*x.SwapUsed/x.MemTotal))
		}
	}
	return out
}

func processColumns() []page.Column {
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100, Whole: true},
		{Key: "what", Kind: page.KindText, Keep: 40},
		{Key: "cpu", Label: "CPU", Kind: page.KindPercent, Keep: 90},
		{Key: "memory", Label: "MEMORY", Kind: page.KindSize, Keep: 80},
	}
}

func processRows(v View) []page.Item {
	// invariant: the busiest by CPU then those only memory ranks, one row
	// per process, with both values where both lists name it.
	t := v.Report.Last.Top
	memory := map[int]float64{}
	for _, p := range t.Memory {
		memory[p.PID] = p.Value
	}
	cpu := map[int]float64{}
	for _, p := range t.CPU {
		cpu[p.PID] = p.Value
	}
	var out []page.Item
	seen := map[int]bool{}
	for _, p := range append(slices.Clone(t.CPU), t.Memory...) {
		if seen[p.PID] {
			continue
		}
		seen[p.PID] = true
		what := page.None()
		if p.VM != "" {
			what = page.Text("VM " + p.VM)
		}
		used, share := page.None(), page.None()
		if m, ok := memory[p.PID]; ok {
			used = page.Size(int64(m))
		}
		if c, ok := cpu[p.PID]; ok {
			share = page.Percent(int64(c * 10))
		}
		r := page.Row{ID: strconv.Itoa(p.PID), Label: p.Name, Cells: []page.Value{what, share, used}}
		out = append(out, page.Item{Row: &r})
	}
	return out
}
