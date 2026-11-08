import SwiftUI

// value: a row's hover tint, its padding and its least height, for the
// rows that lay their own parts: a need, a port, an account.
struct RowSurface: ViewModifier {
    let height: CGFloat
    let padding: CGFloat
    let tinted: Bool
    @Environment(\.stationTokens) private var tokens

    func body(content: Content) -> some View {
        content
            .padding(.horizontal, padding)
            .frame(minHeight: height)
            .background {
                if tinted { RoundedRectangle(cornerRadius: Token.Metric.Row.radius, style: .continuous).fill(tokens.color.q) }
            }
            .contentShape(Rectangle())
            .animation(tokens.motion(Token.Motion.tint), value: tinted)
    }
}

struct QuickSlot: View {
    let action: Action?
    let lit: Bool
    let size: RoundButton.Size
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens

    var body: some View {
        if let action {
            RoundButton(action.icon ?? Token.Symbol.copy, size: size, quiet: true) { run(action) }
                .opacity(lit ? 1 : 0)
                .animation(tokens.motion(Token.Motion.tint), value: lit)
                .offset(x: lit ? 0 : Token.Metric.Row.quickSlide)
                .animation(tokens.motion(Token.Motion.snappy), value: lit)
                .allowsHitTesting(lit)
        }
    }
}

// invariant: a need is one line: its mark in the 24 column, its subject
// in semibold and what it needs, rewritten to fit, then its verb.
public struct NeedRow: View {
    let row: Row
    let actions: [Action]
    let highlighted: Bool
    let running: String?
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now
    @State private var hovering = false

    public init(_ row: Row, actions: [Action] = [], highlighted: Bool = false, running: String? = nil, run: @escaping (Action) -> Void = { _ in }) {
        self.row = row
        self.actions = actions
        self.highlighted = highlighted
        self.running = running
        self.run = run
    }

    var stale: Bool { row.signal == "stale" }

    func line(_ parts: [Value]) -> Text {
        let words = FactsLine.words(parts, now: now)
        let subject = Text(verbatim: row.label ?? "").font(Token.Role.needSubject.font)
        guard !words.isEmpty else { return subject }
        return Text("\(subject)\(Text(verbatim: Token.Glyph.sep + words).font(Token.Role.needWhat.font))")
    }

    public var body: some View {
        let parts = row.detail.map { $0.items ?? [$0] } ?? []
        HStack(spacing: Token.Metric.Row.gap) {
            SwiftUI.Group {
                if let glyph = row.glyph { GlyphView(glyph) } else { MarkView(row.mark) }
            }
            .frame(width: Token.Metric.Row.accessory)
            .opacity(stale ? Token.Metric.Row.staleShare : 1)
            ViewThatFits(in: .horizontal) {
                ForEach(Array(FactsLine.variants(parts).enumerated()), id: \.offset) { _, variant in
                    line(variant).foregroundStyle(tokens.tone(Token.Role.needWhat.tone)).lineLimit(1).fixedSize()
                }
                line(FactsLine.variants(parts).last ?? parts).foregroundStyle(tokens.tone(Token.Role.needWhat.tone)).lineLimit(1)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .opacity(stale ? Token.Metric.Row.staleShare : 1)
            if let verb = RowView.action(row.verb, in: actions) {
                if running != nil, verb.id == running {
                    VerbButton(verb.busy.map { $0 + Token.Glyph.ellipsis } ?? verb.label ?? "", busy: true).allowsHitTesting(false)
                } else {
                    VerbButton(verb.label ?? "") { run(verb) }
                }
            }
        }
        .modifier(RowSurface(height: Token.Metric.Need.height, padding: Token.Metric.Need.padding, tinted: row.nav != nil && (hovering || highlighted)))
        .onHover { hovering = $0 }
    }
}

// invariant: a port: its brand's chip, its name and the machine it runs
// on (or, while its VM does not answer, why, in its signal's colour), its
// address in mono. A row whose verb opens or copies its link is that verb,
// its pointer (↗, or the copy glyph) at its end; any other verb is the
// row's text button (Share); the QR slot is reserved beside it.
public struct PortRow: View {
    let row: Row
    let actions: [Action]
    let highlighted: Bool
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still
    @State private var hovering = false
    @State private var landed = false

    public init(_ row: Row, actions: [Action] = [], highlighted: Bool = false, run: @escaping (Action) -> Void = { _ in }) {
        self.row = row
        self.actions = actions
        self.highlighted = highlighted
        self.run = run
    }

