package page

import "strings"

const Schema = "station.page/v1"

const Revision = 4

type Page struct {
	ID       ID        `json:"id,omitempty"`
	Command  string    `json:"command,omitempty"`
	Context  Context   `json:"context,omitzero"`
	State    State     `json:"state,omitempty"`
	Strip    *Strip    `json:"strip,omitempty"`
	Sections []Section `json:"sections,omitempty"`
	Hints    []Hint    `json:"hints,omitempty"`
	Error    *Error    `json:"error,omitempty"`
	Actions  []Action  `json:"actions,omitempty"`
	Badge    *Badge    `json:"badge,omitempty"`
	Header   *Header   `json:"header,omitempty"`
	Segments []Segment `json:"segments,omitempty"`
	Footer   *Footer   `json:"footer,omitempty"`
	Live     []string  `json:"live,omitempty"`
}

type Header struct {
	Back    string   `json:"back,omitempty"`
	Glyph   Glyph    `json:"glyph,omitzero"`
	Buttons []string `json:"buttons,omitempty"`
	Stale   *Value   `json:"stale,omitempty"`
}

type Segment struct {
	ID     string `json:"id,omitempty"`
	Label  string `json:"label,omitempty"`
	Key    string `json:"key,omitempty"`
	Picked bool   `json:"picked,omitempty"`
}

type Footer struct {
	Kind    FooterKind `json:"kind"`
	Copy    *Value     `json:"copy,omitempty"`
	Note    string     `json:"note,omitempty"`
	Buttons []string   `json:"buttons,omitempty"`
}

type FooterKind string

const (
	FooterFind      FooterKind = "find"
	FooterSSH       FooterKind = "ssh"
	FooterTerminals FooterKind = "terminal-list"
	FooterNone      FooterKind = "none"
)

type Glyph struct {
	Kind     string `json:"kind,omitempty"`
	Signal   Signal `json:"signal,omitempty"`
	Dot      Dot    `json:"dot,omitempty"`
	Ring     string `json:"ring,omitempty"`
	Spin     bool   `json:"spin,omitempty"`
	Progress int    `json:"progress,omitempty"`
}

type Signal string

const (
	SignalActive  Signal = "active"
	SignalWaits   Signal = "waits"
	SignalFailing Signal = "failing"
	SignalWell    Signal = "well"
	SignalQuiet   Signal = "quiet"
	SignalFaint   Signal = "faint"
	SignalDone    Signal = "done"
	SignalAsleep  Signal = "asleep"
	SignalStale   Signal = "stale"
)

type Dot string

const (
	DotFill   Dot = "fill"
	DotRing   Dot = "ring"
	DotDash   Dot = "dash"
	DotSquare Dot = "square"
	DotPing   Dot = "ping"
)

type CardKind string

const (
	CardHero    CardKind = "hero"
	CardTiles   CardKind = "tiles"
	CardGauges  CardKind = "gauges"
	CardFlush   CardKind = "flush"
	CardPlain   CardKind = "plain"
	CardWait    CardKind = "wait"
	CardNow     CardKind = "now"
	CardHealth  CardKind = "health"
	CardCheck   CardKind = "check"
	CardUptime  CardKind = "uptime"
	CardLog     CardKind = "log"
	CardChart   CardKind = "chart"
	CardEmpty   CardKind = "empty"
	CardExposed CardKind = "exposed"
	CardCare    CardKind = "care"
)

type Badge struct {
	// value: what the menu bar's icon says of the page's machines: how many
	// things need you, and whether a server is down, which wins over the count.
	Needs      int    `json:"needs"`
	Down       bool   `json:"down,omitempty"`
	Restarting bool   `json:"restarting,omitempty"`
	Label      string `json:"label,omitempty"`
}

type ID string

type State string

const (
	Loading State = "loading"
	Ready   State = "ready"
	Failed  State = "failed"
)

