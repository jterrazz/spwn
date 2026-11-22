import SwiftUI

// invariant: Home V3: under the header's title and find field, one
// section per kind in the page's order: what needs you (only when it
// does), This Mac over its three measures, the servers, the VMs, the Macs,
// then the Keep awake, Ports and Accounts rows; the menu's lines only
// while the station menu is open. A page the app holds while no engine
// answers says what is wrong and offers Start, then the menu.
struct HomePage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    @Environment(\.stationFocus) private var focus
    @Environment(\.stationPalette) private var palette

    init(page: Page, state: PanelState, handlers: PopoverHandlers) {
        self.page = page
        self.state = state
        self.handlers = handlers
    }

    static let needs = "needs"
    static let thisMac = "this-mac"
    static let empty = "empty"
    static let servers = "servers"
    static let vms = "vms"
    static let macs = "macs"
    static let shortcuts = "shortcuts"
    static let addMac = "add-mac"

    // value: the sections Home draws, in the page's order, and those of
    // them whose rows the arrows walk.
    static let drawn = [needs, thisMac, empty, servers, vms, macs, shortcuts]
    static let walked = [needs, servers, vms, macs, shortcuts]

    var listed: Bool { page.section(Self.macs) != nil }

    var litID: String? {
        if case let .row(id)? = focus { return id }
        return state.lit
    }

    // value: the actions the header and the empty card already offer,
    // which the menu does not repeat.
    var carried: [String] {
        (page.header?.buttons ?? []) + (page.section(Self.empty)?.items ?? []).flatMap { $0.empty?.actions ?? [] }
    }

    var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.group, kit: "HomePage") }

    @ViewBuilder var drawing: some View {
        VStack(alignment: .leading, spacing: listed ? Token.HomeCard.blocks : Token.Tile.section) {
            if listed {
                ForEach(Array((page.sections ?? []).filter { Self.drawn.contains($0.id ?? "") }.enumerated()), id: \.offset) { _, s in
                    section(s)
                }
            } else if let needs = page.section(TodoBadge.section), stopped {
                engineStopped(needs)
            }
            SheetFor(page: page, state: state, handlers: handlers)
            if (!listed || state.menu), state.confirming == nil || page.action(state.confirming ?? "")?.on == "row" {
                menu
            }
        }
        .padding(.top, listed ? Token.HomeCard.top : 0)
    }

    // value: one section of Home: its head (its title, its caption), then
    // its box; the shortcuts' rows go unheaded, their verb a switch.
    @ViewBuilder
    func section(_ s: Section) -> some View {
        let actions = page.actions ?? []
        switch s.id {
        case Self.thisMac:
            VStack(alignment: .leading, spacing: 0) {
                HomeHead(s.title ?? "", caption: s.caption ?? [])
                if let strip = page.strip {
                    Button { handlers.push(Navigation.activity) } label: { HomeStrip(strip: strip).contentShape(Rectangle()) }
                        .buttonStyle(.plain)
                }
            }
        case Self.empty:
            VStack(alignment: .leading, spacing: 0) {
                HomeHead(s.title ?? "", caption: s.caption ?? [])
                ForEach(Array((s.items ?? []).compactMap(\.empty).enumerated()), id: \.offset) { _, empty in
                    Card(flush: false) { EmptyState(empty, actions: actions) { perform($0, handlers) } }
                }
            }
        default:
            HomeSection(s, actions: actions, lit: litID, headed: s.id != Self.shortcuts, switches: s.id == Self.shortcuts,
                        run: { verb(page, $0, handlers) }, go: { go($0, handlers) })
        }
    }

    // value: the engine does not answer: what is wrong in a sentence and
    // the one action that helps, Start, on its tile.
    var stopped: Bool { page.section(TodoBadge.section)?.rows.contains { $0.id == "engine" } == true && page.section("machines") == nil }
    static let start = "start"

    func engineStopped(_ s: Section) -> some View {
        let start = page.action(Self.start)
        let row = s.rows.first
        let title = s.caption?.first.map { Format.show($0) } ?? "The engine is stopped"
        let words = row?.hint?.command.map { "Run \($0) in a terminal to set it up." } ?? "Start it to see your machines and what needs you."
        return HStack(alignment: .center, spacing: Token.Tile.gap) {
            IconTile(Token.Symbol.warn, tint: palette.warn)
            VStack(alignment: .leading, spacing: Token.Gap.line) {
                Text(title).font(.system(size: Token.Size.text, weight: .semibold)).foregroundStyle(Token.Ink.primary)
                    .padding(.vertical, Token.Size.leading).fixedSize(horizontal: false, vertical: true)
                SubText(words)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            if let start {
                if state.running == start.id {
                    PendingDot()
                } else {
                    StationButton("Start", role: .primary) { perform(start, handlers) }.disabled(state.running != nil)
                }
            }
        }
        .fixedSize(horizontal: false, vertical: true)
        .stationTile(tint: palette.warn)
    }

    // value: the footer's lines: the page's actions, but Start while the
    // engine-stopped card offers it already.
    static func footer(_ actions: [Action], stopped: Bool, header: [String] = []) -> [Action] {
        (stopped ? actions.filter { $0.id != start } : actions).filter { !header.contains($0.id ?? "") }
    }

    // value: what a call that takes a while says it does from the click
    // until the page says its outcome.
    static func doing(_ a: Action) -> String? {
        switch a.id {
        case "new-vm": "Creating the VM\(Token.Glyph.ellipsis)"
        case "stop-all": "Stopping everything\(Token.Glyph.ellipsis)"
        default: nil
        }
    }

    var menu: some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionDivider().padding(.bottom, Token.Gap.tight)
            ForEach(Array(Self.footer(page.pageActions, stopped: stopped, header: carried).enumerated()), id: \.offset) { _, a in
                if state.running == a.id || (state.settling == a.id && Self.doing(a) != nil) {
                    HStack(spacing: Token.Tile.gap) {
                        PendingDot()
                        SubText(a.id == StopAll.id ? StopAll.progress(state.stopping, page) : Self.doing(a) ?? "\(a.label ?? "")\(Token.Glyph.ellipsis)")
                    }
                    .frame(minHeight: Token.Tile.row - Token.Gap.pair)
                    .padding(.horizontal, Token.Gap.small)
                    .accessibilityElement(children: .combine)
                } else {
                    let line = MenuLine(action: a, symbol: Symbols.of(a), danger: a.id == "stop-all") { perform(a, handlers) }.disabled(state.running != nil)
                    switch a.key {
                    case "cmd+n": line.keyboardShortcut("n", modifiers: .command)
                    case "cmd+t": line.keyboardShortcut("t", modifiers: .command)
                    default: line
                    }
                }
            }
        }
    }
}

