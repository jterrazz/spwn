import SwiftUI

// invariant: SPEC A4's Ports of all machines when the engine sends its
// header; else SPEC §10.2 Ports and A port, vision panels d and g: what is
// reachable outside Tailscale undeclared first, in red with Review; then
// Public; then Private; each row says its class in words. A workspace's
// page (vision d) says instead whether its shares get an https:// link,
// then lists them under no header. A shared port unfolds in its row: its
// link, its QR, Copy link, Open, Stop sharing.
public struct PortsPage: View {
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    @Environment(\.stationPalette) private var palette

    public init(_ state: PanelState, handlers: PopoverHandlers = PopoverHandlers(), scrolls: Bool = true) {
        self.state = state
        self.handlers = handlers
        self.scrolls = scrolls
    }

    var page: Page { state.page ?? Page() }
    var act: Act { Act(state: state, handlers: handlers) }

    public var body: some View {
        if page.header != nil {
            AllPorts(page: page, state: state, handlers: handlers, scrolls: scrolls)
        } else {
            legacy
        }
    }

    @ViewBuilder
    var legacy: some View {
        let summary = page.summary()
        VStack(alignment: .leading, spacing: Token.Tile.section) {
            PageHeader(back: state.back ?? Page.ThisMacWord, title: "Ports", subtitle: Format.show(summary?.cells?.first ?? Value()), action: handlers.back)
            if let red = page.section("needs-you") { needsYou(red) }
            let needing = Self.needing(page)
            if !needing.isEmpty { needsLook(needing) }
            if let https = page.section("https"), let row = https.rows.first {
                MenuRow(row.label ?? "", subtitle: Format.show(row.cells?.first ?? Value()), bare: true) {
                    EmptyView()
                } trailing: {
                    StationSwitch(on: row.mark == .ok).allowsHitTesting(false)
                }
                .stationTile()
            }
            if let public_ = page.section("public") {
                SectionHeader(public_.header)
                Rows {
                    ForEach(Array(public_.rows.enumerated()), id: \.offset) { _, r in plainRow(public_, r, symbol: Token.Symbol.globe) }
                }
            }
            if let private_ = page.section("private") {
                if page.section("https") == nil { SectionHeader(private_.header) }
                privateRows(private_)
            }
            if let ranges = page.section("ranges") {
                ForEach(Array(ranges.notes.enumerated()), id: \.offset) { _, n in FootNote(n) }
            }
            footer
        }
    }

