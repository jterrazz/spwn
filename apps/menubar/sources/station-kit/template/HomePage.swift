import SwiftUI

// invariant: SPEC A4's Home: the hero the engine decides first; below it,
// until the tiles and the cards land, the machines' list, the Awake and
// Accounts tiles side by side, then the menu. A page from an engine that
// sends no hero keeps today's look: the three measures, what needs you.
struct HomePage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    @State private var unfolded = false
    @State private var lit: Int?
    @Environment(\.stationPalette) private var palette
    @Environment(\.stationStill) private var still

    // value: the machines' lines the arrows move through, as drawn.
    var lines: [Row] {
        guard let machines = page.section("machines") else { return [] }
        return HomeKeys.rows(MachineTree(section: machines, kinds: false, only: nil, open: { _ in }).lines())
    }

    var litID: String? { lit.flatMap { i in lines.indices.contains(i) ? lines[i].id : nil } }

    var hero: Section? { page.section(Self.hero) }
    static let hero = "hero"

    // value: the actions the header and the hero already offer, which the
    // menu does not repeat.
    var carried: [String] { (page.header?.buttons ?? []) + (hero?.items?.first?.empty?.actions ?? []) }

    var body: some View {
        VStack(alignment: .leading, spacing: Token.Tile.section) {
            if let hero {
                Hero(hero, actions: page.actions ?? [], stale: page.header?.stale, run: { verb(page, $0, handlers) }, open: { go($0, handlers) })
                    .onChange(of: Hero.announcement(hero)) { _, said in
                        if let said, !still { AccessibilityNotification.Announcement(said).post() }
                    }
            }
            if hero == nil, let strip = page.strip {
                Button { handlers.push(Navigation.activity) } label: {
                    Metrics(measures: [("CPU", strip.cpu, strip.spark), ("Memory", strip.memory, strip.memory_spark), ("Network", strip.net?.first, strip.net_spark)])
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
            }
            if let needs = page.section(TodoBadge.section), stopped {
                engineStopped(needs)
            } else if hero == nil, let needs = page.section(TodoBadge.section), needs.items?.contains(where: { $0.row != nil }) == true {
                needsYou(needs)
            }
            if let machines = page.section("machines") {
                SectionHeader(machines.title ?? "")
                MachineTree(section: machines, kinds: false, only: nil, lit: litID, stopping: StopAll.stopping(state, page), open: handlers.openMachine)
            }
            tiles
            SheetFor(page: page, state: state, handlers: handlers)
            if state.confirming == nil || page.action(state.confirming ?? "")?.on == "row" {
                menu
            }
        }
        .background {
            if !still, state.confirming == nil {
                ArrowKeys(move: { lit = HomeKeys.moved(lit, by: $0, count: lines.count) },
                          submit: { guard let id = litID else { return false }
                              handlers.openMachine(id)
                              return true })
            }
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

    // value: the folded tile's line, short words as the vision's: each
    // row's words up to their first comma, a launchd label by its last
    // part, without the punctuation that ends them, so the line never
    // ends in `,.`.
    static func folded(_ s: Section) -> String {
        if let glance = s.glance, !glance.isEmpty {
            return glance.map { Format.show($0) }.filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
        }
        return folded((s.items ?? []).compactMap(\.row))
    }

    static func folded(_ rows: [Row]) -> String {
        rows.compactMap(\.label).map(short).filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
    }

    static func short(_ label: String) -> String {
        let head = label.split(separator: ",", maxSplits: 1).first.map(String.init) ?? label
        let words = head.split(separator: " ").map { w -> String in
            let parts = w.split(separator: ".")
            return parts.count > 2 && parts.allSatisfy { !$0.isEmpty } ? String(parts.last!) : String(w)
        }
        return words.joined(separator: " ").trimmingCharacters(in: CharacterSet(charactersIn: " ,.;:"))
    }

    // value: what needs you as the vision's tile: its count in words, what
    // it is under them, a chevron that unfolds each row and its action.
    func needsYou(_ s: Section) -> some View {
        let rows = (s.items ?? []).compactMap(\.row)
        let title = s.caption?.first.map { Format.show($0) } ?? s.title ?? ""
        let worst = TodoBadge.todo(page)
        let tint = (worst?.n ?? 0) == 0 ? nil : worst?.mark == .broken && rows.allSatisfy { $0.mark == .broken } ? palette.bad : palette.warn
        return VStack(alignment: .leading, spacing: Token.Gap.item) {
            Button { unfolded.toggle() } label: {
                HStack(alignment: .center, spacing: Token.Tile.gap) {
                    IconTile(Token.Symbol.warn, tint: tint)
                    VStack(alignment: .leading, spacing: Token.Gap.line) {
                        Text(title).font(.system(size: Token.Size.text, weight: .semibold)).foregroundStyle(Token.Ink.primary).lineLimit(1)
                            .padding(.vertical, Token.Size.leading)
                        if !unfolded {
                            SubText(Self.folded(s)).lineLimit(2)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    Chevron(down: unfolded)
                }
                .fixedSize(horizontal: false, vertical: true)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityValue(unfolded ? "unfolded" : "folded")
            if unfolded {
                ForEach(Array(rows.enumerated()), id: \.offset) { _, row in
                    HStack(spacing: Token.Tile.gap) {
                        StateDot(mark: row.mark).frame(width: Token.Tile.icon)
                        VStack(alignment: .leading, spacing: Token.Gap.line) {
                            Wrap(row.label ?? "").font(.system(size: Token.Size.text)).lineLimit(2)
                                .fixedSize(horizontal: false, vertical: true)
                            if let detail = row.detail, !detail.isNone {
                                SubText(Format.show(detail))
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        ForEach(Array(page.rowActions(row.id ?? "").enumerated()), id: \.offset) { _, a in
                            if state.running == a.id || state.settling == a.id {
                                PendingDot()
                            } else {
                                StationButton(a.label ?? "") { perform(a, handlers) }.disabled(state.running != nil)
                            }
                        }
                    }
                    .frame(minHeight: Token.Line.machine)
                }
            }
        }
        .stationTile()
    }

    // value: the vision's two tiles under the machines, side by side and
    // as tall: Awake with its switch, Accounts with its chevron, each its
    // words under it.
    @ViewBuilder
    var tiles: some View {
        let awake = page.section("awake")
        let accounts = page.section("accounts")
        if awake != nil || accounts != nil {
            HStack(alignment: .top, spacing: Token.Tile.pair) {
                if let awake { awakeLine(awake) }
                if let accounts {
                    let open = page.rowActions(accounts.rows.first?.id ?? "").first
                    homeTile(symbol: Token.Symbol.accounts, active: false, title: accounts.title ?? "",
                          words: accounts.rows.first?.cells?.first.map { Format.show($0) }, tap: open.map { a in { perform(a, handlers) } }) { EmptyView() }
                }
            }
            .fixedSize(horizontal: false, vertical: true)
        }
    }

    func awakeLine(_ s: Section) -> some View {
        let toggle = (page.actions ?? []).first { $0.rows?.contains("awake") == true && $0.run != nil }
        let on = toggle?.id == "awake-off"
        let words = (s.rows.first?.cells?.first).map { Format.show($0) }
        return homeTile(symbol: Token.Symbol.awake, active: on, title: s.title ?? "", words: words, tap: { handlers.push(Navigation.awake) }) {
            if let toggle {
                if state.running == toggle.id || state.settling == toggle.id {
                    PendingDot()
                } else {
                    StationSwitch(on: on) { handlers.run(toggle, state.answers[toggle.id ?? ""] ?? [:]) }.disabled(state.running != nil)
                }
            }
        }
    }

    // invariant: a tile that opens something is one button across it, its
    // words included; what ends its head (a switch) stays its own control,
    // and its words take the tile's whole width under the head, in at most
    // two lines. A tile with nothing else ending its head ends it with a
    // chevron (vision panel a), whatever it opens.
    func homeTile<T: View>(symbol: String, active: Bool, title: String, words: String?, tap: (() -> Void)?,
                           @ViewBuilder trailing: () -> T) -> some View {
        let button = { (label: AnyView) in
            Button { tap?() } label: { label.contentShape(Rectangle()) }.buttonStyle(.plain).allowsHitTesting(tap != nil)
        }
        return VStack(alignment: .leading, spacing: Token.Gap.small) {
            HStack(spacing: Token.Tile.gap) {
                button(AnyView(HStack(spacing: Token.Tile.gap) {
                    IconTile(symbol, active: active)
                    Text(title).font(.system(size: Token.Size.text, weight: .semibold)).foregroundStyle(Token.Ink.primary).lineLimit(1).fixedSize()
                    if tap != nil, T.self == EmptyView.self { Chevron() }
                    Spacer(minLength: 0)
                }))
                trailing().fixedSize()
            }
            .frame(minHeight: Token.Line.card)
            if let words, !words.isEmpty {
                button(AnyView(SubText(words.replacingOccurrences(of: Token.Glyph.dash, with: Token.Glyph.unbroken)).lineLimit(2)
                    .frame(maxWidth: .infinity, alignment: .leading)))
            }
            Spacer(minLength: 0)
        }
        .frame(minWidth: 0, maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .stationTile()
        .contentShape(RoundedRectangle(cornerRadius: Token.Tile.group, style: .continuous))
        .onTapGesture { tap?() }
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

// invariant: a page's keys without a field to type in: Up and Down move
// the line lit, Return opens it; it takes the keys as the page shows, and
// every other key goes on to the panel's shortcuts.
struct ArrowKeys: View {
    let move: (Int) -> Void
    let submit: () -> Bool
    @FocusState private var focused: Bool

    var body: some View {
        Color.clear.frame(width: Token.Detail.anchor, height: Token.Detail.anchor)
            .focusable()
            .focusEffectDisabled()
            .focused($focused)
            .onKeyPress(.downArrow) {
                move(1)
                return .handled
            }
            .onKeyPress(.upArrow) {
                move(-1)
                return .handled
            }
            .onKeyPress(.return) { submit() ? .handled : .ignored }
            .onAppear { DispatchQueue.main.async { focused = true } }
            .accessibilityHidden(true)
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

// invariant: SPEC A4: Home's header: "All machines", then the round +
// (New VM, when the page offers it there) and the round more button of the station
// menu.
struct HomeHeaderBar: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    @Environment(\.stationTokens) private var tokens

    var body: some View {
        let h = Token.HomeHeader.self
        HStack(spacing: h.gap) {
            Text(h.title)
                .font(Token.Role.homeTitle.font)
                .foregroundStyle(tokens.tone(Token.Role.homeTitle.tone))
                .accessibilityAddTraits(.isHeader)
            Spacer(minLength: 0)
            ForEach(Array((page.header?.buttons ?? []).compactMap { page.action($0) }.enumerated()), id: \.offset) { _, a in
                RoundButton(h.plus, size: .topBar) { perform(a, handlers) }
                    .accessibilityLabel(Text(a.label ?? ""))
                    .help(a.label ?? "")
                    .keyboardShortcut("n", modifiers: .command)
                    .disabled(state.running != nil)
            }
            RoundButton(h.menu, size: .topBar, action: handlers.toggleMenu)
                .accessibilityLabel(Text(h.menuLabel))
        }
    }
}
