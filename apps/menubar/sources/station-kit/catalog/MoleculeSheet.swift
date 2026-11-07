import SwiftUI

public struct MoleculeSheet: View {
    public enum Family: String, CaseIterable, Sendable { case rows, lists, blocks }

    let family: Family
    @Environment(\.stationTokens) private var tokens

    public init(_ family: Family) { self.family = family }

    static let samples: [String: Double] = ["cpu:this-mac": 37, "cpu:mac-mini": 82, "cpu:air": 0]

    static func text(_ text: String, pri: Int? = nil, short: String? = nil, signal: String? = nil) -> Value {
        var v = Value(kind: "text", text: text)
        v.pri = pri
        v.short = short
        v.signal = signal
        return v
    }

    static func list(_ parts: Value...) -> Value {
        var v = Value(kind: "list")
        v.items = parts
        return v
    }

    static func glyph(_ kind: String? = nil, signal: String? = nil, dot: String? = nil, ring: String? = nil, spin: Bool? = nil) -> Glyph {
        var g = Glyph()
        g.kind = kind
        g.signal = signal
        g.dot = dot
        g.ring = ring
        g.spin = spin
        return g
    }

    static func action(_ id: String, _ label: String? = nil, icon: String? = nil, picked: Bool? = nil) -> Action {
        var a = Action()
        a.id = id
        a.label = label
        a.icon = icon
        a.picked = picked
        return a
    }

    static func row(_ label: String, glyph: Glyph? = nil, sub: Value? = nil, mac: String? = nil, cells: [Value]? = nil, trail: String? = nil, verb: String? = nil, quick: String? = nil, nav: String? = nil, signal: String? = nil, left: Value? = nil, id: String? = nil) -> Row {
        var r = Row()
        r.id = id ?? label
        r.label = label
        r.glyph = glyph
        r.detail = sub
        r.mac = mac
        r.cells = cells
        r.trail = trail
        r.verb = verb
        r.quick = quick
        r.nav = nav
        r.signal = signal
        r.left = left
        return r
    }

    static func percent(_ tenths: Int64) -> Value {
        var v = Value(kind: "percent")
        v.n = tenths
        return v
    }

    static func free(_ gigabytes: Int64) -> Value {
        var v = Value(kind: "count")
        v.n = gigabytes
        v.unit = "GB free"
        return v
    }

    static func left(_ seconds: Int64, of total: Int64) -> Value {
        var v = Value(kind: "duration")
        v.n = seconds
        v.of = total
        return v
    }

    static let columns: [Column] = {
        var name = Column()
        name.key = "name"
        var value = Column()
        value.key = "value"
        return [name, value]
    }()

    static let actions: [Action] = [
        action("terminal", "Open in Terminal", icon: Token.Symbol.terminal),
        action("new-vm", "New VM", icon: Token.Symbol.plus),
        action("delete", "Delete"),
        action("restart", "Restart"),
        action("review", "Review"),
        action("extend", "Extend"),
        action("share", "Share"),
        action("copy", "Copy", icon: Token.Symbol.copy),
        action("sign-in", "Sign in"),
        action("log", "Show full log"),
        action("awake", "Awake", picked: true),
        action("new-vm-here", "New VM on This Mac", icon: Token.Symbol.plus),
    ]

    static let agents: [Row] = [
        row("review-bot", glyph: glyph("agent", signal: "waits"), sub: text("fixes the token refresh"), mac: "mac-mini", nav: "menubar.machine:review-bot"),
        row("linux-1", glyph: glyph("agent", signal: "active"), sub: list(text("running tests"), text("41/58 pass")), mac: "This Mac", cells: [percent(370)], trail: "value", quick: "terminal", nav: "menubar.machine:linux-1"),
        row("linux-3", glyph: glyph("agent", signal: "active"), sub: text("editing src/auth/refresh.ts"), mac: "mac-mini", cells: [percent(600)], trail: "value", quick: "terminal", nav: "menubar.machine:linux-3"),
        row("agent-7f", glyph: glyph("symbol:\(Token.Symbol.check)", signal: "faint"), sub: text("tests pass"), verb: "delete", signal: "done"),
    ]