    // value: the role every address of a port row is set in: mono, an IP,
    // a port or a domain alike.
    public static let addressRole = Token.Role.address

    // value: a row's address line split where the address ends: the
    // address, set in mono, and the route a public port is reached by
    // after it, in words (a domain, then Cloudflare Tunnel).
    public static func address(_ detail: Value) -> (address: String, route: String?) {
        guard detail.kind == "list", let items = detail.items, let first = items.first, items.count > 1 else { return (Format.show(detail), nil) }
        return (Format.show(first), items.dropFirst().map { Format.show($0) }.joined(separator: Token.Glyph.sep))
    }

    // value: the row's own verb when it opens or copies its link.
    public static func link(_ row: Row, in actions: [Action]) -> Action? {
        guard let verb = RowView.action(row.verb, in: actions), verb.run?.open != nil || verb.run?.copy != nil else { return nil }
        return verb
    }

    var off: Bool { row.signal == "faint" || row.signal == "asleep" }
    var lit: Bool { hovering || highlighted }
    var link: Action? { Self.link(row, in: actions) }

    var name: Text {
        let ink = off ? tokens.color.t3 : tokens.tone(Token.Role.rowName.tone)
        let label = Text(verbatim: row.label ?? "").font(Token.Role.rowName.font).foregroundStyle(ink)
        if let why = row.cells?.first, let words = why.text, !words.isEmpty {
            return Text("\(label)\(Text(verbatim: Token.Glyph.sep + words).font(Token.Role.rowName.font).foregroundStyle(tokens.signalText(why.signal)))")
        }
        guard let on = row.mac else { return label }
        return Text("\(label)\(Text(verbatim: Token.Glyph.sep + on).font(Token.Role.rowName.font).foregroundStyle(off ? tokens.color.t3 : tokens.color.t2))")
    }

    // value: a row's menu, in order: each of its verbs, nil a separator.
    public static func menu(_ row: Row, in actions: [Action]) -> [Action?] {
        (row.menu ?? []).compactMap { id in id == "-" ? .some(nil) : RowView.action(id, in: actions).map { .some($0) } }
    }

    public var body: some View {
        SwiftUI.Group {
            if let link {
                Button { run(link) } label: { line }
                    .buttonStyle(.plain)
                    .accessibilityLabel(link.label ?? row.label ?? "")
            } else {
                line
            }
        }
        .contextMenu {
            ForEach(Array(Self.menu(row, in: actions).enumerated()), id: \.offset) { _, a in
                if let a { Button(a.label ?? "") { run(a) } } else { Divider() }
            }
        }
    }

