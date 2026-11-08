import SwiftUI

// invariant: SPEC §10.2's page of one machine, vision panels c and f: its
// name, kind and state, its sections in the vision's order and words,
// its buttons; each action names its call and asks at its call's level.
// A server's one primary action is Restart.
struct MachineContent: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    @Environment(\.stationPalette) private var palette
    @Environment(\.stationNow) private var now
    @Environment(\.stationStill) private var still

    var head: Section? { page.section("machine") }
    var server: Bool { page.context?.kind == "server" }
    var mac: Bool { page.context?.kind == "mac" }

    // value: where a machine's Ports opens: its own Mac's (vision panel g),
    // where a server's ports are shared too.
    var portsOf: String { (server ? page.context?.on : nil) ?? page.context?.machine ?? "" }

    // value: a Mac's way to its Ports (vision panel g): one row under its
    // VMs.
    var portsRow: some View {
        Rows {
            MenuRow("Ports", tap: { handlers.push(Navigation.page(Navigation.ports, of: portsOf)) }) {
                IconTile(Token.Symbol.ports)
            } trailing: {
                Chevron()
            }
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: Token.Tile.section) {
            if let error = page.error { ErrorLine(error) }
            if let head { header(head) }
            if loading {
                skeleton
            } else {
                ForEach(Array((page.sections ?? []).enumerated()), id: \.offset) { _, s in
                    if s.id != "machine" { section(s) }
                }
                if mac { portsRow }
            }
            if state.confirming?.hasPrefix("stop-sharing:") != true {
                SheetFor(page: page, state: state, handlers: handlers)
            }
            if state.confirming == nil || page.action(state.confirming ?? "")?.on == "row" {
                buttons
            }
        }
        .background {
            if !still, BackKey.gesture(state) == .back { EscapeKey(back: handlers.back) }
        }
    }

    // value: the machine stops, or is stopped: its tiles say no figure.
    var stopping: Bool {
        Self.doing(state.running ?? state.settling) == "Stopping" || head?.rows.first?.reason == "stopped"
    }

    // value: the server restarts: its checks are run again after it.
    var restarting: Bool { Self.doing(state.running ?? state.settling) == "Restarting" }

    // value: a server's health checks as drawn: while it restarts, each
    // waits for the boot that follows, never the last boot's answer.
    public static func checks(_ s: Section, restarting: Bool) -> [Row] {
        guard restarting else { return s.rows }
        return s.rows.map { r in
            var r = r
            r.mark = .pending
            r.cells = r.cells?.map { v in
                (v.kind == "list" ? v.items ?? [] : [v]).first { Format.show($0).hasPrefix("http") } ?? Value(kind: "none")
            }
            return r
        }
    }

    public static func measures(_ s: Section, stopping: Bool) -> [(label: String, value: Value?, spark: [Int]?)] {
        s.rows.map { r in (r.label ?? "", stopping ? nil : r.cells?.first, stopping ? nil : r.cells?.dropFirst().first?.points) }
    }

    // value: the port a Share a port that runs, or has just returned,
    // asked for, until the page lists it.
    public static func sharing(_ page: Page, _ state: PanelState) -> String? {
        guard (state.running ?? state.settling) == "share-port", let port = state.sent["Port"], !port.isEmpty else { return nil }
        let listed = (page.section("ports")?.items ?? []).compactMap(\.link).contains { $0.row == port }
            || (page.section("private")?.rows ?? []).contains { $0.id?.hasSuffix(":" + port) == true }
        return listed ? nil : "Port \(port)"
    }

    // value: the page before its first read: its head alone.
    var loading: Bool { (page.sections ?? []).allSatisfy { $0.id == "machine" } && page.error == nil }

    // value: what stands in for a page being read: its measures waiting and
    // a list waiting, about as tall as the page they make way for, so the
    // panel does not jump.
    var skeleton: some View {
        VStack(alignment: .leading, spacing: Token.Tile.section) {
            Metrics(measures: [("CPU", nil, nil), ("Memory", nil, nil), ("Disk", nil, nil)])
            Rows {
                ForEach(0..<3, id: \.self) { _ in
                    HStack(spacing: Token.Tile.gap) {
                        PendingDot()
                        Spacer()
                    }
                    .padding(Token.Tile.rowPadding)
                    .frame(minHeight: Token.Tile.row)
                }
            }
        }
        .accessibilityLabel("Reading")
    }

    func header(_ s: Section) -> some View {
        let row = s.items?.first?.row
        let mark = row?.mark ?? .ok
        var caption = (s.caption ?? []).map { Format.show($0) }
        var badge: String?
        if caption.first == "Server VM" {
            badge = "Server"
            caption.removeFirst()
        }
        let back = state.back ?? Self.all
        return PageHeader(back: back, title: s.title ?? "", subtitle: Self.subtitle(caption), symbol: Token.Symbol.machine(page.context?.kind),
                          badge: badge, action: handlers.back) {
            if let doing = Self.doing(state.running ?? state.settling) {
                StateWord(doing, mark: .pending)
            } else if let word = row?.cells?.first.map({ Format.show($0) }) {
                StateWord(word, mark: mark)
            }
        }
    }

    // value: the header's words, each part kept whole on its line: a
    // wrap breaks between parts, never inside `on This Mac`.
    public static func subtitle(_ parts: [String]) -> String {
        parts.map { $0.replacingOccurrences(of: " ", with: "\u{00A0}") }.joined(separator: Token.Glyph.sep)
    }

    // value: the state word while a call that changes the machine's state
    // runs, and until the page read after it says its outcome.
    public static func doing(_ action: String?) -> String? {
        switch action {
        case "stop": "Stopping"
        case "restart": "Restarting"
        case "start-vm": "Starting"
        case "delete": "Deleting"
        default: nil
        }
    }

    // value: where a page goes back to: All machines, or for a server the
    // Mac it lives on (vision panels c and f).
    static let all = "All machines"

    func words(_ v: Value?) -> String { v.map { Format.show($0, now: now) } ?? "" }

    @ViewBuilder
    func section(_ s: Section) -> some View {
        switch s.id {
        case "activity":
            if server { SectionHeader("Activity") }
            Metrics(measures: Self.measures(s, stopping: stopping))
        case "up":
            HStack(alignment: .top, spacing: Token.Tile.pair) {
                ForEach(Array(s.rows.enumerated()), id: \.offset) { _, r in figure(r) }
            }
            .fixedSize(horizontal: false, vertical: true)
        case "health":
            SectionHeader(s.header)
            Rows {
                ForEach(Array(Self.checks(s, restarting: restarting).enumerated()), id: \.offset) { _, r in health(r) }
            }
        case "inside":
            Rows {
                ForEach(Array(s.rows.enumerated()), id: \.offset) { _, r in
                    let open = page.rowActions(r.id ?? "").first
                    MenuRow(r.label ?? "", tap: open.map { a in { perform(a, handlers) } }) {
                        IconTile(Token.Symbol.line(r.id ?? "") ?? Token.Symbol.forward)
                    } trailing: {
                        TrailingValue(words(r.cells?.first))
                        if open != nil { Chevron() }
                    }
                }
            }
        case "ports":
            SectionHeader("\(s.header)\(Token.Glyph.sep)\((s.items ?? []).compactMap(\.link).count)")
            Rows {
                ForEach(Array((s.items ?? []).compactMap(\.empty).enumerated()), id: \.offset) { _, empty in
                    EmptyState(empty).padding(Token.Tile.rowPadding)
                }
                ForEach(Array((s.items ?? []).compactMap(\.link).enumerated()), id: \.offset) { _, l in link(l) }
                if let port = Self.sharing(page, state) {
                    MenuRow(port, subtitle: "Sharing on Tailscale\(Token.Glyph.ellipsis)") {
                        PendingDot().frame(width: Token.Tile.icon)
                    }
                }
                if let share = page.action("share-port") {
                    MenuRow("\(share.label ?? "")\(Token.Glyph.ellipsis)", tap: { perform(share, handlers) }) {
                        IconTile(Token.Symbol.share)
                    }
                }
            }
        case "shared":
            SectionHeader(s.header)
            Rows {
                ForEach(Array((s.items ?? []).compactMap(\.empty).enumerated()), id: \.offset) { _, empty in
                    EmptyState(empty).padding(Token.Tile.rowPadding)
                }
                ForEach(Array(s.rows.enumerated()), id: \.offset) { _, r in
                    MenuRow(r.label ?? "", subtitle: words(r.cells?.first)) {
                        EmptyView()
                    } trailing: {
                        ForEach(Array(page.rowActions(r.id ?? "").enumerated()), id: \.offset) { _, a in
                            StationButton(a.label ?? "") { perform(a, handlers) }.disabled(state.running != nil)
                        }
                    }
                }
            }
        case "server":
            Rows {
                ForEach(Array(s.rows.enumerated()), id: \.offset) { _, r in serverRow(r) }
            }
            ForEach(Array(s.notes.enumerated()), id: \.offset) { _, n in FootNote(n) }
        case "vms":
            SectionHeader(s.header)
            MachineTree(section: s, kinds: true, only: nil, open: handlers.openMachine)
        default:
            SectionHeader(s.header)
            Rows {
                ForEach(Array((s.items ?? []).enumerated()), id: \.offset) { _, item in
                    if let row = item.row {
                        markRow(s, row)
                    } else if let note = item.note?.text {
                        FootNote(note).padding(Token.Tile.rowPadding)
                    } else {
                        ItemsView([item], columns: s.columns ?? [], width: Token.Space.width - Token.Space.margin * 2).padding(Token.Tile.rowPadding)
                    }
                }
            }
        }
    }

    // value: an Overview figure as the vision's two tiles: its name, its
    // figure in 13 pt bold, its words under it.
    func figure(_ r: Row) -> some View {
        let value = r.cells?.first
        let items = value?.kind == "list" ? value?.items ?? [] : value.map { [$0] } ?? []
        let shown: String
        if r.id == "restarts" {
            shown = "\(Token.Glyph.restart) \(items.first?.n ?? 0)"
        } else {
            shown = items.first.flatMap { Self.span($0) }.map { Format.show($0) }.flatMap { $0.isEmpty ? nil : $0 } ?? Token.Glyph.missing
        }
        let rest = items.dropFirst().map { Format.show($0) }.joined(separator: Token.Glyph.sep)
        return VStack(alignment: .leading, spacing: Token.Gap.small) {
            SubText(r.label ?? "")
            Text(shown).font(.system(size: Token.Size.text, weight: .bold)).monospacedDigit()
            if !rest.isEmpty { SubText(rest) }
        }
        .frame(maxHeight: .infinity, alignment: .topLeading)
        .stationTile()
    }

    // value: the Up figure without the word its label already says: `up
    // 12 d` is shown 12 d.
    static func span(_ v: Value?) -> Value? {
        guard var v, let text = v.text, text.hasPrefix("up ") else { return v }
        v.text = String(text.dropFirst(3))
        return v
    }

    // value: a health check as the vision's: a check in its state's
    // colour, its name, an address under it when it is one, what it
    // answered (or its value) on the right.
    func health(_ r: Row) -> some View {
        let value = r.cells?.first.flatMap { $0.isNone ? nil : $0 }
        let parts = value.map { $0.kind == "list" ? $0.items ?? [] : [$0] } ?? []
        let address = parts.first.flatMap { Format.show($0).hasPrefix("http") ? Format.show($0) : nil }
        let rest = parts.dropFirst(address == nil ? 0 : 1).map { Format.show($0) }.joined(separator: Token.Glyph.sep)
        return MenuRow(r.label ?? "", subtitle: address, link: true, top: true) {
            Image(systemName: r.mark == .ok ? Token.Symbol.health : Token.Symbol.of(r.mark ?? .notKnown))
                .font(.system(size: Token.Size.check, weight: .medium)).foregroundStyle(palette.of(r.mark ?? .notKnown))
                .frame(width: Token.Tile.chevron, height: Token.Size.text + 2 * Token.Size.leading + Token.Gap.tight)
        } trailing: {
            if !rest.isEmpty { TrailingValue(rest) }
        }
    }

    func markRow(_ s: Section, _ r: Row) -> some View {
        let fix = s.columns?.first { $0.kind == "command" }?.key
        let command = fix.flatMap { s.cell(r, $0)?.text }
        let value = (r.cells ?? []).filter { !$0.isNone && $0.text != command }.map { Format.show($0, now: now) }.filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
        let title = Self.sentence(r.short ?? r.label ?? "", in: s)
        let long = value.count > 14 || title.count + value.count > 44
        return MenuRow(title, subtitle: long ? value : nil, titleLines: 2) {
            StateDot(mark: r.mark).frame(width: Token.Tile.icon)
        } trailing: {
            if !value.isEmpty, !long { TrailingValue(value) }
            ForEach(Array(page.rowActions(r.id ?? "").enumerated()), id: \.offset) { _, a in
                TextAction(a.label ?? "") { perform(a, handlers) }.disabled(state.running != nil)
            }
        }
    }

    // value: a row's words as a menu line starts them, capital first,
    // unless they start with a name the owner types as it is (`npm`,
    // `com.docker.socket`, `~/go`): Needs you's lines name what they are.
    static func sentence(_ words: String, in s: Section) -> String {
        guard s.id != TodoBadge.section, let first = words.split(separator: " ").first,
              first.allSatisfy({ $0.isLetter }) else { return words }
        return words.capitalizedFirst
    }

    func serverRow(_ r: Row) -> some View {
        let value = words(r.cells?.first)
        if r.id == "ssh" {
            return AnyView(MenuRow(r.label ?? "", subtitle: value, link: true) {
                IconTile(Token.Symbol.terminal)
            } trailing: {
                CopyIcon { handlers.copy(value) }
            })
        }
        guard r.id == "ports" else {
            return AnyView(MenuRow(r.label ?? "") {
                IconTile(Token.Symbol.line(r.id ?? "") ?? Token.Symbol.forward)
            } trailing: {
                TrailingValue(value)
            })
        }
        return AnyView(MenuRow(r.label ?? "", tap: { handlers.push(Navigation.page(Navigation.ports, of: portsOf)) }) {
            IconTile(Token.Symbol.ports)
        } trailing: {
            TrailingValue(Self.counted(r.cells?.first))
            Chevron()
        })
    }

    // value: a shared port as the vision's line: its name and port, its
    // address under it in the accent, Copy, and for a web link QR and Open
    // at its end; the QR unfolds under it as panel d's card: the link, where
    // it works, Copy link, Open and Stop sharing. A raw TCP port is copied
    // only.
    func link(_ l: Link) -> some View {
        let url = l.value.flatMap(Format.url).flatMap { ["http", "https"].contains($0.scheme ?? "") ? $0 : nil }
        let address = l.value.map { Format.show($0) } ?? ""
        let lead = l.lead.map { Format.show($0) } ?? l.label ?? ""
        let stop = page.rowActions(l.row ?? "").first { $0.id?.hasPrefix("stop-sharing:") == true }
        let open = url.map { state.qr == $0.absoluteString } ?? false
        return VStack(alignment: .center, spacing: 0) {
            MenuRow(lead, subtitle: address, link: true) {
                EmptyView()
            } trailing: {
                HStack(spacing: Token.Gap.block) {
                    CopyIcon { handlers.copy(address) }
                    if let url {
                        PlainIcon(Token.Symbol.qr, picked: open) { handlers.toggleQR(url.absoluteString) }
                        PlainIcon(Token.Symbol.open) { handlers.open(url) }
                    }
                }
            }
            if let url, open {
                SharedPortCard(url: url, stop: stop, page: page, state: state, handlers: handlers)
            }
        }
    }

    // invariant: the page's own actions as the vision's buttons, side by
    // side and as wide: Restart the one primary, Stop, and what destroys
    // in red; Open in Terminal answers its key.
    var buttons: some View {
        let shown = page.pageActions.filter { ["restart", "stop", "start", "start-vm", "delete", "unlock"].contains($0.id ?? "") }
        let terminal = page.action("open-station")
        return HStack(spacing: Token.Tile.pair) {
            ForEach(Array(shown.enumerated()), id: \.offset) { _, a in
                if state.running == a.id {
                    PendingDot().frame(maxWidth: .infinity)
                } else {
                    StationButton(Self.button(a), role: Self.role(a), wide: true) { perform(a, handlers) }
                        .disabled(state.running != nil)
                }
            }
        }
        .fixedSize(horizontal: false, vertical: true)
        .background {
            if let terminal, !still {
                Button("") { handlers.run(terminal, [:]) }.keyboardShortcut("t", modifiers: .command).opacity(0).allowsHitTesting(false)
            }
        }
    }

    // value: a count said as the vision says it, its number first: `Public
    // 1` is 1 public.
    static func counted(_ v: Value?) -> String {
        let parts = (v?.items ?? v.map { [$0] } ?? []).map { Format.show($0) }
        return parts.map { p in
            let words = p.split(separator: " ")
            guard words.count == 2, Int(words[1]) != nil else { return p }
            return "\(words[1]) \(words[0].lowercased())"
        }.joined(separator: Token.Glyph.sep)
    }

    // value: a page button's role: Restart the primary, Delete in red, any
    // other plain and live; a button whose sheet asks for the name says so
    // by its ellipsis alone, the sheet says what to type.
    static func role(_ a: Action) -> StationRole {
        switch a.id {
        case "restart", "start-vm": .primary
        case "delete": .danger
        default: .plain
        }
    }

    static func button(_ a: Action) -> String { label(a) }

    // value: an action that opens Terminal on its own command (Unlock),
    // whose label ends in an ellipsis: it opens something more.
    static func opensTerminal(_ a: Action) -> Bool { a.run?.terminal == true && !(a.run?.argv ?? []).isEmpty && a.id != "open-station" }

    static func label(_ a: Action) -> String {
        (a.form != nil || a.confirm?.type != nil || Self.opensTerminal(a)) ? "\(a.label ?? "")\(Token.Glyph.ellipsis)" : a.label ?? ""
    }
}
