import SwiftUI

// invariant: SPEC §10.2 Awake, vision panel e: This Mac first, Keep awake
// and its switch, Keep awake for, the switches its capabilities allow,
// its guards; then the other Macs by their mode, each with its Change,
// whose y/n sheet opens in its row. No grouping: rows on the panel.
public struct AwakePage: View {
    let state: PanelState
    let handlers: PopoverHandlers
    @Environment(\.stationPalette) private var palette

    public init(_ state: PanelState, handlers: PopoverHandlers = PopoverHandlers()) {
        self.state = state
        self.handlers = handlers
    }

    var page: Page { state.page ?? Page() }
    var act: Act { Act(state: state, handlers: handlers) }

    // invariant: while one call runs every control of the page waits; once
    // it returned, another may run at once: each call sends the switches as
    // the owner set them (flags), never a stale page's.
    var busy: Bool { state.running != nil }
    func waiting(_ a: Action?) -> Bool { a != nil && (state.running == a?.id || state.settling == a?.id) }

    public var body: some View {
        let mac = page.section("this-mac")
        VStack(alignment: .leading, spacing: Token.Tile.section) {
            PageHeader(back: state.back ?? Page.ThisMacWord, title: "Awake", subtitle: "Every Mac\(Token.Glyph.sep)this one first",
                       symbol: Token.Symbol.awake, action: handlers.back)
            if let mac, let row = mac.rows.first { thisMac(mac, row) }
            if let guards = page.section("guards") { guardsSection(guards) }
            if let others = page.section("other-macs") {
                SectionHeader(others.title ?? "")
                Rows {
                    ForEach(Array(others.rows.enumerated()), id: \.offset) { _, r in other(r) }
                }
                ForEach(Array(others.notes.enumerated()), id: \.offset) { _, n in FootNote(n) }
            }
        }
    }

    func words(_ v: Value?) -> String { Format.show(v ?? Value()) }

    // value: This Mac: Keep awake and its switch on a tile, how long as a
    // segmented choice on another, then Lid closed and On battery with
    // their switches in a list.
    @ViewBuilder
    func thisMac(_ s: Section, _ r: Row) -> some View {
        let actions = page.actions(for: r.id ?? "")
        let keeps = actions.filter { $0.id?.hasPrefix("keep") == true }
        let switches = actions.filter { $0.id == "lid" || $0.id == "battery" }
        let on = r.mark == .ok
        let flip = on ? actions.first { $0.id == "let-sleep" } : keeps.first
        SectionHeader(s.title ?? Page.ThisMacWord)
        HStack(alignment: .center, spacing: Token.Tile.gap) {
            RowText(title: "Keep awake", subtitle: words(r.cells?.first))
                .frame(maxWidth: .infinity, alignment: .leading)
            if let flip {
                if waiting(flip) { PendingDot() } else { StationSwitch(on: on) { act.tap(flip) }.disabled(busy) }
            }
        }
        .fixedSize(horizontal: false, vertical: true)
        .stationTile()
        if !keeps.isEmpty {
            VStack(alignment: .leading, spacing: Token.Gap.pair) {
                SubText("Keep awake for\(Token.Glyph.ellipsis)")
                if let running = keeps.first(where: { waiting($0) }) {
                    HStack(spacing: Token.Gap.item) {
                        PendingDot()
                        SubText(act.label(running))
                    }
                } else {
                    Segments(options: keeps.map { k in
                        let chosen = (on || state.held[Self.keepKey] != nil) && Self.picked(k, among: keeps, held: state.held)
                        return (Self.held(k) && chosen ? k.label ?? "" : Self.offered(k), chosen, { k.form == nil ? run(k) : act.tap(k) })
                    }).disabled(busy)
                }
            }
            .stationTile()
            ForEach(Array(keeps.enumerated()), id: \.offset) { _, k in act.sheet(k) }
        }
        if !switches.isEmpty {
            Rows {
                ForEach(Array(switches.enumerated()), id: \.offset) { _, a in toggle(a) }
            }
        }
    }

    // value: Until while a keep-awake holds: it says the time it holds
    // until (`Until 18:00`), the term chosen.
    static func held(_ k: Action) -> Bool { k.form != nil && (k.label ?? "").contains(where: \.isNumber) }

    // value: a segment's words while it is not the term chosen: Until
    // never names the time another term holds until.
    static func offered(_ k: Action) -> String {
        guard k.form != nil else { return k.label ?? "" }
        let word = (k.label ?? "").split(separator: " ").first.map(String.init) ?? ""
        return word + Token.Glyph.ellipsis
    }

    // value: the key under which the term the owner just picked is held,
    // and the For it sent, until the page read again picks it too.
    public static let keepKey = "Keep"
    public static let forKey = "For"
    public static let untilKey = "Until"

    // value: the segment drawn chosen: the term the owner just picked,
    // else the term the page says is in force; from an engine that says
    // none, Until while it holds.
    public static func picked(_ k: Action, among keeps: [Action], held chosen: [String: String] = [:]) -> Bool {
        if let id = chosen[keepKey] { return k.id == id }
        return keeps.contains { $0.picked == true } ? k.picked == true : held(k)
    }

    // value: what a keep-awake that worked leaves held: its term and the
    // For it sent, so the page drawn before the next read shows it, and a
    // switch flipped meanwhile keeps it.
    public static func holding(_ action: Action, answers: [String: String]) -> [String: String] {
        guard action.id?.hasPrefix("keep") == true else { return [:] }
        if let end = answers[untilKey] { return [keepKey: action.id ?? "", untilKey: end] }
        guard let sent = answers[forKey] ?? action.run?.in?[forKey] else { return [:] }
        return [keepKey: action.id ?? "", forKey: sent]
    }