type Context struct {
	Machine    string `json:"machine,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Registered bool   `json:"registered,omitempty"`
	// value: the Mac a VM's page is on, as the owner names it (`mac-mini`,
	// `This Mac`); empty for a Mac's own.
	On string `json:"on,omitempty"`
}

type Section struct {
	ID      string  `json:"id,omitempty"`
	Title   string  `json:"title,omitempty"`
	Role    Role    `json:"role,omitempty"`
	Fold    Fold    `json:"fold,omitempty"`
	Caption []Value `json:"caption,omitempty"`
	// value: the caption reads on as the title's own words, one list with it
	// (All machines' first line), not a muted aside.
	Joined  bool     `json:"joined,omitempty"`
	Columns []Column `json:"columns,omitempty"`
	Items   []Item   `json:"items,omitempty"`
	Counts  []Tally  `json:"counts,omitempty"`
	// value: a section whose marks are no verdict to sum (storage's sizes) shows no counts.
	Uncounted bool `json:"uncounted,omitempty"`
	// value: the pipe prints a title's counts only where its records always
	// carried them (the tools and config categories, Home's Tools, Config and
	// Background); every other `# <title>` line stays bare (§20).
	PipeCounts bool `json:"pipe_counts,omitempty"`
	// value: the pipe prints each heading group as its own titled section
	// (Secrets' `# Lend list` and `# Kept`), never the section's title: its
	// records do not depend on how the eye nests them.
	PipeGroups bool `json:"pipe_groups,omitempty"`
	// value: a Summary drawn in two columns starts its right one at this item.
	Break int `json:"break,omitempty"`
	// value: a line too short for every column says which it left out
	// (All machines' table, SPEC §9.9).
	SaysDropped bool `json:"says_dropped,omitempty"`
	// value: on a terminal's page, the rows shown before Rest, the one row
	// that sums the others (vision 07 "station disk": `+ 15 more  7.8 GB`);
	// the TUI and the menu bar show every row, --all unfolds it.
	Top  int   `json:"top,omitempty"`
	Rest *Item `json:"rest,omitempty"`
	// value: the section in a few words, for a surface that gives it one
	// line (the menu bar's folded Needs you); a terminal shows its rows.
	Glance  []Value  `json:"glance,omitempty"`
	Segment string   `json:"segment,omitempty"`
	Card    CardKind `json:"card,omitempty"`
	Picker  []Action `json:"picker,omitempty"`
	// value: what the section's title leaves unsaid, behind the info
	// button beside it (Private · on your tailnet, SPEC A4 pop.info).
	About *About `json:"about,omitempty"`
}

// value: an info button's name and the words it opens.
type About struct {
	Label string `json:"label,omitempty"`
	Text  string `json:"text,omitempty"`
}

type Role string

const (
	Plain   Role = ""
	Actions Role = "actions"
	Summary Role = "summary"
	Banner  Role = "banner"
	// value: what needs a look, lifted first on a terminal's page from the
	// sections its rows stay in (vision 07 "station settings"): the pipe, the
	// TUI and the menu bar read those rows where they are.
	Lifted Role = "lifted"
)

type Fold string

const (
	Open   Fold = "open"
	Folded Fold = "folded"
	Auto   Fold = "auto"
	// value: one line on a terminal's page, open in the TUI and the menu bar
	// (vision 07 "station apps" folds Not installed; screen 2 lists it).
	Brief Fold = "brief"
	// value: one line on a terminal's page, folded in the TUI while every
	// row is calm, as Auto is (Settings' Set by station).
	Calm Fold = "calm"
)

type Column struct {
	Key    string   `json:"key,omitempty"`
	Label  string   `json:"label,omitempty"`
	Kind   Kind     `json:"kind,omitempty"`
	Keep   int      `json:"keep,omitempty"`
	Show   Show     `json:"show,omitempty"`
	Levels []string `json:"levels,omitempty"`
	// value: a column a short line drops whole rather than cuts (a process
	// name, a machine's resources: half of one says nothing, §13.5, §13.6).
	Whole bool `json:"whole,omitempty"`
}

type Show string

const (
	Everywhere Show = ""
	PipeOnly   Show = "pipe"
	// value: a column for the eye only (a peer's online state, a host's role):
	// the pipe's records keep the fields they always had (§20).
	EyeOnly Show = "eye"
	// value: a column only a row's detail shows (the folders a disk row
	// holds): no line of a list and no record of the pipe carries it.
	DetailOnly Show = "detail"
)