// invariant: Stop everything says what it does while the engine does it:
// each VM it stops says Stopping on its own line until the page says it
// stopped, and the footer names the one it waits for and how far it is
// (Stopping linux-1, 1 of 2), then the engine itself.
public enum StopAll {
    public static let id = "stop-all"

    // value: the VMs Stop everything stops: This Mac's workspaces that
    // run, as Home lists them when it is asked; a server is kept.
    public static func vms(_ page: Page?) -> [String] {
        guard let machines = page?.section("machines") else { return [] }
        let mac = (machines.items ?? []).compactMap(\.group).first { $0.head?.label == Page.ThisMacWord }
        return (mac?.items ?? []).flatMap { item -> [Row] in
            if let g = item.group { return (g.items ?? []).compactMap(\.row) }
            return item.row.map { [$0] } ?? []
        }
        .filter { $0.cells?.first?.text == workspace && !stopped($0) }
        .compactMap(\.id)
    }

    static let workspace = "Workspace"

    static func stopped(_ row: Row) -> Bool { row.reason == "stopped" || row.mark == .absent }

    static func row(_ name: String, _ page: Page?) -> Row? {
        func find(_ items: [Item]) -> Row? {
            for item in items {
                if let r = item.row, r.id == name { return r }
                if let g = item.group {
                    if let h = g.head, h.id == name { return h }
                    if let r = find(g.items ?? []) { return r }
                }
            }
            return nil
        }
        return find(page?.section("machines")?.items ?? [])
    }

