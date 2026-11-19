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
    let switches: Bool
    let run: (Action) -> Void
    let hovered: (Bool) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationFocus) private var focus
    @Environment(\.stationPointerHeld) private var held
    @Environment(\.stationHomeRows) private var home
    @State private var hovering = false

    public init(_ row: Row, columns: [Column] = [], actions: [Action] = [], highlighted: Bool = false, switches: Bool = false,
                run: @escaping (Action) -> Void = { _ in }, hovered: @escaping (Bool) -> Void = { _ in }) {
        self.row = row
        self.columns = columns
        self.actions = actions
        self.highlighted = highlighted
        self.switches = switches
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

    // value: the trailing value when it is a series, the CPU a running
    // row ends in.
    public static func spark(_ row: Row, columns: [Column]) -> Value? {
        guard let value = trail(row, columns: columns), value.kind == "spark" else { return nil }
        return value
    }

    // value: a value without its signal, which its dot says beside it.
    static func unsignalled(_ value: Value) -> Value {
        var v = value
        v.signal = nil
        return v
    }

    // value: a series' value now, under the series' live key.
    static func lead(_ spark: Value) -> Value {
        var lead = spark.items?.first ?? Value()
        lead.live = spark.live
        return lead
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

    var height: CGFloat {
        if home { return sub == nil ? Token.HomeRow.height : Token.HomeRow.twoLines }
        return sub == nil ? m.height : m.twoLines
    }

    public var body: some View {
        let m = Token.Metric.Row.self
        HStack(spacing: m.gap) {
            accessory
                .frame(width: home ? Token.HomeRow.lead : m.accessory)
                .opacity(dim ? m.dimShare : 1)
                .rowPart(.accessory)
            VStack(alignment: .leading, spacing: m.subGap) {
                nameLine.rowPart(.name)
                if let sub {
                    LiveRead { store in FactsLine(store.current(sub), role: .rowSub) }.opacity(stale ? m.staleShare : 1).rowPart(.sub)
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
        .frame(minHeight: height)
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
        if home, let glyph = row.glyph, Self.accessory(row) != .dot {
            HomeLead(glyph: glyph)
        } else if let glyph = row.glyph {
            switch Self.accessory(row) {
            case .dot:
                if let progress = glyph.progress { ProgressRing(perMille: progress) } else { StatusDot(glyph.dot, signal: glyph.signal, size: home ? Token.HomeRow.dot : Token.Metric.Marks.dot) }
            case .glyph, .ring: GlyphView(glyph)
            case .none: Color.clear
            }
        } else if let mark = row.mark {
            MarkView(mark)
        }
    }

    var nameLine: some View {
        let m = Token.Metric.Row.self
        return HStack(alignment: .firstTextBaseline, spacing: home ? Token.HomeRow.suffixGap : m.suffixGap) {
            Text(row.label ?? "")
                .font(home ? Token.Role.homeName.font : Token.Role.rowName.font)
                .foregroundStyle(row.signal == "done" || (home && dim) ? tokens.color.t2 : tokens.tone(Token.Role.rowName.tone))
                .opacity(dim && !home ? m.dimShare : 1)
            if let mac = row.mac {
                let role = home ? Token.Role.homeSuffix : Token.Role.rowSuffix
                Text(mac).font(role.font).foregroundStyle(tokens.tone(role.tone))
            }
        }
        .lineLimit(1)
    }

    @ViewBuilder var trailing: some View {
        if let left = row.left {
            TimeLeft(left).rowPart(.trail)
        } else if let spark = Self.spark(row, columns: columns) {
            HStack(spacing: Token.Metric.Chart.rowSparkGap) {
                LiveRead { store in RowSpark(store.points(spark, else: spark.points) ?? []) }
                LiveText(Self.lead(spark), role: home ? .homeValue : .rowValue).fixedSize()
            }
            .rowPart(.trail)
        } else if let value = Self.trail(row, columns: columns) {
            if home, let signal = value.signal {
                HStack(spacing: Token.HomeRow.valueGap) {
                    StatusDot("fill", signal: signal, size: Token.HomeRow.valueDot)
                    LiveText(Self.unsignalled(value), role: .homeValue).fixedSize()
                }
                .rowPart(.trail)
            } else {
                LiveText(value, role: home ? .homeValue : .rowValue).fixedSize().rowPart(.trail)
            }
        }
    }

    @ViewBuilder var control: some View {
        if switches, let verb = Self.action(row.verb, in: actions) {
            StationSwitch(on: verb.picked == true) { run(verb) }
                .rowPart(.control)
        } else if home, let verb = Self.action(row.verb, in: actions) {
            HomeVerb(verb.label ?? "") { run(verb) }
                .rowPart(.control)
        } else if let verb = Self.action(row.verb, in: actions) {
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
    let switches: Bool
    let open: ((Row) -> Void)?
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationFocus) private var focus
    @Environment(\.stationHomeRows) private var home
    @State private var hovered: Int?

    var lit: String? {
        if case let .row(id)? = focus { return id }
        return highlighted
    }

    public init(_ rows: [Row], columns: [Column] = [], actions: [Action] = [], highlighted: String? = nil, switches: Bool = false,
                open: ((Row) -> Void)? = nil, run: @escaping (Action) -> Void = { _ in }) {
        self.rows = rows
        self.columns = columns
        self.actions = actions
        self.highlighted = highlighted
        self.switches = switches
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
                RowView(row, columns: columns, actions: actions, highlighted: highlighted != nil && row.id == highlighted, switches: switches, run: run) { on in
                    if on { hovered = i } else if hovered == i { hovered = nil }
                }
                .onTapGesture { if row.nav != nil { open?(row) } }
                .stationContextMenu(MenuModel.context(row, actions: actions), nav: row.nav)
                .menuAnchor(MenuAnchors.row(row.id ?? ""))
                .overlay(alignment: .top) {
                    if i > 0 {
                        Rectangle().fill(tokens.color.hair)
                            .frame(height: m.hairline)
                            .padding(.leading, home ? Token.HomeRow.inset - Token.HomeCard.rows.leading : RowView.inset(RowView.accessory(row)))
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

// invariant: a lead on Home that is no dot: a Mac's glyph on its 28 pt
// grey plate, in t1 (t2 while it sleeps), any other glyph bare in t1 at
// three quarters.
struct HomeLead: View {
    let glyph: Glyph
    @Environment(\.stationTokens) private var tokens

    var mac: Bool { ["laptop", "desktop"].contains(glyph.kind ?? "") }

    var body: some View {
        let h = Token.HomeRow.self
        if mac {
            Image(systemName: Token.Object.mac(lid: glyph.kind == "laptop"))
                .font(.system(size: h.plateGlyph, weight: .regular))
                .foregroundStyle(glyph.signal == "quiet" ? tokens.color.t2 : tokens.color.t1)
                .frame(width: h.plate, height: h.plate)
                .background(Circle().fill(tokens.color.q))
                .overlay(Circle().strokeBorder(tokens.color.edge, lineWidth: Token.Metric.Row.hairline))
        } else if case let .symbol(name) = GlyphView.drawn(glyph) {
            Image(systemName: name)
                .font(.system(size: h.glyph, weight: .regular))
                .foregroundStyle(tokens.color.t1.opacity(h.glyphShare))
        } else {
            GlyphView(glyph)
        }
    }
}

// invariant: a row's verb on Home, the board's: a filled accent capsule,
// 24 tall, its word white.
struct HomeVerb: View {
    let title: String
    let action: () -> Void
    @Environment(\.stationTokens) private var tokens

    init(_ title: String, action: @escaping () -> Void) {
        self.title = title
        self.action = action
    }

    var body: some View {
        let h = Token.HomeRow.self
        Button(action: action) {
            Text(title).font(Token.Role.homeVerb.font).foregroundStyle(Color.white).lineLimit(1)
                .padding(h.verb)
                .frame(height: h.verbHeight)
                .background(Capsule().fill(tokens.color.accent))
                .contentShape(Capsule())
        }
        .buttonStyle(.plain)
    }
}