type Item struct {
	Row     *Row     `json:"row,omitempty"`
	Fact    *Fact    `json:"fact,omitempty"`
	Group   *Group   `json:"group,omitempty"`
	Empty   *Empty   `json:"empty,omitempty"`
	Error   *Error   `json:"error,omitempty"`
	Note    *Note    `json:"note,omitempty"`
	Link    *Link    `json:"link,omitempty"`
	Tile    *Tile    `json:"tile,omitempty"`
	Gauge   *Gauge   `json:"gauge,omitempty"`
	Chart   *Chart   `json:"chart,omitempty"`
	Message *Message `json:"message,omitempty"`
	Steps   []Step   `json:"steps,omitempty"`
	Log     *Log     `json:"log,omitempty"`
	Uptime  *Uptime  `json:"uptime,omitempty"`
}

type Tile struct {
	Caption string `json:"caption,omitempty"`
	Glyph   Glyph  `json:"glyph,omitzero"`
	Number  Value  `json:"number,omitzero"`
	Word    string `json:"word,omitempty"`
	Signal  Signal `json:"signal,omitempty"`
	Toggle  string `json:"toggle,omitempty"`
	Nav     ID     `json:"nav,omitempty"`
	Plus    string `json:"plus,omitempty"`
}

type Gauge struct {
	Title string `json:"title,omitempty"`
	Value Value  `json:"value,omitzero"`
	Spark []int  `json:"spark,omitempty"`
	Bar   *Value `json:"bar,omitempty"`
	Stale bool   `json:"stale,omitempty"`
}

type ChartKind string

const (
	ChartBars   ChartKind = "bars"
	ChartLine   ChartKind = "line"
	ChartMirror ChartKind = "mirror"
)

type Chart struct {
	Kind    ChartKind `json:"kind,omitempty"`
	Window  string    `json:"window,omitempty"`
	Series  []Series  `json:"series,omitempty"`
	Peak    *Peak     `json:"peak,omitempty"`
	Ends    []string  `json:"ends,omitempty"`
	Anomaly *Anomaly  `json:"anomaly,omitempty"`
}

type Series struct {
	Key    string    `json:"key,omitempty"`
	Points []float64 `json:"points,omitempty"`
	Unit   string    `json:"unit,omitempty"`
}

type Peak struct {
	Index int    `json:"index"`
	Value Value  `json:"value,omitzero"`
	At    string `json:"at,omitempty"`
}

type Anomaly struct {
	From     int    `json:"from"`
	To       int    `json:"to"`
	Sentence string `json:"sentence,omitempty"`
}

type Message struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	Since Value  `json:"since,omitzero"`
	PR    *PR    `json:"pr,omitempty"`
	Reply bool   `json:"reply,omitempty"`
}

type PR struct {
	Number int    `json:"number,omitempty"`
	URL    string `json:"url,omitempty"`
	Files  int    `json:"files,omitempty"`
	Plus   int    `json:"plus,omitempty"`
	Minus  int    `json:"minus,omitempty"`
	Opened Value  `json:"opened,omitzero"`
}

type Step struct {
	At      Value  `json:"at,omitzero"`
	Text    string `json:"text,omitempty"`
	Diff    string `json:"diff,omitempty"`
	Current bool   `json:"current,omitempty"`
}

type Log struct {
	Lines []string `json:"lines,omitempty"`
	Cause int      `json:"cause,omitempty"`
	More  string   `json:"more,omitempty"`
}

type DayMark string

const (
	DayOK      DayMark = "ok"
	DayRestart DayMark = "restart"
	DayDown    DayMark = "down"
	DayNone    DayMark = "none"
)

type Uptime struct {
	Days         []Day `json:"days,omitempty"`
	Availability Value `json:"availability,omitzero"`
}

type Day struct {
	Date string  `json:"date,omitempty"`
	Mark DayMark `json:"mark,omitempty"`
	Tip  string  `json:"tip,omitempty"`
}

