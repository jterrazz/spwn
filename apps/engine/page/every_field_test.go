package page_test

import "github.com/jterrazz/studio-station/apps/engine/page"

func everyField() page.Page {
	cpu := page.Value{Kind: page.KindPercent, N: 340, Live: "cpu:linux-1", Signal: page.SignalActive}
	disk := page.Value{Kind: page.KindSize, N: 2_400_000_000, Of: 20_000_000_000, Unit: "GB", Label: "disk", Stale: true}
	on := page.Value{Kind: page.KindText, Text: "on mac-mini", Short: "mini", Pri: 2, Nav: "menubar.machine:mac-mini"}
	since := page.Value{Kind: page.KindAge, At: "2026-11-03T10:20:00Z", N: 600}
	hint := &page.Hint{Command: "station machines linux-1", Text: "open it"}
	createForm := &page.Form{
		Title:    "New VM",
		Subtitle: "on This Mac",
		Fields: []page.Field{
			{Key: "kind", Label: "Kind", Kind: page.FieldSegmented, Options: []page.Option{{Value: "workspace", Label: "Workspace"}, {Value: "server", Label: "Server", Note: "always on", Disabled: "Needs infra running"}}, Default: "workspace", Required: true},
			{Key: "task", Label: "Task", Kind: page.FieldText, Note: "what Claude starts on", When: &page.When{Key: "kind", Value: "workspace"}},
			{Key: "memory", Label: "Memory", Kind: page.FieldSlider, Stops: []int{4, 8, 16}, Limit: 16, LimitNote: "the most this Mac can lend"},
			{Key: "on", Label: "On", Kind: page.FieldSelect, Swaps: true, Options: []page.Option{{Value: "new-vm", Label: "This Mac"}}},
			{Key: "name", Label: "Name", Kind: page.FieldText, Taken: []string{"linux-1"}, TakenHud: "{name} already exists"},
			{Key: "lend-for", Label: "Lend for", Kind: page.FieldSegmented, When: &page.When{Key: "lend", Set: true}},
			{Key: "more", Label: "More", Kind: page.FieldDisclosure, Summary: page.Text("NAT, ephemeral"), Fields: []page.Field{{Key: "network", Kind: page.FieldSelect}}},
			{Key: "Account", Label: "Account", Kind: page.FieldPick, Options: []page.Option{{Value: "github", Label: "GitHub", Note: "jterrazz", Glyph: page.Glyph{Kind: "account:github"}, Of: "linux-1"}}, Follows: "VMs"},
			{Key: "copy", Kind: page.FieldCommand, Note: "On the new Mac, run this once.", Hint: page.Cmd("station machines add mac-mini")},
		},
		Primary: []page.Primary{{Label: "Create"}, {When: &page.When{Key: "task", Value: "*", Count: 1}, Label: "Create and Start Claude", Danger: true}},
		Split:   []string{"create-only"},
		Warn:    "This Mac may sleep: a server pauses with it.",
	}
	agentRow := page.Row{
		ID: "agent-7f", Mark: page.Attention, Label: "agent-7f", Short: "a7", Sentence: true,
		Cells:  []page.Value{cpu},
		Detail: page.List(on, since), Reason: page.Waiting, Hint: hint,
		Glyph: page.Glyph{Kind: "agent", Signal: page.SignalWaits, Dot: page.DotPing, Ring: "cpu:linux-1", Spin: true, Progress: 400},
		Mac:   "mac-mini", Trail: "value", Verb: "reply", Quick: "copy", Nav: "menubar.machine:linux-1",
		Menu: []string{"copy", "-", "delete"}, Signal: page.SignalAsleep, Fresh: true,
		Left:  &page.Value{Kind: page.KindUntil, N: 1350, Of: 3600},
		Birth: &page.Value{Kind: page.KindCount, N: 400, Of: 1000},
	}
	machine := page.Section{
		ID: "agents", Title: "Agents", Role: page.Plain, Fold: page.Auto,
		Caption: []page.Value{page.Text("2 working")}, Joined: true,
		Columns:    []page.Column{{Key: "name", Label: "Name", Kind: page.KindText, Keep: 100, Show: page.EyeOnly, Whole: true}, {Key: "value", Kind: page.KindLevels, Levels: []string{"data"}}},
		Items:      []page.Item{{Row: &agentRow}, {Group: &page.Group{Head: page.Row{ID: "group", Mark: page.OK}, Items: []page.Item{row(page.Row{ID: "child", Mark: page.Awaiting})}, Fold: page.Folded}}},
		PipeCounts: true, PipeGroups: true, Break: 1, SaysDropped: true,
		Top: 1, Rest: &page.Item{Note: &page.Note{Text: "+ 3 more"}},
		Glance:  []page.Value{page.Text("1 waits")},
		Segment: "overview", Card: page.CardNow,
		Picker: []page.Action{{ID: "window-6m", Label: "6 min", Picked: true, Run: &page.Run{Call: "activity.window", In: map[string]string{"Window": "6m"}}}},
		About:  &page.About{Label: "About agents", Text: "An agent is Claude at work in its own VM."},
	}
	cards := page.Section{
		ID: "cards", Title: "Cards", Uncounted: true, Counts: []page.Tally{{Mark: page.OK, N: 3}}, Segment: "activity", Card: page.CardHealth,
		Items: []page.Item{
			{Fact: &page.Fact{Key: "uptime", Value: page.Duration(7200)}},
			{Empty: &page.Empty{Noun: "VMs", Text: "No VMs yet", Hint: page.Cmd("station machines create"), Glyph: page.Glyph{Kind: "symbol:shippingbox"}, Sub: "Create one to start", Actions: []string{"new-vm"}}},
			{Error: &page.Error{Message: "linux-1 does not answer", Hint: "station machines linux-1"}},
			{Link: &page.Link{Label: "web", Lead: page.PortMap(5173, 40001), Value: page.URL("http://localhost:40001"), Detail: page.Text("vite"), Page: "menubar.ports", Row: "web"}},
			{Tile: &page.Tile{Caption: "VMs", Glyph: page.Glyph{Kind: "server"}, Number: page.Count(3, ""), Word: "running", Signal: page.SignalWell, Toggle: "awake", Nav: "menubar.awake", Plus: "+1", Tip: page.Texts("3 running", "right-click for a term"), Key: "cmd+e"}},
			{Gauge: &page.Gauge{Title: "CPU", Note: "4 cores", Value: cpu, Spark: []int{10, 20, 34}, Bar: &disk, Stale: true}},
			{Chart: &page.Chart{
				Kind: page.ChartBars, Window: "6m",
				Series:  []page.Series{{Key: "cpu", Points: []float64{1.5, 34, 12}, Unit: "%"}},
				Peak:    &page.Peak{Index: 1, Value: page.Percent(340), At: "10:28"},
				Ends:    []string{"6 min ago", "now"},
				Anomaly: &page.Anomaly{From: 1, To: 2, Sentence: "CPU stayed high for 2 min"},
				At:      []string{"15 s ago", "10 s ago", "5 s ago"},
			}},
			{Message: &page.Message{Title: "Claude waits", Body: "Should I merge?", Since: since, Reply: true,
				PR: &page.PR{Number: 42, URL: "https://github.com/jterrazz/x/pull/42", Files: 3, Plus: 120, Minus: 8, Opened: since}}},
			{Steps: []page.Step{{At: since, Text: "ran the tests", Diff: "+12 -3", Current: true}}},
			{Log: &page.Log{Lines: []string{"listening", "panic: nil map"}, Cause: 2, More: "logs"}},
			{Uptime: &page.Uptime{Days: []page.Day{{Date: "2026-11-02", Mark: page.DayRestart, Tip: "restarted at 04:10"}}, Availability: page.Percent(998)}},
		},
	}
	summary := page.Section{ID: "summary", Title: "linux-1", Role: page.Summary, Caption: []page.Value{on}}
	return page.Page{
		ID: "menubar.machine:linux-1", Command: "machines linux-1",
		Context: page.Context{Machine: "linux-1", Kind: "workspace", Registered: true, On: "mac-mini"},
		Strip: &page.Strip{
			CPU: cpu, Spark: []int{1, 2}, Memory: page.Percent(500), MemorySpark: []int{3}, NetSpark: []int{4},
			Battery: page.Percent(800), Net: []page.Value{page.Size(1)}, Awake: page.Value{Kind: page.KindUntil, At: "2026-11-03T18:00:00Z", N: 27000},
			Top: &page.Section{ID: "top", Title: "Top"},
		},
		Sections: []page.Section{summary, machine, cards},
		Hints:    []page.Hint{{Command: "open -a Terminal", External: true}},
		Error:    &page.Error{Message: "partly read", Hint: "station machines linux-1"},
		Actions: []page.Action{
			{ID: "copy", Key: "cmd+c", Label: "Copy the address", On: page.OnRow, Rows: []string{"agent-7f"}, Priority: 1, Icon: "doc.on.doc",
				Run: &page.Run{Copy: "linux-1.local"}, Hud: &page.Value{Kind: page.KindText, Text: "Copied linux-1.local"}},
			{ID: "reply", Label: "Reply", Default: true, Busy: "Sending", Run: &page.Run{Call: "agents.reply", Target: &page.Target{Name: "linux-1", On: "mac-mini"}}},
			{ID: "delete", Label: "Delete", Mutates: true, Destructive: true, Run: &page.Run{Terminal: true, Argv: []string{"machines", "delete", "linux-1"}},
				Confirm: &page.Confirm{Question: "Delete linux-1?", Lines: []page.Value{page.Text("its disk is removed")}, Danger: true, Type: "linux-1", Yes: "Delete", No: "Cancel", Icon: "warn"}},
			{ID: "new-vm", Label: "New VM", Key: "cmd+n", Sheet: page.SheetNewVM, Prefill: map[string]string{"kind": "server"}, Form: createForm, Run: &page.Run{Call: "machines.create-workspace"}},
			{ID: "create-only", Label: "Create only", Run: &page.Run{Call: "machines.create-workspace", Then: &page.Run{Terminal: true, Argv: []string{"--on", "{name}"}},
				And: []page.Run{{Call: "agents.forget", In: map[string]string{"Agent": "agent-7f"}}}}},
			{ID: "awake", Label: "Keep awake", Run: &page.Run{Call: "awake.keep"}},
			{ID: "logs", Label: "Show the log", Run: &page.Run{Open: "https://grafana.example/linux-1"}},
			{ID: "menu", Label: "More", Submenu: []string{"copy", "logs"}},
			{ID: "attach", Label: "Attach", Disabled: "Needs infra running"},
		},
		Badge:    &page.Badge{Needs: 1, Down: true, Restarting: true, Label: "Station, 1 needs you"},
		Header:   &page.Header{Back: "All machines", Glyph: page.Glyph{Kind: "workspace"}, Buttons: []string{"new-vm", "menu"}, Stale: &page.Value{Kind: page.KindAge, N: 40, Text: "seen 40 s ago, checking"}},
		Segments: []page.Segment{{ID: "overview", Label: "Overview", Picked: true}, {ID: "activity", Label: "Activity", Key: "cmd+i"}},
		Footer:   &page.Footer{Kind: page.FooterSSH, Copy: &page.Value{Kind: page.KindCommand, Text: "linux-1.local"}, Note: "Needs infra running", Buttons: []string{"attach"}},
		Live:     []string{"cpu:linux-1"},
	}
}