    // value: the VMs still stopping while Stop everything runs: those it
    // stops whose line does not say stopped yet.
    public static func stopping(_ state: PanelState, _ page: Page?) -> Set<String> {
        guard state.running == id || state.settling == id else { return [] }
        return Set(state.stopping.filter { name in row(name, page).map { !stopped($0) } ?? false })
    }

    public static func progress(_ names: [String], _ page: Page?) -> String {
        let left = names.filter { name in row(name, page).map { !stopped($0) } ?? false }
        guard !names.isEmpty else { return "Stopping everything\(Token.Glyph.ellipsis)" }
        guard let next = left.first else { return "Stopping the engine\(Token.Glyph.ellipsis)" }
        return "Stopping \(next)\(Token.Glyph.ellipsis) \(names.count - left.count + 1) of \(names.count)"
    }

    // value: a VM's line while it stops: Stopping in place of its state,
    // its lifetime kept, its mark pending.
    public static func shown(_ row: Row) -> Row {
        var r = row
        r.mark = .pending
        let lifetime = row.cells?.dropFirst().first?.items?.first
        var said = Value(kind: "list")
        said.items = [lifetime, Value(kind: "text", text: stoppingWord)].compactMap { $0 }
        r.cells = [row.cells?.first ?? Value(kind: "text", text: workspace), said]
        return r
    }

    static let stoppingWord = "Stopping"
}

public enum HomeKeys {
    // value: the rows of a machines' tree, captions left out.
    static func rows(_ lines: [MachineTree.Line]) -> [Row] {
        lines.compactMap { if case let .row(r, _, _) = $0 { r } else { nil } }
    }

    // invariant: Down lights the first line when none is, Up the last;
    // each press moves one line, never past either end.
    public static func moved(_ lit: Int?, by: Int, count: Int) -> Int? {
        guard count > 0 else { return nil }
        return min(max((lit ?? (by > 0 ? -1 : count)) + by, 0), count - 1)
    }
}

// value: a verb that runs something runs it, its form or its question
// first; one that runs nothing opens the page of the row it ends.
@MainActor
func verb(_ page: Page, _ action: Action, _ handlers: PopoverHandlers) {
    if action.run != nil {
        perform(action, handlers)
        return
    }
    let rows = (page.sections ?? []).flatMap { ($0.items ?? []).compactMap(\.row) }
    if let nav = rows.first(where: { $0.verb == action.id && $0.nav != nil })?.nav { go(nav, handlers) }
}

// value: a page by its id: a machine's opens as the machines do, any other
// is pushed by its watch name.
@MainActor
func go(_ nav: String, _ handlers: PopoverHandlers) {
    let machine = PageID.machine + ":"
    if nav.hasPrefix(machine) {
        handlers.openMachine(String(nav.dropFirst(machine.count)))
    } else if !nav.isEmpty {
        handlers.push(nav)
    }
}