    // value: what is reachable outside Tailscale undeclared, as the
    // vision's red finding: its count in words, why, the ports and what
    // they are, what is known of it, Review.
    func needsYou(_ s: Section) -> some View {
        let rows = s.rows
        let ports = rows.map { r in
            let port = s.cell(r, "port").map { Format.show($0) } ?? ""
            return s.cell(r, "what").map { "\(port) (\(Format.show($0)))" } ?? port
        }
        let review = rows.first.flatMap { page.actions(for: $0.id ?? "").first { $0.run?.terminal == true } }
        let reason = s.title.flatMap { t in t.range(of: ": ").map { String(t[$0.upperBound...]) } } ?? ""
        let why = reason.components(separatedBy: ", ")
        let title = "\(rows.count) \(rows.count == 1 ? "port" : "ports") \(why.first ?? "")"
        let words = why.dropFirst().map(\.capitalizedFirst) + [ports.joined(separator: ", ")] + (s.caption ?? []).map { Format.show($0) }
        return HStack(alignment: .center, spacing: Token.Tile.gap) {
            IconTile(Token.Symbol.warn, active: true, tint: palette.bad)
            VStack(alignment: .leading, spacing: Token.Gap.line) {
                Wrap(title).font(.system(size: Token.Size.text, weight: .semibold)).lineLimit(2)
                SubText(words.filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep))
            }
            .padding(.trailing, Token.Tile.findInset)
            .frame(maxWidth: .infinity, alignment: .leading)
            .layoutPriority(1)
            if let review { StationButton(review.label ?? "") { act.tap(review) } }
        }
        .stationTile(tint: palette.bad)
    }

    // value: SPEC §10.2's Needs you on top when nothing is reachable
    // outside Tailscale undeclared: each port of Public or Private the
    // engine marks, by its name and port, with what it says of it.
    public static func needing(_ page: Page) -> [(title: String, words: String)] {
        guard page.section("needs-you")?.rows.isEmpty ?? true else { return [] }
        return ["public", "private"].compactMap { page.section($0) }.flatMap { s in
            s.rows.filter { $0.mark == .attention || $0.mark == .broken }.map { r in
                let port = s.cell(r, "port").map { Format.show($0) } ?? ""
                let words = [s.cell(r, "what"), s.cell(r, "declared")].compactMap { $0.map { Format.show($0) } }.filter { !$0.isEmpty }
                return ("\(r.short ?? r.label ?? "")\(Token.Glyph.sep)\(port)", words.joined(separator: Token.Glyph.sep))
            }
        }
    }

    func needsLook(_ rows: [(title: String, words: String)]) -> some View {
        HStack(alignment: .center, spacing: Token.Tile.gap) {
            IconTile(Token.Symbol.warn, active: true, tint: palette.bad)
            VStack(alignment: .leading, spacing: Token.Gap.item) {
                ForEach(Array(rows.enumerated()), id: \.offset) { _, r in
                    VStack(alignment: .leading, spacing: Token.Gap.line) {
                        Wrap(r.title).font(.system(size: Token.Size.text, weight: .semibold)).lineLimit(2)
                        SubText(r.words)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .stationTile(tint: palette.bad)
    }

    func line(title: String, detail: String, mark: Mark?, symbol: String?, trailing: Bool = false, open: Bool = false, tap: (() -> Void)? = nil) -> some View {
        MenuRow(title, subtitle: detail, tap: tap) {
            if let symbol { IconTile(symbol) }
        } trailing: {
            if mark == .broken { StateDot(mark: mark) }
            if trailing {
                if open { PlainIcon(Token.Symbol.down) { tap?() } } else { Chevron() }
            }
        }
    }

    func plainRow(_ s: Section, _ r: Row, symbol: String) -> some View {
        let port = s.cell(r, "port").map { Format.show($0) } ?? ""
        let words = [s.cell(r, "what"), s.cell(r, "declared")].compactMap { $0.map { Format.show($0) } }
        return line(title: "\(r.short ?? r.label ?? "")\(Token.Glyph.sep)\(port)", detail: words.joined(separator: Token.Glyph.sep), mark: nil, symbol: symbol, trailing: true)
    }

    func shared(_ s: Section, _ r: Row) -> URL? { s.cell(r, "url").flatMap(Format.url) }

    // value: the private list: a VM's shared ports one row each (its
    // unfolded QR in the list), any other machine's ports one row a
    // machine, its ports after its name and what they are under it.
    func privateRows(_ s: Section) -> some View {
        let rows = s.rows
        let reserved = rows.filter { shared(s, $0) != nil || s.cell($0, "port")?.kind == "portmap" }
        var machines: [(String, [Row])] = []
        for r in rows where !reserved.contains(where: { $0.id == r.id }) {
            let name = r.short ?? r.label ?? ""
            if let i = machines.firstIndex(where: { $0.0 == name }) { machines[i].1.append(r) } else { machines.append((name, [r])) }
        }
        return VStack(alignment: .leading, spacing: 0) {
            ForEach(Array((s.items ?? []).compactMap(\.empty).enumerated()), id: \.offset) { _, empty in
                EmptyState(empty).padding(Token.Tile.rowPadding)
            }
            if rows.isEmpty, (s.items ?? []).allSatisfy({ $0.empty == nil }) {
                EmptyState(Empty(noun: nil, text: "No private port", hint: nil)).padding(Token.Tile.rowPadding)
            }
            ForEach(Array(reserved.enumerated()), id: \.offset) { i, r in
                if i > 0, !unfoldedRow(s, reserved[i - 1]) { RowSeparator() }
                privateRow(s, r)
            }
            ForEach(Array(machines.enumerated()), id: \.offset) { i, m in
                if i > 0 || !reserved.isEmpty { RowSeparator() }
                machineRow(s, m.0, m.1)
            }
        }
        .stationRows()
    }

    func unfoldedRow(_ s: Section, _ r: Row) -> Bool { shared(s, r).map { state_qr($0) } ?? false }

    // value: a machine's private ports as the vision's one row: its name
    // and its ports, what listens there, how Tailscale carries them when
    // it is not plainly on Tailscale.
    func machineRow(_ s: Section, _ name: String, _ rows: [Row]) -> some View {
        let ports = rows.map { "\(s.cell($0, "port")?.n ?? 0)" }
        let what = rows.compactMap { s.cell($0, "process").map { Format.show($0) } }
        var how: [String] = []
        for r in rows {
            if let st = s.cell(r, "state").map({ Format.show($0) }), st != "on Tailscale", !how.contains(st) { how.append(st) }
        }
        let detail = ([what.joined(separator: ", ")] + how).filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
        let broken = rows.contains { $0.mark == .broken }
        return line(title: ([name] + ports).joined(separator: Token.Glyph.sep), detail: detail, mark: broken ? .broken : nil, symbol: Token.Symbol.tailnet, trailing: true)
    }

    // invariant: a VM's shared port names its class in words, its
    // reserved port, under no tile as linux-1's links (vision c, d).
    func privateRow(_ s: Section, _ r: Row) -> some View {
        let port = s.cell(r, "port")
        let url = shared(s, r)
        let reserved = port?.kind == "portmap" ? "reserved port \(port?.of ?? 0)" : nil
        let state = s.cell(r, "state").map { Format.show($0) }
        let profile = Format.show(r.detail ?? Value())
        let what = !profile.isEmpty ? profile.capitalized : "TCP"
        let title = "\(what)\(Token.Glyph.sep)\(port?.n ?? 0)"
        let detail = ["Private", reserved ?? state].compactMap { $0 }.joined(separator: Token.Glyph.sep)
        let open = url.map { state_qr($0) } ?? false
        return VStack(alignment: .leading, spacing: 0) {
            line(title: title, detail: detail, mark: r.mark, symbol: nil,
                 trailing: true, open: open, tap: url.map { u in { handlers.toggleQR(u.absoluteString) } })
            if let url, open { unfolded(r, url) }
        }
    }

    func state_qr(_ url: URL) -> Bool { state.qr == url.absoluteString }

    // value: what a shared port's unfolded row does with its link: Copy
    // link puts it on the pasteboard, Open opens it.
    public static func link(_ url: URL, handlers: PopoverHandlers) -> [(label: String, tap: () -> Void)] {
        [("Copy link", { handlers.copy(url.absoluteString) }), ("Open", { handlers.open(url) })]
    }

    // value: the footnote that declares a port public when tapped, the
    // routes in the words its form offers them (vision panel g ends on it).
    public static func routes(_ declare: Action) -> String {
        let labels = (declare.form?.fields?.first { $0.key == "Via" }?.options ?? []).compactMap(\.label)
        let words = labels.count > 1 ? labels.dropLast().joined(separator: ", ") + " or " + labels[labels.count - 1] : labels.joined()
        return "To open a port on purpose, declare it public: a \(words)."
    }

    func unfolded(_ r: Row, _ url: URL) -> some View {
        let stop = page.actions(for: r.id ?? "").first { $0.id?.hasPrefix("stop-sharing:") == true }
        return SharedPortCard(url: url, stop: stop, page: page, state: state, handlers: handlers)
    }

    var footer: some View {
        let declare = page.action("declare-public")
        let share = page.action("share-port")
        return VStack(alignment: .leading, spacing: Token.Space.gap) {
            if let declare {
                Button { act.tap(declare) } label: { FootNote(Self.routes(declare)) }
                    .buttonStyle(.plain)
                    .accessibilityLabel(act.label(declare))
                    .accessibilityHint(Self.routes(declare))
            }
            if let declare, act.open(declare) { act.sheet(declare) }
            if let share, act.open(share) { act.sheet(share) }
            VStack(alignment: .leading, spacing: 0) {
                if let share, !act.open(share) {
                    Button { act.tap(share) } label: { MenuText(symbol: Token.Symbol.plus, title: act.label(share), key: nil, danger: false) }.buttonStyle(.plain)
                }
            }
        }
    }
}

// value: a shared port unfolded in its row, as the vision's panel d: its
// QR on white, its address, where it works, Copy link the primary action
// and Open beside it, Stop sharing in red under them, its sheet in place.
struct SharedPortCard: View {
    let url: URL
    let stop: Action?
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    @Environment(\.stationPalette) private var palette
    @State private var copied = false

    var body: some View {
        let act = Act(state: state, handlers: handlers)
        let links = PortsPage.link(url, handlers: handlers)
        VStack(alignment: .center, spacing: Token.Tile.gap) {
            QRView(url.absoluteString)
                .padding(Token.Gap.edge)
                .background(RoundedRectangle(cornerRadius: Token.Tile.qrCorner, style: .continuous).fill(palette.qrLight))
                .padding(.top, Token.Gap.small)
            LinkText(url.absoluteString, center: true)
            SubText("Scan with your phone. Works on your tailnet.").multilineTextAlignment(.center)
            HStack(spacing: Token.Tile.pair) {
                ForEach(Array(links.enumerated()), id: \.offset) { i, l in
                    StationButton(i == 0 && copied ? "Copied" : l.label, role: i == 0 ? .primary : .plain, wide: true) {
                        l.tap()
                        if i == 0 { flash() }
                    }
                }
            }
            .fixedSize(horizontal: false, vertical: true)
            if let stop, !act.open(stop) {
                StationButton(stop.label ?? "", role: .danger, wide: true) { act.tap(stop) }.fixedSize(horizontal: false, vertical: true)
            }
            if let stop, act.open(stop) { SheetFor(page: page, state: state, handlers: handlers) }
        }
        .frame(maxWidth: .infinity)
        .padding(EdgeInsets(top: 0, leading: Token.Gap.edge, bottom: Token.Gap.edge, trailing: Token.Gap.edge))
    }

    func flash() {
        copied = true
        Task { @MainActor in
            try? await Task.sleep(nanoseconds: 1_500_000_000)
            copied = false
        }
    }
}

// invariant: SPEC A4's Ports of all machines: the header (the way back, the
// network glyph, Ports, its menu, its facts), each section by its card,
// then the footer: the full list in Terminal, and Share a port.
struct AllPorts: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    @Environment(\.stationTokens) private var tokens

    func tap(_ action: Action) { perform(action, handlers) }

    var body: some View {
        let layer = SheetLayer.of(page, state, handlers)
        Panel(insets: .page, bounded: scrolls, key: "\(state.openings)/\(page.id ?? "")/\(page.context?.machine ?? "")", sheet: state.confirming, layers: AlertLayer.layers(page, state, handlers, sheet: layer)) {
            MachineHeader(page, segment: nil, back: handlers.back, run: tap).modifier(UnderSheet(layer != nil))
        } content: {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                ForEach(Array((page.sections ?? []).filter { $0.id != "summary" }.enumerated()), id: \.offset) { _, s in
                    PortsSection(s, actions: page.actions ?? [], run: tap, qr: handlers.toggleQR, about: { _ in handlers.toggleAbout(s.id ?? "") }, shown: state.qr, opened: state.about != nil && state.about == s.id)
                }
                if state.confirming != nil { SheetFor(page: page, state: state, handlers: handlers) }
            }
            .modifier(UnderSheet(layer != nil))
        } footer: {
            VStack(spacing: Token.Metric.Ports.footerGap) {
                if let status = state.status { StatusBar(status.line, failed: status.failed) }
                footer
            }
            .modifier(UnderSheet(layer != nil))
        }
        .background { BackKey(state: state, handlers: handlers) }
        .environment(\.stationHints, false)
    }

    @ViewBuilder
    var footer: some View {
        let buttons = (page.footer?.buttons ?? []).compactMap { page.action($0) }
        if !buttons.isEmpty {
            HStack(spacing: Token.Metric.Row.gap) {
                ForEach(Array(buttons.enumerated()), id: \.offset) { i, a in
                    if i == 0 && a.run?.terminal == true {
                        Button { tap(a) } label: {
                            Text((a.label ?? "") + " " + Token.Glyph.outward).font(Token.Role.footerLink.font).foregroundStyle(tokens.tone(Token.Role.footerLink.tone))
                        }
                        .buttonStyle(.plain)
                        Spacer(minLength: 0)
                    } else {
                        PanelButton((a.label ?? "") + (a.form != nil ? Token.Glyph.ellipsis : ""), symbol: Token.Symbol.plus, big: true) { tap(a) }
                    }
                }
            }
        }
    }
}
