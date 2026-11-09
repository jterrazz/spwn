import SwiftUI

// invariant: SPEC A4, the Awake page of All Macs: its header, This Mac's card (its
// switch, the picker, the run it follows, the switches its capabilities
// allow), the running work that pauses, then each other Mac on its own
// card; no footer. What the owner just set is drawn set until the page
// read again says it.
public struct AwakePage: View {
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still

    public init(_ state: PanelState, handlers: PopoverHandlers = PopoverHandlers(), scrolls: Bool = true) {
        self.state = state
        self.handlers = handlers
        self.scrolls = scrolls
    }

    var page: Page { state.page ?? Page() }
    var act: Act { Act(state: state, handlers: handlers) }

    // invariant: while one call runs every control of the page waits; once
    // it returned, another may run at once: each call sends the switches as
    // the owner set them (flags), never a stale page's.
    var busy: Bool { state.running != nil }
    func waiting(_ a: Action?) -> Bool { a != nil && (state.running == a?.id || state.settling == a?.id) }

    public var body: some View {
        Panel(insets: .page, bounded: scrolls, key: "\(state.openings)/\(page.id ?? "")", sheet: state.confirming) {
            MachineHeader(page, segment: nil, back: handlers.back)
        } content: {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                ForEach(Array((page.sections ?? []).filter { $0.id != "summary" }.enumerated()), id: \.offset) { _, s in
                    section(s)
                }
                if state.confirming != nil { SheetFor(page: page, state: state, handlers: handlers) }
            }
        } footer: {
            if let status = state.status { StatusBar(status.line, failed: status.failed) }
        }
        .background {
            BackKey(state: state, handlers: handlers)
            if !still, BackKey.gesture(state) == .back { EscapeKey(back: handlers.back) }
        }
        .environment(\.stationHints, false)
    }

    @ViewBuilder
    func section(_ s: Section) -> some View {
        if s.card == nil {
            ForEach(Array(s.notes.enumerated()), id: \.offset) { _, n in footnote(n) }
        } else {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
                CardLabel(s.title ?? "")
                Card {
                    if s.id == "this-mac" { thisMac(s) } else { ForEach(Array(s.rows.enumerated()), id: \.offset) { _, r in other(r) } }
                }
            }
        }
    }

    func footnote(_ text: String) -> some View {
        Text(text).font(Token.Role.footnote.font).foregroundStyle(tokens.tone(Token.Role.footnote.tone))
            .lineSpacing(Token.Metric.Footnote.lineSpacing)
            .fixedSize(horizontal: false, vertical: true)
            .padding(Token.Metric.Footnote.padding)
    }

    func hairline() -> some View {
        Rectangle().fill(tokens.color.hair)
            .frame(height: Token.Metric.Row.hairline)
            .padding(.leading, Token.Metric.Row.insetGlyph)
            .padding(.trailing, Token.Metric.Row.insetEnd)
    }

    var keeps: [Action] {
        (page.actions ?? []).filter { $0.id?.hasPrefix(Self.termPrefix) == true || $0.id == Self.untilID }
    }

    // value: This Mac's card: Keep This Mac awake and its switch, the
    // picker under it, the run it follows, then a line per switch.
    @ViewBuilder
    func thisMac(_ s: Section) -> some View {
        let rows = s.rows
        if let keep = rows.first {
            let verb = RowView.action(keep.verb, in: page.actions ?? [])
            let on = keep.cells?.first?.kind == "on" || state.held[Self.keepKey] != nil
            AwakeMac(keep, on: on, waiting: waiting(verb) || keeps.contains { waiting($0) }, toggle: verb.map { v in { tap(v) } })
                .disabled(busy)
            VStack(alignment: .leading, spacing: Token.Metric.Awake.follows) {
                picker(on: on)
                ForEach(Array(s.notes.enumerated()), id: \.offset) { _, n in
                    Text(n).font(Token.Role.rowSub.font).foregroundStyle(tokens.tone(Token.Role.rowSub.tone))
                }
            }
            .padding(Token.Metric.Awake.picker)
            ForEach(Array(rows.dropFirst().enumerated()), id: \.offset) { i, r in
                if i > 0 { hairline() }
                if let a = RowView.action(r.verb, in: page.actions ?? []) {
                    AwakeMac(r, on: self.on(a), waiting: waiting(a), toggle: { run(a, flipping: true) }).disabled(busy)
                }
            }
        }
    }

    func picker(on: Bool) -> some View {
        let keeps = keeps
        return Segmented(options: keeps.map { k in
            (k.label ?? "", on && Self.picked(k, among: keeps, held: state.held), { k.form == nil ? run(k) : act.tap(k) })
        })
        .disabled(busy)
    }

    func other(_ r: Row) -> some View {
        let change = RowView.action(r.verb, in: page.actions ?? [])
        return AwakeMac(r, on: r.cells?.first?.kind == "on", waiting: waiting(change), label: change?.label ?? "", toggle: change.map { c in { act.tap(c) } })
            .disabled(busy)
    }

    // value: the row's switch: a keep-awake sends the switches as the owner
    // set them; letting it sleep sends nothing more.
    func tap(_ a: Action) {
        if a.id?.hasPrefix("keep") == true { run(a) } else { act.tap(a) }
    }

    static let termPrefix = "keep:"
    static let untilID = "keep-until"

    // value: the key under which the term the owner just picked is held,
    // and the For it sent, until the page read again picks it too.
    public static let keepKey = "Keep"
    public static let forKey = "For"
    public static let untilKey = "Until"

    // value: the segment drawn chosen: the term the owner just picked,
    // else the term the page says is in force; none while a keep-awake
    // follows a run.
    public static func picked(_ k: Action, among keeps: [Action], held chosen: [String: String] = [:]) -> Bool {
        if let id = chosen[keepKey] { return k.id == id }
        return k.picked == true
    }

    // value: what a keep-awake that worked leaves held: its term and the
    // For it sent, so the page drawn before the next read shows it, and a
    // switch flipped meanwhile keeps it.
    public static func holding(_ action: Action, answers: [String: String]) -> [String: String] {
        guard action.id?.hasPrefix(termPrefix) == true || action.id == untilID else { return [:] }
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
}

extension String {
    // value: the words with their first letter capital, as a line of a
    // menu starts.
    var capitalizedFirst: String { prefix(1).uppercased() + dropFirst() }
}