    var line: some View {
        let m = Token.Metric.Row.self
        return HStack(spacing: m.gap) {
            SwiftUI.Group { if let glyph = row.glyph { GlyphView(glyph) } }
                .frame(width: m.accessory)
                .opacity(off ? m.dimShare : 1)
            VStack(alignment: .leading, spacing: m.subGap) {
                name.lineLimit(1)
                if let detail = row.detail, !detail.isNone {
                    let (address, route) = Self.address(detail)
                    (Text(verbatim: address).font(Self.addressRole.font)
                        + Text(verbatim: route.map { Token.Glyph.sep + $0 } ?? "").font(Token.Role.rowSub.font))
                        .foregroundStyle(off ? tokens.color.t3 : tokens.color.t2)
                        .lineLimit(1)
                        .truncationMode(.middle)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            QuickSlot(action: RowView.action(row.quick, in: actions), lit: lit, size: .small, run: run)
            if let link {
                Image(systemName: link.run?.open != nil ? Token.Symbol.outward : Token.Symbol.copy)
                    .font(.system(size: Token.Metric.Port.pointerGlyph, weight: .medium))
                    .foregroundStyle(lit ? tokens.color.t1 : tokens.color.t3)
                    .frame(width: Token.Metric.Port.pointer)
            } else if let verb = RowView.action(row.verb, in: actions) {
                Button { run(verb) } label: {
                    Text(verb.label ?? "").font(Token.Role.button.font).foregroundStyle(tokens.color.accentText)
                }
                .buttonStyle(.plain)
            } else if row.nav != nil {
                Image(systemName: Token.Symbol.outward)
                    .font(.system(size: Token.Metric.Port.pointerGlyph, weight: .medium))
                    .foregroundStyle(lit ? tokens.color.t1 : tokens.color.t3)
                    .frame(width: Token.Metric.Port.pointer)
            }
        }
        .modifier(RowSurface(height: row.detail == nil ? m.height : m.twoLines, padding: m.padding, tinted: (row.nav != nil || link != nil) && lit))
        .background {
            if row.fresh == true {
                Color.clear.modifier(FreshTint(progress: still ? FreshTint.frozen(Token.Motion.freshHold / 2) : landed ? 1 : 0))
                    .clipShape(RoundedRectangle(cornerRadius: m.radius, style: .continuous))
            }
        }
        .onAppear {
            guard row.fresh == true, !still else { return }
            withAnimation(Token.Motion.fresh) { landed = true }
        }
        .onHover { hovering = $0 }
    }
}

// invariant: an account: its brand's chip, its name and the VM it is
// given to; what it holds under it; its time left as its trailing value,
// or its verb.
public struct AccountRow: View {
    let row: Row
    let actions: [Action]
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ row: Row, actions: [Action] = [], run: @escaping (Action) -> Void = { _ in }) {
        self.row = row
        self.actions = actions
        self.run = run
    }

    var name: Text {
        let label = Text(verbatim: row.label ?? "").font(Token.Role.rowName.font).foregroundStyle(tokens.tone(Token.Role.rowName.tone))
        guard let target = row.mac else { return label }
        return Text("\(label)\(Text(verbatim: " \(Token.Glyph.arrow) \(target)").font(Token.Role.rowName.font.weight(.regular)).foregroundStyle(tokens.color.t2))")
    }

    public var body: some View {
        let m = Token.Metric.Row.self
        HStack(spacing: m.gap) {
            SwiftUI.Group { if let glyph = row.glyph { GlyphView(glyph) } }
                .frame(width: m.accessory)
            VStack(alignment: .leading, spacing: m.subGap) {
                name.lineLimit(1)
                if let sub = row.detail, !sub.isNone { FactsLine(sub, role: .rowSub) }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            if let left = row.left {
                TimeLeft(left)
            } else if let verb = RowView.action(row.verb, in: actions) {
                VerbButton(verb.label ?? "") { run(verb) }
            }
        }
        .modifier(RowSurface(height: row.detail == nil ? m.height : m.twoLines, padding: m.padding, tinted: false))
    }
}

// invariant: a check of a health or a restart: one 16 pt slot (passed,
// running, next, failed), one text x for every check, its result on the
// trailing side.
public struct CheckRow: View {
    public enum State: Equatable, Sendable { case passed, running, next, failed }

    let row: Row
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now

    public init(_ row: Row) { self.row = row }

    public static func state(_ row: Row) -> State {
        switch row.signal {
        case "well": .passed
        case "active": .running
        case "failing": .failed
        default: row.mark == .ok ? .passed : row.mark == .broken ? .failed : row.mark == .pending ? .running : .next
        }
    }

    @ViewBuilder var slot: some View {
        let size = Token.Metric.Marks.check
        switch Self.state(row) {
        case .passed:
            Image(systemName: Token.Symbol.passed).font(.system(size: size)).symbolRenderingMode(.palette).foregroundStyle(tokens.onTint, tokens.color.green)
        case .failed:
            Image(systemName: Token.Symbol.failed).font(.system(size: size)).symbolRenderingMode(.palette).foregroundStyle(tokens.onTint, tokens.color.red)
        case .running:
            Spinner()
        case .next:
            StatusDot("dash", size: size - Token.Metric.Marks.badgeRing * 2)
        }
    }

    public var body: some View {
        let next = Self.state(row) == .next
        HStack(spacing: Token.Metric.Check.gap) {
            slot.frame(width: Token.Metric.Marks.check, height: Token.Metric.Marks.check)
            Text(row.label ?? "")
                .font(Token.Role.check.font)
                .foregroundStyle(next ? tokens.color.t3 : tokens.tone(Token.Role.check.tone))
                .lineLimit(1)
                .frame(maxWidth: .infinity, alignment: .leading)
            if let result = row.detail, !result.isNone {
                Text(Format.show(result, now: now))
                    .font(Token.Role.checkResult.font)
                    .foregroundStyle(tokens.tone(Token.Role.checkResult.tone))
                    .lineLimit(1)
            }
        }
        .frame(minHeight: Token.Metric.Check.height)
    }
}
