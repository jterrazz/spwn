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
    let busy: Set<String>
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now
    @State private var hovering = false

    public init(_ row: Row, actions: [Action] = [], highlighted: Bool = false, busy: Set<String> = [], run: @escaping (Action) -> Void = { _ in }) {
        self.row = row
        self.actions = actions
        self.highlighted = highlighted
        self.busy = busy
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
                let waiting = verb.id.map(busy.contains) == true && verb.busy != nil
                VerbButton(waiting ? (verb.busy ?? "") + Token.Glyph.ellipsis : verb.label ?? "", busy: waiting) { run(verb) }
                    .disabled(waiting)
            }
        }
        .modifier(RowSurface(height: Token.Metric.Need.height, padding: Token.Metric.Need.padding, tinted: row.nav != nil && (hovering || highlighted)))
        .onHover { hovering = $0 }
    }
}

// invariant: a port: its brand's chip, its name and the machine it runs
// on, its address in mono; Share when it only listens, else the pointer
// out; the copy and QR slot reserved beside it.
public struct PortRow: View {
    let row: Row
    let actions: [Action]
    let highlighted: Bool
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens
    @State private var hovering = false

    public init(_ row: Row, actions: [Action] = [], highlighted: Bool = false, run: @escaping (Action) -> Void = { _ in }) {
        self.row = row
        self.actions = actions
        self.highlighted = highlighted
        self.run = run
    }

    var off: Bool { row.signal == "faint" || row.signal == "asleep" }
    var lit: Bool { hovering || highlighted }

    var name: Text {
        let ink = off ? tokens.color.t3 : tokens.tone(Token.Role.rowName.tone)
        let label = Text(verbatim: row.label ?? "").font(Token.Role.rowName.font).foregroundStyle(ink)
        guard let on = row.mac else { return label }
        return Text("\(label)\(Text(verbatim: Token.Glyph.sep + on).font(Token.Role.rowSub.font).foregroundStyle(off ? tokens.color.t3 : tokens.color.t2))")
    }

    public var body: some View {
        let m = Token.Metric.Row.self
        HStack(spacing: m.gap) {
            SwiftUI.Group { if let glyph = row.glyph { GlyphView(glyph) } }
                .frame(width: m.accessory)
                .opacity(off ? m.dimShare : 1)
            VStack(alignment: .leading, spacing: m.subGap) {
                name.lineLimit(1)
                if let address = row.detail, !address.isNone {
                    Text(verbatim: Format.show(address))
                        .font(Token.Role.address.font)
                        .foregroundStyle(off ? tokens.color.t3 : tokens.color.t2)
                        .lineLimit(1)
                        .truncationMode(.middle)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            QuickSlot(action: RowView.action(row.quick, in: actions), lit: lit, size: .small, run: run)
            if let verb = RowView.action(row.verb, in: actions) {
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
        .modifier(RowSurface(height: row.detail == nil ? m.height : m.twoLines, padding: m.padding, tinted: row.nav != nil && lit))
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