    static let accessories: [Row] = [
        row("infra", glyph: glyph(signal: "well", dot: "fill"), sub: list(text("healthy"), text("up 12 days"), text("2 ports")), nav: "menubar.machine:infra"),
        row("infra", glyph: glyph(signal: "failing", dot: "fill"), sub: text("down since 14:02", signal: "failing"), verb: "restart", id: "infra-down"),
        row("This Mac", glyph: glyph("laptop", ring: "cpu:this-mac"), sub: list(text("1 working"), text("4 upkeep")), cells: [free(23)], trail: "value", quick: "new-vm", nav: "menubar.machine:this-mac"),
        row("mac-mini", glyph: glyph("desktop", ring: "cpu:mac-mini"), sub: list(text("1 working"), text("1 waiting")), cells: [free(19)], trail: "value", quick: "new-vm", nav: "menubar.machine:mac-mini", signal: "stale"),
        row("air", glyph: glyph("laptop", signal: "asleep", ring: "cpu:air"), sub: list(text("asleep"), text("seen 2 hr ago")), nav: "menubar.machine:air", signal: "asleep"),
        row("Vite", glyph: glyph("port:Vite"), sub: text("linux-1:5173"), verb: "share"),
        row("Keep this Mac awake", glyph: glyph("symbol:\(Token.Symbol.awake)", signal: "faint"), sub: text("until 18:00"), nav: "menubar.awake"),
    ]

