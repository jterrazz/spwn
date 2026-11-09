import SwiftUI

// value: the parts of a row, by the frame each takes in its row.
public enum RowPart: Hashable, Sendable { case accessory, name, sub, quick, trail, control }

public struct RowFrames: PreferenceKey {
    public static let defaultValue: [RowPart: CGRect] = [:]
    public static func reduce(value: inout [RowPart: CGRect], nextValue: () -> [RowPart: CGRect]) {
        value.merge(nextValue()) { $1 }
    }
}

extension View {
    func rowPart(_ part: RowPart) -> some View {
        background(GeometryReader { g in Color.clear.preference(key: RowFrames.self, value: [part: g.frame(in: .named(RowView.space))]) })
    }
}

// invariant: SPEC A2.1, one grammar for every row of the panel:
// [accessory 24] [name line / sub line] [quick slot] [one trailing value]
// [one trailing control]. The quick slot is reserved whether it shows or
// not, so nothing moves under the pointer; a verb takes the control's
// place and the chevron goes.
public struct RowView: View {
    public enum Accessory: Equatable, Sendable { case none, dot, glyph, ring }

    static let space = "station.row"

    let row: Row
    let columns: [Column]
    let actions: [Action]
    let highlighted: Bool
    let run: (Action) -> Void
    let hovered: (Bool) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationFocus) private var focus
    @Environment(\.stationPointerHeld) private var held
    @State private var hovering = false

    public init(_ row: Row, columns: [Column] = [], actions: [Action] = [], highlighted: Bool = false, run: @escaping (Action) -> Void = { _ in }, hovered: @escaping (Bool) -> Void = { _ in }) {
        self.row = row
        self.columns = columns
        self.actions = actions
        self.highlighted = highlighted
        self.run = run
        self.hovered = hovered
    }

    public static func accessory(_ row: Row) -> Accessory {
        guard let glyph = row.glyph else { return .none }
        if (glyph.kind ?? "").isEmpty { return glyph.dot == nil ? .none : .dot }
        if case .mac = GlyphView.drawn(glyph) { return .ring }
        return .glyph
    }

    // value: where the hairline above a row starts, by its accessory.
    public static func inset(_ accessory: Accessory) -> CGFloat {
        let m = Token.Metric.Row.self
        switch accessory {
        case .ring: return m.insetRing
        case .glyph: return m.insetGlyph
        case .dot, .none: return m.inset
        }
    }

    // value: the one cell drawn as the trailing value: the cell under the
    // column the row's trail names (a row's cells start at the second
    // column, the first being its label).
    public static func trail(_ row: Row, columns: [Column]) -> Value? {
        guard let key = row.trail, let i = columns.firstIndex(where: { $0.key == key }), i > 0 else { return nil }
        let cells = row.cells ?? []
        return i - 1 < cells.count ? cells[i - 1] : nil
    }

    static func action(_ id: String?, in actions: [Action]) -> Action? {
        guard let id else { return nil }
        return actions.first { $0.id == id || $0.key == id }
    }

    var keyed: Bool { row.id.map { focus == .row($0) } ?? false }
    var lit: Bool { highlighted || hovering || keyed }
    var navigable: Bool { row.nav != nil }
    var dim: Bool { row.signal == "asleep" || row.signal == "faint" }
    var stale: Bool { row.signal == "stale" }
    var sub: Value? { row.detail.flatMap { $0.isNone ? nil : $0 } }

    public var body: some View {
        let m = Token.Metric.Row.self
        HStack(spacing: m.gap) {
            accessory
                .frame(width: m.accessory)
                .opacity(dim ? m.dimShare : 1)
                .rowPart(.accessory)
            VStack(alignment: .leading, spacing: m.subGap) {
                nameLine.rowPart(.name)
                if let sub {
                    FactsLine(sub, role: .rowSub).opacity(stale ? m.staleShare : 1).rowPart(.sub)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            if let quick = Self.action(row.quick, in: actions) {
                RoundButton(quick.icon ?? Token.Symbol.terminal, size: Self.accessory(row) == .ring ? .small : .regular) { run(quick) }
                    .accessibilityLabel(Text(quick.label ?? ""))
                    .opacity(lit ? 1 : 0)
                    .animation(tokens.motion(Token.Motion.tint), value: lit)
                    .offset(x: lit ? 0 : m.quickSlide)
                    .animation(tokens.motion(Token.Motion.snappy), value: lit)
                    .allowsHitTesting(lit)
                    .rowPart(.quick)
            }
            trailing.opacity(stale ? m.staleShare : 1)
            control
        }
        .padding(.horizontal, m.padding)
        .frame(minHeight: sub == nil ? m.height : m.twoLines)
        .overlay(alignment: .bottom) { if let birth = row.birth { BirthBar(birth) } }
        .background {
            if navigable && lit {
                RoundedRectangle(cornerRadius: m.radius, style: .continuous).fill(tokens.color.q)
            }
        }
        .contentShape(Rectangle())
        .coordinateSpace(name: Self.space)
        .onContinuousHover { phase in
            let on: Bool
            switch phase {
            case .active: on = !held
            case .ended: on = false
            }
            guard on != hovering else { return }
            hovering = on
            hovered(on)
        }
        .modifier(RowFocus(id: row.id, navigable: navigable || row.verb != nil))
        .animation(tokens.motion(Token.Motion.tint), value: lit)
    }

    @ViewBuilder var accessory: some View {
        if let glyph = row.glyph {
            switch Self.accessory(row) {
            case .dot:
                if let progress = glyph.progress { ProgressRing(perMille: progress) } else { StatusDot(glyph.dot, signal: glyph.signal) }
            case .glyph, .ring: GlyphView(glyph)
            case .none: Color.clear
            }
        } else if let mark = row.mark {
            MarkView(mark)
        }
    }

    var nameLine: some View {
        let m = Token.Metric.Row.self
        return HStack(alignment: .firstTextBaseline, spacing: m.suffixGap) {
            Text(row.label ?? "")
                .font(Token.Role.rowName.font)
                .foregroundStyle(row.signal == "done" ? tokens.color.t2 : tokens.tone(Token.Role.rowName.tone))
                .opacity(dim ? m.dimShare : 1)
            if let mac = row.mac {
                Text(mac).font(Token.Role.rowSuffix.font).foregroundStyle(tokens.tone(Token.Role.rowSuffix.tone))
            }
        }
        .lineLimit(1)
    }

    @ViewBuilder var trailing: some View {
        if let left = row.left {
            TimeLeft(left).rowPart(.trail)
        } else if let value = Self.trail(row, columns: columns) {
            LiveText(value, role: .rowValue).fixedSize().rowPart(.trail)
        }
    }

    @ViewBuilder var control: some View {
        if let verb = Self.action(row.verb, in: actions) {
            VerbButton(verb.label ?? "") { run(verb) }
                .padding(.trailing, m.verbTrail)
                .rowPart(.control)
        } else if navigable {
            Image(systemName: Token.Symbol.forward)
                .font(.system(size: m.chevron, weight: .semibold))
                .foregroundStyle(tokens.color.t3)
                .rowPart(.control)
        }
    }

    var m: Token.Metric.Row.Type { Token.Metric.Row.self }
}

// invariant: SPEC A4, B5.10: a VM being created carries how far its
// create is as a bar at its row's foot.
struct BirthBar: View {
    let share: Double
    @Environment(\.stationTokens) private var tokens

    init(_ birth: Value) {
        share = (birth.of ?? 0) > 0 ? min(max(Double(birth.n ?? 0) / Double(birth.of ?? 1), 0), 1) : 0
    }

    var body: some View {
        let m = Token.Metric.Birth.self
        Capsule().fill(tokens.color.track)
            .frame(height: m.bar)
            .overlay(alignment: .leading) {
                GeometryReader { g in Capsule().fill(tokens.color.accent).frame(width: g.size.width * share) }
            }
            .animation(tokens.motion(Token.Motion.progress), value: share)
            .padding(m.inset)
    }
}

// invariant: rows in a card: each row after the first has its inset
// hairline, and a tinted row hides its own and the next one's.
public struct RowStack: View {
    let rows: [Row]
    let columns: [Column]
    let actions: [Action]
    let highlighted: String?
    let open: ((Row) -> Void)?
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationFocus) private var focus
    @State private var hovered: Int?

    var lit: String? {
        if case let .row(id)? = focus { return id }
        return highlighted
    }

    public init(_ rows: [Row], columns: [Column] = [], actions: [Action] = [], highlighted: String? = nil, open: ((Row) -> Void)? = nil,
                run: @escaping (Action) -> Void = { _ in }) {
        self.rows = rows
        self.columns = columns
        self.actions = actions
        self.highlighted = highlighted
        self.open = open
        self.run = run
    }

    // value: whether the hairline above row i shows: hidden when that row
    // or the one before it is tinted, and only a navigable row tints.
    public static func hairline(_ i: Int, of rows: [Row], hovered: Int?, highlighted: String?) -> Bool {
        func tinted(_ j: Int) -> Bool {
            guard rows.indices.contains(j), rows[j].nav != nil else { return false }
            return hovered == j || (highlighted != nil && rows[j].id == highlighted)
        }
        return i > 0 && !tinted(i) && !tinted(i - 1)
    }

    public var body: some View {
        let m = Token.Metric.Row.self
        VStack(spacing: 0) {
            ForEach(Array(rows.enumerated()), id: \.offset) { i, row in
                RowView(row, columns: columns, actions: actions, highlighted: highlighted != nil && row.id == highlighted, run: run) { on in
                    if on { hovered = i } else if hovered == i { hovered = nil }
                }
                .onTapGesture { if row.nav != nil { open?(row) } }
                .stationContextMenu(MenuModel.context(row, actions: actions), nav: row.nav)
                .menuAnchor(MenuAnchors.row(row.id ?? ""))
                .overlay(alignment: .top) {
                    if i > 0 {
                        Rectangle().fill(tokens.color.hair)
                            .frame(height: m.hairline)
                            .padding(.leading, RowView.inset(RowView.accessory(row)))
                            .padding(.trailing, m.insetEnd)
                            .opacity(Self.hairline(i, of: rows, hovered: hovered, highlighted: lit) ? 1 : 0)
                    }
                }
                .caughtRow(row.id)
            }
        }
    }
}

struct RowFocus: ViewModifier {
    let id: String?
    let navigable: Bool

    func body(content: Content) -> some View {
        if let id, navigable {
            content.panelFocus(.row(id), look: .row, radius: Token.Metric.Row.radius)
        } else {
            content
        }
    }
}