// invariant: Home's header as the board draws it: the page's title (a
// small spinner beside it while a Mac is still answering), then its bare
// glyphs, + (New VM, when the page offers it there) and the station menu's
// more; under them, the find field with the page's prompt.
struct HomeHeaderBar: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still
    @Environment(\.stationMenuPick) private var pick

    var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.group, kit: "HomeHeaderBar") }

    @ViewBuilder var drawing: some View {
        let h = Token.HomeHeader.self
        VStack(alignment: .leading, spacing: h.field) {
            HStack(spacing: 0) {
                Text(page.header?.title ?? "")
                    .font(Token.Role.homeTitle.font)
                    .foregroundStyle(tokens.tone(Token.Role.homeTitle.tone))
                    .accessibilityAddTraits(.isHeader)
                if let stale = page.header?.stale {
                    Spinner(small: true).accessibilityLabel(Text(Format.show(stale) + Token.Glyph.ellipsis)).padding(.leading, Token.Gap.small)
                }
                Spacer(minLength: 0)
                ForEach(Array((page.header?.buttons ?? []).compactMap { page.action($0) }.enumerated()), id: \.offset) { _, a in
                    RoundButton(h.plus, size: .bare) { perform(a, handlers) }
                        .accessibilityLabel(Text(a.label ?? ""))
                        .help(a.label ?? "")
                        .keyboardShortcut("n", modifiers: .command)
                        .disabled(state.running != nil)
                        .panelFocus(.control(a.id ?? ""), look: .control, radius: h.button / 2)
                }
                station
                    .accessibilityLabel(Text(h.menuLabel))
                    .menuAnchor(MenuAnchors.page)
                    .panelFocus(.menu, look: .control, radius: h.button / 2)
            }
            .padding(.leading, h.titleInset)
            .padding(.trailing, h.glyphsEnd)
            .frame(height: h.bar)
            if let prompt = page.header?.search {
                FindField(resting: prompt) { handlers.find(FindQuery()) }
                    .panelFocus(.find, look: .control, radius: h.fieldHeight / 2)
                    .padding(.horizontal, h.fieldOut)
            }
        }
    }

    // value: the station menu, the system's own, from the more glyph.
    @ViewBuilder
    var station: some View {
        let h = Token.HomeHeader.self
        let model = MenuModel.station(page)
        if still || model.isEmpty {
            RoundButton(h.menu, size: .bare, action: handlers.toggleMenu)
        } else {
            Image(systemName: h.menu).font(.system(size: h.glyph, weight: .regular))
                .foregroundStyle(tokens.color.t1.opacity(h.glyphShare))
                .frame(width: h.button, height: h.button)
                .contentShape(Circle())
                .systemMenu(model) { pick?($0, nav: nil) }
                .accessibilityElement()
                .accessibilityAddTraits(.isButton)
        }
    }
}

// invariant: a section's head on Home: its title in the row's size, bold,
// at 12 inside its box's edge, its caption at the other end in t2, or in
// its signal's colour when it carries one.
struct HomeHead: View {
    let title: String
    let caption: [Value]
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now

    init(_ title: String, caption: [Value]) {
        self.title = title
        self.caption = caption
    }

    var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.heading, title, kit: "HomeHead") }

    @ViewBuilder var drawing: some View {
        let c = Token.HomeCard.self
        HStack(alignment: .firstTextBaseline) {
            Text(title).font(.system(size: c.headSize, weight: .bold)).foregroundStyle(tokens.color.t1)
                .accessibilityAddTraits(.isHeader)
            Spacer(minLength: 0)
            if !caption.isEmpty {
                Text(FactsLine.words(caption, now: now)).font(.system(size: c.headSize)).monospacedDigit()
                    .foregroundStyle(caption.first?.signal.map(tokens.signalText) ?? tokens.color.t2)
            }
        }
        .lineLimit(1)
        .frame(height: c.headLine)
        .padding(c.head)
    }
}

// invariant: This Mac's three measures in one box, three columns split by
// a hairline: each its name, its value now, its spark in t3 ending in the
// accent's dot, all read live under the strip's keys.
struct HomeStrip: View {
    let strip: Strip
    @Environment(\.stationTokens) private var tokens

    var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.group, kit: "HomeStrip") }

    @ViewBuilder var drawing: some View {
        let c = Token.HomeCard.self
        let measures: [(String, Value?, [Int]?)] = [("CPU", strip.cpu, strip.spark), ("Memory", strip.memory, strip.memory_spark), ("Network", strip.net?.first, strip.net_spark)]
        HStack(alignment: .top, spacing: 0) {
            ForEach(Array(measures.enumerated()), id: \.offset) { i, m in
                LiveRead { store in HomeMeasure(label: m.0, value: store.now(m.1), spark: store.points(m.1, else: m.2)) }
                    .padding(.leading, i == 0 ? c.cell : c.cellAfter)
                    .padding(.trailing, c.cell)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .overlay(alignment: .leading) {
                        if i > 0 { Rectangle().fill(tokens.color.hair).frame(width: Token.Metric.Row.hairline) }
                    }
            }
        }
        .padding(c.strip)
        .frame(maxWidth: .infinity, alignment: .leading)
        .modifier(CardFill(lit: false, radius: Token.Metric.Card.radius))
    }
}

