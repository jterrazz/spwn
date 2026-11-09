package menubar

import (
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

const (
	SectionMemory      = "memory"
	SectionProcesses   = "processes"
	SectionMeasured    = "measured"
	SectionGuestUpkeep = "guest-upkeep"
	// value: the section of one window's charts, by its flag
	// ("window-1h"), and the picker's action of that window.
	SectionWindow = "window-"
	ActionWindow  = "window-"
	// value: the window a machine's Activity opens on until the owner
	// picks another (SPEC A4.1).
	WindowDefault = "1h"
	// value: the CPU share a window's anomaly holds at or above (SPEC
	// A4.1): the bars it spans are shaded and the title becomes its
	// sentence.
	HighCPU = 90
	RowCPU  = "cpu"
	RowNet  = "network"
	MB      = 1e6
)

type activityWindow struct {
	Flag, Words, Title string
	For                time.Duration
	Kind               page.ChartKind
}

// value: SPEC A4.1's picker: a line of 72 × 5 s over 6 min, bars of 60 ×
// 1 min over the hour and 48 × 30 min over the day.
func activityWindows() []activityWindow {
	return []activityWindow{
		{"6m", "6 min", "Last 6 minutes", 6 * time.Minute, page.ChartLine},
		{"1h", "1 hr", "Last hour", time.Hour, page.ChartBars},
		{"24h", "24 hr", "Last 24 hours", 24 * time.Hour, page.ChartBars},
	}
}

// value: a Mac's Activity segment (SPEC A4.1).
func macActivity(m Machine, h model.MacView) []page.Section {
	return activitySegment(m, h, nil)
}

// value: a VM's Activity segment: its Mac's segment as measured inside
// the guest, its upkeep when it is a workspace with updates, and the
// footnote that says where the figures come from.
func vmActivity(m Machine, h model.MacView, vm model.VM) []page.Section {
	return activitySegment(m, h, &vm)
}

type activityOf struct {
	m     Machine
	h     model.MacView
	vm    *model.VM
	name  string
	label string
}

// invariant: SPEC A4.1, B3: Activity for any machine carries each of its
// three windows at its points, so the window picked is the client's own
// and asks nothing; the window shown first is the one picked last, else
// an hour.
func activitySegment(m Machine, h model.MacView, vm *model.VM) []page.Section {
	a := activityOf{m: m, h: h, vm: vm, name: h.Name, label: model.Label(*m.All, h)}
	if vm != nil {
		a.name, a.label = vm.Name, vm.Name
	}
	var out []page.Section
	for _, w := range activityWindows() {
		out = append(out, a.window(w))
	}
	out = append(out, a.memory(), a.processes())
	if vm != nil {
		if s, ok := a.guestUpkeep(); ok {
			out = append(out, s)
		}
		out = append(out, a.measuredWhere())
	}
	for i := range out {
		out[i].Segment = SegmentActivity
	}
	return out
}

// value: the picker of every window, the window the page opens on picked.
func (a activityOf) picker() []page.Action {
	ws := activityWindows()
	out := make([]page.Action, 0, len(ws))
	for _, w := range ws {
		out = append(out, page.Action{ID: ActionWindow + w.Flag, Label: w.Words, On: page.OnPage, Picked: w.Flag == WindowDefault})
	}
	return out
}

// value: one window of the machine: its CPU then its network, each its
// card's title and its value now over its chart.
func (a activityOf) window(w activityWindow) page.Section {
	s := page.Section{ID: SectionWindow + w.Flag, Title: w.Title, Role: page.Summary, Card: page.CardChart, Columns: columns(), Picker: a.picker(), Uncounted: true}
	readings := a.m.Samples
	m := measured(readings)
	cpu := page.Row{ID: RowCPU, Label: "CPU", Detail: page.Text("now"), Cells: []page.Value{live(pendingOr(m.cpu), engine.LiveKey(engine.LiveCPU, a.name))}}
	net := page.Row{ID: RowNet, Label: "Network", Cells: []page.Value{netNow(readings)}}
	points := a.points(w)
	if len(points) < 2 {
		s.Items = []page.Item{{Row: &cpu}, {Row: &net}}
		return s
	}
	chart := activity.Fold(points, w.For, a.m.Now)
	ends, at := a.ends(w), a.times(w, chart)
	cpuChart := page.Chart{Kind: w.Kind, Window: w.Flag, Ends: ends, At: at, Series: []page.Series{{Key: RowCPU, Unit: "%", Points: means(chart)}}}
	if i := peakOf(chart); i >= 0 {
		b := chart.Buckets[i]
		cpuChart.Peak = &page.Peak{Index: i, Value: page.Percent(int64(math.Round(b.Mean)) * 10), At: a.when(w, b)}
		if from, to, ok := anomaly(chart, i); ok {
			cpuChart.Anomaly = &page.Anomaly{From: from, To: to, Sentence: a.sentence(chart, from, to, i)}
		}
	}
	sent, received := netFold(points, w.For, a.m.Now, chart.Every)
	netChart := page.Chart{Kind: page.ChartMirror, Window: w.Flag, Ends: ends, At: at, Series: []page.Series{{Key: "sent", Unit: "MB/s", Points: sent}, {Key: "received", Unit: "MB/s", Points: received}}}
	s.Items = []page.Item{{Row: &cpu}, {Chart: &cpuChart}, {Row: &net}, {Chart: &netChart}}
	return s
}

// value: the points a window folds: the engine's 2 s points for the 6 min
// it keeps them, its minutes for the hour and the day; none until read.
func (a activityOf) points(w activityWindow) []activity.Sample {
	r := a.m.Activity
	if r == nil {
		return nil
	}
	if w.For <= activity.FineEvery*activity.FinePoints && len(r.Graph) > 1 {
		return r.Graph
	}
	return r.History
}

func means(c activity.Chart) []float64 {
	out := make([]float64, len(c.Buckets))
	for i, b := range c.Buckets {
		out[i] = math.Round(b.Mean*10) / 10
	}
	return out
}

// value: the bar drawn highest: the bucket of the highest mean; -1 when
// no bucket holds a point, or when every bar is drawn at one height, a
// flat series having no peak to name.
func peakOf(c activity.Chart) int {
	peak, low := -1, math.Inf(1)
	for i, b := range c.Buckets {
		if b.Points == 0 {
			continue
		}
		low = min(low, math.Round(b.Mean))
		if peak < 0 || b.Mean > c.Buckets[peak].Mean {
			peak = i
		}
	}
	if peak >= 0 && math.Round(c.Buckets[peak].Mean) <= low {
		return -1
	}
	return peak
}

// value: the run of buckets at or above HighCPU around the peak; none
// when the peak stays under it.
func anomaly(c activity.Chart, peak int) (from, to int, ok bool) {
	high := func(i int) bool { return c.Buckets[i].Points > 0 && c.Buckets[i].Mean >= HighCPU }
	if !high(peak) {
		return 0, 0, false
	}
	from, to = peak, peak
	for from > 0 && high(from-1) {
		from--
	}
	for to < len(c.Buckets)-1 && high(to+1) {
		to++
	}
	return from, to, true
}

// value: the anomaly in the owner's words: how long the CPU has stayed
// high when it still is, else when it peaked.
func (a activityOf) sentence(c activity.Chart, from, to, peak int) string {
	if to == len(c.Buckets)-1 {
		return "CPU on " + a.label + " has been above " + strconv.Itoa(HighCPU) + " % for " + minutes(time.Duration(to-from+1)*c.Every)
	}
	b := c.Buckets[peak]
	return "Peaked " + strconv.Itoa(int(math.Round(b.Mean))) + "% at " + a.clock(b.MaxAt)
}

// value: when a bucket peaked, as its chip says it: how long ago within
// the hour, the time of day over the day.
func (a activityOf) when(w activityWindow, b activity.Bucket) string {
	if w.For > time.Hour {
		return a.clock(b.MaxAt)
	}
	return ago(a.m.Now.Sub(b.MaxAt)) + " ago"
}

// value: when each bucket of the window starts, as the scrub's callout
// says it: how long ago, to the second over 6 min ("2 min 35 s ago"), to
// the minute over the hour, the time of day over the day.
func (a activityOf) times(w activityWindow, c activity.Chart) []string {
	out := make([]string, len(c.Buckets))
	for i, b := range c.Buckets {
		d := a.m.Now.Sub(b.At)
		switch {
		case w.For > time.Hour:
			out[i] = a.clock(b.At)
		case c.Every < time.Minute && d >= time.Minute && d%time.Minute != 0:
			out[i] = strconv.Itoa(int(d/time.Minute)) + " min " + strconv.Itoa(int(d%time.Minute/time.Second)) + " s ago"
		default:
			out[i] = ago(d) + " ago"
		}
	}
	return out
}

func (a activityOf) clock(t time.Time) string {
	zone := a.m.Zone
	if zone == nil {
		zone = time.Local
	}
	return t.In(zone).Format("15:04")
}

// value: the chart's two ends: when the window starts, then now.
func (a activityOf) ends(w activityWindow) []string {
	if w.For < 24*time.Hour {
		return []string{w.Words + " ago", "now"}
	}
	start := a.m.Now.Add(-w.For)
	return []string{"yesterday " + a.clock(start), "now"}
}

// value: the network's rates sent and received in each bucket of the
// window, their means in MB/s, oldest first.
func netFold(points []activity.Sample, window time.Duration, now time.Time, every time.Duration) (sent, received []float64) {
	n := max(int(window/every), 1)
	start := now.Add(-time.Duration(n) * every)
	sent, received = make([]float64, n), make([]float64, n)
	count := make([]int, n)
	for i := 1; i < len(points); i++ {
		at := points[i].At
		if at.Before(start) || !at.Before(start.Add(time.Duration(n)*every)) {
			continue
		}
		k := int(at.Sub(start) / every)
		r := activity.Between(points[i-1], points[i])
		sent[k] += r.NetOut / MB
		received[k] += r.NetIn / MB
		count[k]++
	}
	for k := range sent {
		if count[k] > 0 {
			sent[k] = math.Round(sent[k]/float64(count[k])*100) / 100
			received[k] = math.Round(received[k]/float64(count[k])*100) / 100
		}
	}
	return sent, received
}

// value: the network now, received then sent, each labeled ("0.5", "0.5
// MB/s"), from the last two readings; pending until there are two.
func netNow(readings []activity.Reading) page.Value {
	n := len(readings)
	if n < 2 {
		return page.Pending()
	}
	r := activity.Between(readings[n-2].Sample, readings[n-1].Sample)
	say := func(v float64) string { return strconv.FormatFloat(v/MB, 'f', 1, 64) }
	return page.List(page.Labeled("received", page.Text(say(r.NetIn))), page.Labeled("sent", page.Text(say(r.NetOut)+" MB/s")))
}

// value: the memory now: what is free, then what holds the rest (macOS
// and each VM on a Mac, used and free inside a VM) over the total, which
// its bar draws.
func (a activityOf) memory() page.Section {
	s := page.Section{ID: SectionMemory, Title: "Memory", Role: page.Summary, Card: page.CardChart, Columns: columns(), Caption: []page.Value{page.Text("now")}, Uncounted: true}
	gauge := page.Gauge{Title: "Memory", Value: live(pendingOr(freeWords(a.m.Samples)), engine.LiveKey(engine.LiveFree, a.name))}
	if n := len(a.m.Samples); n > 0 {
		if last := a.m.Samples[n-1].Sample; last.MemTotal > 0 {
			used := min(last.MemUsed, last.MemTotal)
			var bar page.Value
			if a.vm != nil {
				bar = page.List(part("used", used), part("free", last.MemTotal-used), part("of", last.MemTotal))
			} else {
				var vms []page.Value
				held := int64(0)
				for _, vm := range a.h.VMs {
					if vm.State == model.Running && vm.Resources.MemoryBytes > 0 {
						vms = append(vms, page.Size(vm.Resources.MemoryBytes))
						held += vm.Resources.MemoryBytes
					}
				}
				all := part("VMs", held)
				all.Kind, all.Items = page.KindList, vms
				bar = page.List(part("macOS", max(used-held, 0)), all, part("of", last.MemTotal))
			}
			gauge.Bar = &bar
		}
	}
	s.Items = []page.Item{{Gauge: &gauge}}
	return s
}

// value: one part of the memory bar: its bytes, and its words in the GB
// the free memory is said in ("9 GB").
func part(label string, bytes int64) page.Value {
	v := page.Labeled(label, page.Size(bytes))
	v.Text = strconv.FormatInt((bytes+1<<29)>>30, 10) + " GB"
	return v
}

// value: the busiest processes now, each its share of the machine's CPU:
// on a Mac each VM it runs is one of them, by its name, and opens its page.
func (a activityOf) processes() page.Section {
	s := page.Section{ID: SectionProcesses, Title: "Top processes", Role: page.Summary, Columns: vmColumns(), Uncounted: true,
		Caption: []page.Value{page.Text("CPU, of " + a.label)}}
	n := len(a.m.Samples)
	if n == 0 {
		return s
	}
	last := a.m.Samples[n-1]
	var rows []page.Row
	if a.vm == nil {
		for _, u := range last.VMs {
			vm, ok := vmNamed(a.h, u.VM)
			if !ok {
				continue
			}
			row := page.Row{ID: "vm:" + vm.Name, Label: vm.Name, Glyph: processGlyph(vm), Nav: machinePage(vm.Name), Trail: "cpu",
				Cells: []page.Value{live(page.Percent(int64(u.CPU*10)), engine.LiveKey(engine.LiveCPUMac, vm.Name))}}
			if c := vm.Resources.CPUs; c > 0 {
				row.Detail = page.Text(strconv.Itoa(c) + " cores")
			}
			rows = append(rows, row)
		}
	}
	for _, p := range last.Top.CPU {
		if p.VM != "" && a.vm == nil {
			continue
		}
		rows = append(rows, page.Row{ID: strconv.Itoa(p.PID), Label: p.Name, Glyph: symbol("cpu", page.SignalQuiet), Trail: "cpu", Cells: []page.Value{page.Percent(int64(p.Value * 10))}})
	}
	slices.SortStableFunc(rows, func(x, y page.Row) int { return int(y.Cells[0].N - x.Cells[0].N) })
	s.Items = items(rows)
	return s
}

func vmNamed(h model.MacView, name string) (model.VM, bool) {
	for _, vm := range h.VMs {
		if vm.Name == name {
			return vm, true
		}
	}
	return model.VM{}, false
}

// value: a VM among the processes: the agent mark while it hosts a live
// agent, else its kind's glyph.
func processGlyph(vm model.VM) page.Glyph {
	switch {
	case vm.Lifetime.Kind() == machine.Server:
		return page.Glyph{Kind: "server"}
	case vm.Agent != nil && !vm.Finished():
		return page.Glyph{Kind: "agent"}
	}
	return page.Glyph{Kind: "workspace"}
}

// value: SPEC A4 A workspace: its upkeep under its Activity when its
// guest has updates, with Update.
func (a activityOf) guestUpkeep() (page.Section, bool) {
	if a.vm.Lifetime.Kind() == machine.Server || a.m.Summary == nil {
		return page.Section{}, false
	}
	s := page.Section{ID: SectionGuestUpkeep, Title: "Upkeep", Role: page.Summary, Columns: columns(), Uncounted: true}
	for _, n := range a.m.Summary.Needs {
		if n.Call != CallUpdate {
			continue
		}
		row := page.Row{ID: "update", Label: n.Text, Glyph: symbol("arrow.down.circle", page.SignalWaits), Verb: ActionUpdateGuest}
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	return s, len(s.Items) > 0
}

const ActionUpdateGuest = "update-guest"

// value: the action a workspace's upkeep row runs: its apps updated
// inside it.
func updateGuest(h model.MacView, vm model.VM) page.Action {
	t := &page.Target{Name: vm.Name}
	if !h.ThisMac {
		t.On = h.Name
	}
	return page.Action{ID: ActionUpdateGuest, Label: VerbUpdate, On: page.OnRow, Rows: []string{"update"}, Mutates: true,
		Run: &page.Run{Call: CallUpdate, Target: t}, Hud: hud("Everything is up to date")}
}

// value: SPEC A4.1: where a VM's figures come from.
func (a activityOf) measuredWhere() page.Section {
	note := "Measured inside the guest. If " + a.vm.Name + " stops answering, this shows what " + model.Label(*a.m.All, a.h) + " sees of it."
	return page.Section{ID: SectionMeasured, Title: "Measured", Items: []page.Item{{Note: &page.Note{Text: note}}}}
}