type Row struct {
	ID    string `json:"id,omitempty"`
	Mark  Mark   `json:"mark,omitempty"`
	Label string `json:"label,omitempty"`
	// value: the label a terminal shows when the full one is for records
	// (a launchd label without its owner's prefix); the pipe and json keep Label.
	Short string `json:"short,omitempty"`
	// value: on a terminal, the label is the subject of the row's first
	// words, one sentence with them (`This Mac stays awake until 18:00`).
	Sentence bool     `json:"sentence,omitempty"`
	Cells    []Value  `json:"cells,omitempty"`
	Detail   Value    `json:"detail,omitzero"`
	Reason   Reason   `json:"reason,omitempty"`
	Hint     *Hint    `json:"hint,omitempty"`
	Glyph    Glyph    `json:"glyph,omitzero"`
	Mac      string   `json:"mac,omitempty"`
	Trail    string   `json:"trail,omitempty"`
	Verb     string   `json:"verb,omitempty"`
	Quick    string   `json:"quick,omitempty"`
	Nav      ID       `json:"nav,omitempty"`
	Menu     []string `json:"menu,omitempty"`
	Signal   Signal   `json:"signal,omitempty"`
	Fresh    bool     `json:"fresh,omitempty"`
	Left     *Value   `json:"left,omitempty"`
	Birth    *Value   `json:"birth,omitempty"`
}

type Fact struct {
	Key   string `json:"key,omitempty"`
	Value Value  `json:"value,omitzero"`
}

type Group struct {
	Head  Row    `json:"head,omitzero"`
	Items []Item `json:"items,omitempty"`
	Fold  Fold   `json:"fold,omitempty"`
}

func (g Group) Heading() bool {
	// value: a head with no mark and no cells only names its children's group
	// (Background's `known`): it carries no verdict and the pipe prints no record for it.
	return g.Head.Mark == Unjudged && len(g.Head.Cells) == 0
}

type Empty struct {
	Noun string `json:"noun,omitempty"`
	// value: the whole line when "no <noun>" is not it (No VMs yet).
	Text    string   `json:"text,omitempty"`
	Hint    *Hint    `json:"hint,omitempty"`
	Glyph   Glyph    `json:"glyph,omitzero"`
	Sub     string   `json:"sub,omitempty"`
	Actions []string `json:"actions,omitempty"`
}