// value: one measure of This Mac's box: its name in 11 pt t2, its value
// in 13 pt semibold (a memory said as what it holds, not what it is out
// of), its spark under it.
struct HomeMeasure: View {
    let label: String
    let value: Value?
    let spark: [Int]?
    @Environment(\.stationNow) private var now
    @Environment(\.stationTokens) private var tokens

    var shown: String {
        guard let value, value.kind != "pending" else { return Token.Glyph.missing }
        let (number, unit, of) = Metric.parts(Format.show(value, now: now))
        if number.isEmpty { return Token.Glyph.missing }
        return of == nil && unit.isEmpty ? number : "\(number) \(unit)"
    }

    var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.group, label, kit: "HomeMeasure") }

    @ViewBuilder var drawing: some View {
        let c = Token.HomeCard.self
        VStack(alignment: .leading, spacing: c.cellGap) {
            Text(label).font(.system(size: c.cellKey)).foregroundStyle(tokens.color.t2).lineLimit(1)
            Text(shown).font(.system(size: c.cellValue, weight: .semibold)).monospacedDigit().foregroundStyle(tokens.color.t1).lineLimit(1)
            RowSpark(Metric.scaled(spark ?? []), size: c.spark).padding(.top, c.sparkTop)
        }
    }
}

// invariant: a section of Home: its head and its caption (unless it goes
// unheaded), then its rows in the page's order in one box, edge to edge, each
// navigable row opening its page; Add a Mac is the accent line under the
// Macs.
struct HomeSection: View {
    let section: Section
    let actions: [Action]
    let lit: String?
    let headed: Bool
    let switches: Bool
    let run: (Action) -> Void
    let go: (String) -> Void
    @Environment(\.stationTokens) private var tokens

    init(_ section: Section, actions: [Action], lit: String?, headed: Bool, switches: Bool,
         run: @escaping (Action) -> Void, go: @escaping (String) -> Void) {
        self.section = section
        self.actions = actions
        self.lit = lit
        self.headed = headed
        self.switches = switches
        self.run = run
        self.go = go
    }

    var rows: [Row] { section.rows.filter { $0.id != HomePage.addMac } }
    var add: Row? { section.rows.first { $0.id == HomePage.addMac } }

    var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.region, section.title, kit: "HomeSection") }

    @ViewBuilder var drawing: some View {
        VStack(alignment: .leading, spacing: 0) {
            if headed {
                HomeHead(section.title ?? "", caption: section.caption ?? [])
            }
            Card(padding: Token.HomeCard.rows) {
                RowStack(rows, columns: section.columns ?? [], actions: actions, highlighted: lit, switches: switches, open: { row in
                    if let nav = row.nav { go(nav) }
                }, run: run)
                if let add, let action = RowView.action(add.verb, in: actions) {
                    addLine(add, action)
                }
            }
            .environment(\.stationHomeRows, true)
            .accessibilityElement(children: .contain)
            .accessibilityLabel(Text(section.title ?? ""))
        }
    }

    func addLine(_ add: Row, _ action: Action) -> some View {
        Button { run(action) } label: {
            HStack(spacing: Token.Metric.Row.gap) {
                Image(systemName: Token.Symbol.plus)
                    .font(.system(size: Token.Metric.Row.chevron, weight: .semibold))
                    .frame(width: Token.HomeRow.lead)
                Text(add.label ?? "").font(Token.Role.rowName.font).lineLimit(1)
                Spacer(minLength: 0)
            }
            .foregroundStyle(tokens.color.accentText)
            .padding(.horizontal, Token.Metric.Row.padding)
            .frame(minHeight: Token.HomeCard.line)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .overlay(alignment: .top) {
            Rectangle().fill(tokens.color.hair).frame(height: Token.Metric.Row.hairline)
                .padding(.leading, Token.HomeRow.inset - Token.HomeCard.rows.leading).padding(.trailing, Token.Metric.Row.insetEnd)
        }
    }
}