    func card<Content: View>(flush: Bool = true, @ViewBuilder _ content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 0) { content() }
            .padding(flush ? EdgeInsets(top: Token.Metric.Card.flush, leading: Token.Metric.Card.flush, bottom: Token.Metric.Card.flush, trailing: Token.Metric.Card.flush) : Token.Metric.Card.padding)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(RoundedRectangle(cornerRadius: Token.Metric.Card.radius, style: .continuous).fill(tokens.color.card))
    }

    var rows: some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
            card { RowStack(Self.agents, columns: Self.columns, actions: Self.actions, highlighted: "linux-3") }
            card { RowStack(Self.accessories, columns: Self.columns, actions: Self.actions, highlighted: "This Mac") }
        }
    }

    var lists: some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
            card {
                NeedRow(Self.row("review-bot", glyph: Self.glyph("symbol:\(Token.Symbol.share)", signal: "waits"), sub: Self.text("PR #42 is ready"), verb: "review"), actions: Self.actions)
                NeedRow(Self.row("linux-1", glyph: Self.glyph("symbol:\(Token.Symbol.thermo)", signal: "waits"), sub: Self.list(Self.text("Claude ends in 23 min"), Self.text("on This Mac", pri: 1)), verb: "extend"), actions: Self.actions)
            }
            card {
                PortRow(Self.row("Vite", glyph: Self.glyph("port:Vite"), sub: Self.text("linux-1:5173"), nav: "open"), actions: Self.actions)
                PortRow(Self.row("Traefik", glyph: Self.glyph("port:Traefik"), sub: Self.text("jterrazz.com"), mac: "infra", quick: "copy", nav: "open"), actions: Self.actions, highlighted: true)
                PortRow(Self.row("Astro", glyph: Self.glyph("port:Astro"), sub: Self.text("linux-1:4321"), verb: "share"), actions: Self.actions)
                PortRow(Self.row("Postgres", glyph: Self.glyph("port:Postgres"), sub: Self.text("linux-2:5432"), signal: "faint"), actions: Self.actions)
            }
            card {
                AccountRow(Self.row("GitHub", glyph: Self.glyph("account:github"), sub: Self.text("read only"), mac: "linux-1", left: Self.left(13920, of: 36000)), actions: Self.actions)
                AccountRow(Self.row("Claude", glyph: Self.glyph("account:claude"), sub: Self.text("Max plan"), mac: "linux-1", left: Self.left(1380, of: 36000)), actions: Self.actions)
                AccountRow(Self.row("npm", glyph: Self.glyph("account:npm"), sub: Self.text("signed out"), verb: "sign-in"), actions: Self.actions)
            }
            card(flush: false) {
                CheckRow(Self.row("Running", sub: Self.text("VM state running"), signal: "well"))
                CheckRow(Self.row("Has an address", sub: Self.text("checking"), signal: "active"))
                CheckRow(Self.row("SSH answers", sub: Self.text("next")))
                CheckRow(Self.row("Check passes", sub: Self.text("502"), signal: "failing"))
            }
            card(flush: false) {
                StepSlots(Self.steps)
            }
            card(flush: false) {
                LogLines(Self.log, more: Self.actions.first { $0.id == "log" })
            }
        }
    }

    static let steps: [Step] = {
        func step(_ at: String, _ text: String, diff: String? = nil, current: Bool? = nil) -> Step {
            var s = Step()
            s.at = MoleculeSheet.text(at)
            s.text = text
            s.diff = diff
            s.current = current
            return s
        }
        return [
            step("now", "running tests\(Token.Glyph.sep)41/58 pass", current: true),
            step("3 min ago", "edited src/pages/index.astro", diff: "+86 -12"),
            step("7 min ago", "added the pricing section"),
        ]
    }()

    static let log: Log = {
        var l = Log()
        l.lines = [
            "14:30:58 k3s etcd: slow fdatasync, took 1.9 s",
            "14:30:59 kernel Out of memory: killed k3s-server",
            "14:31:00 systemd Reached target Shutdown.",
        ]
        l.cause = 2
        return l
    }()

    static func tile(_ caption: String, glyph: String, number: Value, word: String? = nil, signal: String? = nil, toggle: String? = nil, plus: String? = nil) -> Tile {
        var t = Tile()
        t.caption = caption
        t.glyph = MoleculeSheet.glyph("symbol:\(glyph)")
        t.number = number
        t.word = word
        t.signal = signal
        t.toggle = toggle
        t.plus = plus
        t.nav = caption
        return t
    }

    static func count(_ n: Int64) -> Value {
        var v = Value(kind: "count")
        v.n = n
        return v
    }

    static func gauge(_ title: String, _ value: Value, spark: [Int]? = nil, bar: Value? = nil, stale: Bool? = nil) -> PageGauge {
        var g = PageGauge.init()
        g.title = title
        g.value = value
        g.spark = spark
        g.bar = bar
        g.stale = stale
        return g
    }

    static func share(_ n: Int64, of: Int64) -> Value {
        var v = Value(kind: "ratio")
        v.n = n
        v.of = of
        return v
    }

    static let empty: Empty = {
        var e = Empty()
        e.text = "No VMs yet"
        e.sub = "35 GB free on This Mac"
        e.glyph = MoleculeSheet.glyph("symbol:\(Token.Symbol.workspace)")
        e.actions = ["new-vm-here"]
        return e
    }()

    static let hud: Value = Value(kind: "text", text: "Copied ssh linux-1")

    var blocks: some View {
        let tiles = Token.Metric.Tiles.self
        return VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
            HStack(spacing: tiles.gap) {
                TileView(Self.tile("Ports", glyph: Token.Symbol.ports, number: Self.count(5), word: "shared", plus: "+1"))
                TileView(Self.tile("Accounts", glyph: Token.Symbol.key, number: Self.count(1), word: "signed out", signal: "waits"), highlighted: true)
                TileView(Self.tile("Awake", glyph: Token.Symbol.awake, number: Self.left(14400, of: 14400), word: "left", toggle: "awake"), actions: Self.actions)
            }
            HStack(spacing: Token.Metric.Gauges.gap) {
                GaugeTile(Self.gauge("CPU\(Token.Glyph.sep)4 cores", Self.percent(310), spark: [12, 18, 15, 22, 30, 41, 35, 28, 31, 30, 33, 31]))
                GaugeTile(Self.gauge("Memory", Self.free(3), bar: Self.share(60, of: 100)))
                GaugeTile(Self.gauge("Disk", Self.free(2), bar: Self.share(97, of: 100)))
            }
            HStack(spacing: Token.Metric.Gauges.gap) {
                GaugeTile(Self.gauge("CPU\(Token.Glyph.sep)last", Self.percent(200), spark: [20, 21], stale: true))
                GaugeTile(Self.gauge("Memory\(Token.Glyph.sep)last", Self.free(10), bar: Self.share(88, of: 100), stale: true))
                GaugeTile(Self.gauge("Last seen", Self.text("14:30"), stale: true))
            }
            HStack {
                FoldButton("3 more", open: false)
                FoldButton("Show less", open: true)
            }
            card(flush: false) { EmptyState(Self.empty, actions: Self.actions) }
            HStack {
                Spacer(minLength: 0)
                HUDCapsule(Self.hud, symbol: Token.Symbol.copy)
                Spacer(minLength: 0)
            }
        }
    }

    public var body: some View {
        SwiftUI.Group {
            switch family {
            case .rows: rows
            case .lists: lists
            case .blocks: blocks
            }
        }
        .environment(\.stationSamples, Self.samples)
        .padding(Token.Metric.Panel.body)
        .frame(width: Token.Metric.Panel.width, alignment: .leading)
        .background(tokens.color.panelSolid)
    }
}