type Error struct {
	Message string `json:"message,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

type Note struct {
	Text string `json:"text,omitempty"`
}

type Link struct {
	Label string `json:"label,omitempty"`
	// value: what the line leads with in the label's place, never cut (a
	// service's `5173 → 40001`); Label stays the link's name for the records.
	Lead  Value `json:"lead,omitzero"`
	Value Value `json:"value,omitzero"`
	// value: dropped whole, never cut, when the line is short (a process name).
	Detail Value  `json:"detail,omitzero"`
	Page   ID     `json:"page,omitempty"`
	Row    string `json:"row,omitempty"`
}

type Hint struct {
	Command  string `json:"command,omitempty"`
	External bool   `json:"external,omitempty"`
	Key      string `json:"key,omitempty"`
	Text     string `json:"text,omitempty"`
}

type Action struct {
	Key      string   `json:"key,omitempty"`
	Label    string   `json:"label,omitempty"`
	On       Scope    `json:"on,omitempty"`
	Rows     []string `json:"rows,omitempty"`
	Confirm  *Confirm `json:"confirm,omitempty"`
	Priority int      `json:"priority,omitempty"`
	Mutates  bool     `json:"mutates,omitempty"`
	ID       string   `json:"id,omitempty"`
	Run      *Run     `json:"run,omitempty"`
	Form     *Form    `json:"form,omitempty"`
	// value: the choice in force among its row's choices (Awake's term):
	// a segmented control's selected segment.
	Picked      bool              `json:"picked,omitempty"`
	Icon        string            `json:"icon,omitempty"`
	Default     bool              `json:"default,omitempty"`
	Busy        string            `json:"busy,omitempty"`
	Sheet       string            `json:"sheet,omitempty"`
	Prefill     map[string]string `json:"prefill,omitempty"`
	Submenu     []string          `json:"submenu,omitempty"`
	Disabled    string            `json:"disabled,omitempty"`
	Hud         *Value            `json:"hud,omitempty"`
	Destructive bool              `json:"destructive,omitempty"`
}

const (
	SheetNewVM        = "new-vm"
	SheetShareAccount = "share-account"
	SheetSharePort    = "share-port"
	SheetCleanup      = "cleanup"
	SheetHint         = "hint:"
)

type Run struct {
	// value: the call the action makes, by its name in calls.json, over the
	// engine's socket.
	Call string `json:"call,omitempty"`
	// value: the call's input the action fixes, by the path of each field
	// (`Request.Image`); a Form's answers fill the paths their keys name. A
	// value written as JSON (`"5173"`, `["never-sleeps"]`) is sent as that
	// JSON: a number the call reads as text, a list.
	In map[string]string `json:"in,omitempty"`
	// value: a new terminal runs station, with these words after it: the one
	// way an action starts a program.
	Terminal bool     `json:"terminal,omitempty"`
	Argv     []string `json:"argv,omitempty"`
	// value: the machine the call runs for when it is not this Mac's engine's
	// own: another Mac by its name, or a VM of another Mac by its name and
	// its Mac.
	Target *Target `json:"target,omitempty"`
	Copy   string  `json:"copy,omitempty"`
	Open   string  `json:"open,omitempty"`
}

type Target struct {
	Name string `json:"name,omitempty"`
	On   string `json:"on,omitempty"`
}

type Scope string

const (
	OnRow  Scope = "row"
	OnPage Scope = "page"
)

type Confirm struct {
	Question string  `json:"question,omitempty"`
	Lines    []Value `json:"lines,omitempty"`
	Danger   bool    `json:"danger,omitempty"`
	Type     string  `json:"type,omitempty"`
	Yes      string  `json:"yes,omitempty"`
	No       string  `json:"no,omitempty"`
	Icon     string  `json:"icon,omitempty"`
}

type Form struct {
	Title    string    `json:"title,omitempty"`
	Fields   []Field   `json:"fields,omitempty"`
	Subtitle string    `json:"subtitle,omitempty"`
	Primary  []Primary `json:"primary,omitempty"`
	Split    []string  `json:"split,omitempty"`
}

type Primary struct {
	When  *When  `json:"when,omitempty"`
	Label string `json:"label,omitempty"`
}

type When struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Field struct {
	Key       string    `json:"key,omitempty"`
	Label     string    `json:"label,omitempty"`
	Kind      FieldKind `json:"kind,omitempty"`
	Options   []Option  `json:"options,omitempty"`
	Default   string    `json:"default,omitempty"`
	Note      string    `json:"note,omitempty"`
	Required  bool      `json:"required,omitempty"`
	Stops     []int     `json:"stops,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	LimitNote string    `json:"limit_note,omitempty"`
	Fields    []Field   `json:"fields,omitempty"`
	Summary   Value     `json:"summary,omitzero"`
	When      *When     `json:"when,omitempty"`
}

type FieldKind string

const (
	FieldText       FieldKind = "text"
	FieldSecret     FieldKind = "secret"
	FieldSelect     FieldKind = "select"
	FieldMulti      FieldKind = "multiselect"
	FieldConfirm    FieldKind = "confirm"
	FieldSegmented  FieldKind = "segmented"
	FieldSlider     FieldKind = "slider"
	FieldDisclosure FieldKind = "disclosure"
)

type Option struct {
	Value    string `json:"value,omitempty"`
	Label    string `json:"label,omitempty"`
	Note     string `json:"note,omitempty"`
	Disabled string `json:"disabled,omitempty"`
}

type Strip struct {
	CPU    Value `json:"cpu,omitzero"`
	Spark  []int `json:"spark,omitempty"`
	Memory Value `json:"memory,omitzero"`
	// value: the memory used and the network's rate, in and out together,
	// over the same readings as Spark (the menu bar's three measures).
	MemorySpark []int   `json:"memory_spark,omitempty"`
	NetSpark    []int   `json:"net_spark,omitempty"`
	Battery     Value   `json:"battery,omitzero"`
	Net         []Value `json:"net,omitempty"`
	// value: a held keep-awake's word and time left, after the battery.
	Awake Value    `json:"awake,omitzero"`
	Top   *Section `json:"top,omitempty"`
}

type Tally struct {
	Mark Mark `json:"mark,omitempty"`
	N    int  `json:"n,omitempty"`
}