    // invariant: a switch sends the term the owner last picked, by its
    // length or by its end, in place of the one its page was read with:
    // the call is the action with its input set so, never a stale page's.
    public static func sending(_ a: Action, _ page: Page, held: [String: String], flipping: Bool) -> Action {
        var out = a
        var input = a.run?.in ?? [:]
        for (k, v) in flags(page, held: held, flipping: flipping ? a : nil) { input[k] = v }
        if flipping, held[keepKey] != nil {
            if let sent = held[forKey] {
                input[forKey] = sent
                input[untilKey] = nil
            } else if let end = held[untilKey] {
                input[untilKey] = end
                input[forKey] = nil
            }
        }
        out.run?.in = input
        return out
    }

    // invariant: a switch's action flips it, so its input holds the other
    // side: Lid closed is on when its action sends Lid false; a switch the
    // owner just set reads as set until the page says so too.
    func on(_ a: Action) -> Bool {
        let key = Self.key(a)
        if let held = state.held[key] { return held == "true" }
        return Self.paged(a)
    }

    static func key(_ a: Action) -> String { a.id == "lid" ? "Lid" : "Battery" }
    static func paged(_ a: Action) -> Bool { a.run?.in?[key(a)] == "false" }

    // invariant: every Awake call sends both switches as the owner sees
    // them, the one tapped flipped: a call never carries a switch's state
    // from a page read before the last call.
    public static func flags(_ page: Page, held: [String: String], flipping: Action? = nil) -> [String: String] {
        var out: [String: String] = [:]
        for a in page.actions ?? [] where a.id == "lid" || a.id == "battery" {
            let key = key(a)
            let now = held[key].map { $0 == "true" } ?? paged(a)
            out[key] = String(flipping?.id == a.id ? !now : now)
        }
        if flipping != nil, let sent = held[forKey] { out[forKey] = sent }
        return out
    }

    // value: what is still held once a page is read: the switches it does
    // not say yet.
    public static func settled(_ held: [String: String], _ page: Page) -> [String: String] {
        guard page.id == MenuPage.awake else { return held }
        var out = held
        for a in page.actions ?? [] where a.id == "lid" || a.id == "battery" {
            if held[key(a)] == String(paged(a)) { out[key(a)] = nil }
        }
        if let id = held[keepKey], page.actions?.contains(where: { $0.id == id && $0.picked == true }) == true {
            out[keepKey] = nil
            out[forKey] = nil
            out[untilKey] = nil
        }
        return out
    }

    func run(_ a: Action, flipping: Bool = false) {
        handlers.run(Self.sending(a, page, held: state.held, flipping: flipping), [:])
    }

    func toggle(_ a: Action) -> some View {
        let lid = a.id == "lid"
        let on = on(a)
        return MenuRow(a.label ?? "", subtitle: lid ? "Stays awake with the lid shut" : "Stays awake when unplugged") {
            IconTile(lid ? Token.Symbol.lid : Token.Symbol.battery, active: on)
        } trailing: {
            if waiting(a) { PendingDot() } else { StationSwitch(on: on) { run(a, flipping: true) }.disabled(busy) }
        }
    }

    func symbol(_ key: String) -> String {
        switch key {
        case "Temperature": Token.Symbol.thermo
        case "Battery": Token.Symbol.battery
        default: Token.Symbol.screen
        }
    }

    @ViewBuilder
    func guardsSection(_ s: Section) -> some View {
        SectionHeader(s.title ?? "")
        Rows {
            ForEach(Array((s.items ?? []).compactMap(\.fact).enumerated()), id: \.offset) { _, f in
                let parts = (f.value?.kind == "list" ? f.value?.items : nil)?.map { Format.show($0) }
                if let parts, parts.count > 1 {
                    MenuRow("\(f.key ?? "") \(parts[0])", subtitle: parts.dropFirst().joined(separator: Token.Glyph.sep).capitalizedFirst) {
                        IconTile(symbol(f.key ?? ""))
                    }
                } else {
                    MenuRow(f.key ?? "") {
                        IconTile(symbol(f.key ?? ""))
                    } trailing: {
                        TrailingValue(words(f.value).capitalizedFirst)
                    }
                }
            }
        }
        ForEach(Array(s.notes.enumerated()), id: \.offset) { _, n in FootNote(n) }
    }

    func other(_ r: Row) -> some View {
        let change = page.actions(for: r.id ?? "").first { $0.id?.hasPrefix("change:") == true }
        return VStack(alignment: .leading, spacing: Token.Gap.item) {
            HStack(spacing: Token.Tile.gap) {
                IconTile(Token.Symbol.mac)
                RowText(title: r.short ?? r.label ?? "", subtitle: words(r.cells?.first))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .layoutPriority(1)
                StateDot(mark: r.mark)
                if let change { StationButton("\(change.label ?? "")\(Token.Glyph.ellipsis)") { act.tap(change) } }
            }
            if let detail = r.detail { SubText(words(detail).capitalizedFirst) }
            if let change { act.sheet(change) }
        }
        .padding(Token.Tile.groupPadding)
    }
}

extension String {
    // value: the words with their first letter capital, as a line of a
    // menu starts.
    var capitalizedFirst: String { prefix(1).uppercased() + dropFirst() }
}