type Mark string

const (
	Unjudged  Mark = ""
	OK        Mark = "ok"
	Attention Mark = "attention"
	Absent    Mark = "absent"
	Broken    Mark = "broken"
	Awaiting  Mark = "pending"
	// value: your choice not to manage it here (Off here).
	OffHere Mark = "off-here"
	// value: it does not apply to this machine (Not for this machine).
	NotForThisMachine Mark = "not-for-this-machine"
	// value: present but not station's, or not readable now (Not managed, Unknown).
	NotKnown Mark = "unknown"
)

func Marks() [8]Mark {
	return [8]Mark{OK, Attention, Absent, Broken, Awaiting, OffHere, NotForThisMachine, NotKnown}
}

type Reason string

const (
	Outdated    Reason = "outdated"
	Misplaced   Reason = "misplaced"
	Unreachable Reason = "unreachable"
	Mismatch    Reason = "mismatch"
	Unmanaged   Reason = "unmanaged"
	Missing     Reason = "missing"
	Partial     Reason = "partial"
	Differs     Reason = "differs"
	Unset       Reason = "unset"
	Stopped     Reason = "stopped"
	Late        Reason = "late"
	Degraded    Reason = "degraded"
	Orphan      Reason = "orphan"
	Waiting     Reason = "waiting"
	Failing     Reason = "failing"
	Refused     Reason = "refused"
	Down        Reason = "down"
	Outlived    Reason = "stale"
	Expiring    Reason = "expiring"
	Guard       Reason = "guard"
	Exposed     Reason = "exposed"
)

type Value struct {
	Kind   Kind    `json:"kind,omitempty"`
	Text   string  `json:"text,omitempty"`
	N      int64   `json:"n,omitempty"`
	Of     int64   `json:"of,omitempty"`
	At     string  `json:"at,omitempty"`
	Unit   string  `json:"unit,omitempty"`
	Label  string  `json:"label,omitempty"`
	Items  []Value `json:"items,omitempty"`
	Stale  bool    `json:"stale,omitempty"`
	Live   string  `json:"live,omitempty"`
	Pri    int     `json:"pri,omitempty"`
	Short  string  `json:"short,omitempty"`
	Signal Signal  `json:"signal,omitempty"`
}

type Kind string

const (
	KindText     Kind = "text"
	KindNone     Kind = "none"
	KindCount    Kind = "count"
	KindSize     Kind = "size"
	KindRatio    Kind = "ratio"
	KindPercent  Kind = "percent"
	KindVersion  Kind = "version"
	KindDuration Kind = "duration"
	KindAge      Kind = "age"
	KindUntil    Kind = "until"
	KindURL      Kind = "url"
	KindIdentity Kind = "identity"
	KindPath     Kind = "path"
	KindPort     Kind = "port"
	KindPortMap  Kind = "portmap"
	KindRange    Kind = "range"
	KindCommand  Kind = "command"
	KindList     Kind = "list"
	KindLevels   Kind = "levels"
	KindOn       Kind = "on"
	KindOff      Kind = "off"
	KindUnknown  Kind = "unknown"
	KindPending  Kind = "pending"
	KindListing  Kind = "listing"
	KindSpark    Kind = "spark"
)

const OnSection = "on"

const Outcome ID = "outcome"

func OnMachine(machine string) Section {
	// value: the line an output for another machine starts with (SPEC §8.1).
	return Section{ID: OnSection, Role: Banner, Uncounted: true, Items: []Item{{Note: &Note{Text: "On " + machine + ":"}}}}
}

const (
	// value: the interface's name for the Mac you are on (SPEC §2).
	ThisMac = "This Mac"
)

func MacName(name, self string) string {
	// invariant: how the interface names a Mac (SPEC §2.5): This Mac for the
	// one this binary runs on, named self, and every other by its name
	// without the `<user>-` both names start with (`mac-mini` for
	// `jterrazz-mac-mini`).
	if name == self && self != "" {
		return ThisMac
	}
	owner, _, ok := strings.Cut(self, "-")
	if !ok || owner == "" {
		return name
	}
	if short, ok := strings.CutPrefix(name, owner+"-"); ok && short != "" {
		return short
	}
	return name
}
