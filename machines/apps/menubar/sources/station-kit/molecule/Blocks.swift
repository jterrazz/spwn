import SwiftUI

// value: a shown value split into its figure and its unit at its first
// space ("11.0 GB free": "11.0", "GB free"); a percent keeps its sign
// with its figure ("37%"), and a value with no space is all figure.
public enum Figure {
    public static func split(_ shown: String) -> (figure: String, unit: String) {
        guard let space = shown.firstIndex(where: { $0 == " " || $0 == "\u{00A0}" }) else { return (shown, "") }
        let unit = String(shown[shown.index(after: space)...])
        return unit == Token.Glyph.percent ? (shown, "") : (String(shown[..<space]), unit)
    }
}

struct CardFill: ViewModifier {
    let lit: Bool
    let radius: CGFloat
    @Environment(\.stationTokens) private var tokens

    func body(content: Content) -> some View {
        let shape = RoundedRectangle(cornerRadius: radius, style: .continuous)
        content
            .background(shape.fill(lit ? tokens.color.card2 : tokens.color.card))
            .overlay { shape.strokeBorder(tokens.cardEdge, lineWidth: Token.Metric.Card.edge) }
            .animation(tokens.motion(Token.Motion.tint), value: lit)
    }
}

// invariant: a gauge: its title, its value on the shared baseline with
// its unit smaller, its visual (a spark or a bar) in the 14 pt bottom
// slot; a stale gauge says its last value in t3.
public struct GaugeTile: View {
    let gauge: PageGauge
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationSamples) private var samples
    @Environment(\.stationNow) private var now

    public init(_ gauge: PageGauge) { self.gauge = gauge }

    var stale: Bool { gauge.stale == true }

    func value(_ value: Value?) -> some View {
        let shown = Figure.split(value.map { Format.show(Live.value($0, samples: samples), now: now) } ?? "")
        let figure = stale || value?.stale == true ? tokens.color.t3 : tokens.tone(Token.Role.gaugeValue.tone)
        let big = Text(verbatim: shown.figure).font(Token.Role.gaugeValue.font).foregroundStyle(figure)
        let unit = Text(verbatim: shown.unit.isEmpty ? "" : " " + shown.unit).font(Token.Role.gaugeUnit.font).foregroundStyle(stale ? tokens.color.t3 : tokens.tone(Token.Role.gaugeUnit.tone))
        return Text("\(big)\(unit)").lineLimit(1)
    }

    static func share(_ bar: Value) -> Double {
        guard let of = bar.of, of > 0 else { return 0 }
        return min(max(Double(bar.n ?? 0) / Double(of), 0), 1)
    }

    func bar(_ bar: Value) -> some View {
        let m = Token.Metric.GaugeParts.self
        let share = Self.share(bar)
        let ink = share > m.full ? tokens.color.red : share > m.hot ? tokens.color.orange : tokens.color.accent
        return Capsule().fill(tokens.color.track)
            .frame(height: m.bar)
            .overlay(alignment: .leading) {
                GeometryReader { g in Capsule().fill(stale ? tokens.color.t3 : ink).frame(width: g.size.width * share) }
            }
            .animation(tokens.motion(Token.Motion.sample), value: share)
    }

    public var body: some View {
        let m = Token.Metric.Gauges.self
        VStack(alignment: .leading, spacing: 0) {
            Text(gauge.heading)
                .font(Token.Role.gaugeTitle.font)
                .foregroundStyle(stale ? tokens.color.t3 : tokens.tone(Token.Role.gaugeTitle.tone))
                .lineLimit(1)
            LiveRead { store in value(store.now(gauge.value) ?? gauge.value) }
            Spacer(minLength: 0)
            SwiftUI.Group {
                if let bar = gauge.bar {
                    self.bar(bar)
                } else if let spark = gauge.spark, !spark.isEmpty, !stale {
                    Spark(spark, height: m.visual)
                }
            }
            .frame(height: m.visual)
        }
        .padding(m.padding)
        .frame(maxWidth: .infinity, alignment: .leading)
        .frame(height: m.height)
        .modifier(CardFill(lit: false, radius: Token.Metric.Card.radius))
    }
}

// invariant: a fold's own button: the page's words for it ("3 more",
// "Show less"), its chevron turned when it is open.
public struct FoldButton: View {
    let label: String
    let open: Bool
    let tap: () -> Void
    @Environment(\.stationTokens) private var tokens
    @State private var hovering = false

    public init(_ label: String, open: Bool, tap: @escaping () -> Void = {}) {
        self.label = label
        self.open = open
        self.tap = tap
    }

    public var body: some View {
        let m = Token.Metric.Fold.self
        Button(action: tap) {
            HStack(spacing: m.gap) {
                Text(label).font(Token.Role.fold.font).lineLimit(1)
                Image(systemName: Token.Symbol.down)
                    .font(.system(size: m.chevron, weight: .semibold))
                    .rotationEffect(open ? m.turned : .zero)
                    .animation(tokens.motion(Token.Motion.smooth), value: open)
            }
            .foregroundStyle(hovering ? tokens.color.t1 : tokens.tone(Token.Role.fold.tone))
            .padding(m.padding)
            .background(RoundedRectangle(cornerRadius: Token.Metric.Row.radius, style: .continuous).fill(hovering ? tokens.color.q : Color.clear))
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .onHover { hovering = $0 }
        .animation(tokens.motion(Token.Motion.tint), value: hovering)
        .accessibilityValue(open ? "open" : "closed")
    }
}

// invariant: an empty place: its symbol, a bold line, a grey line and one
// or two buttons, centred; an Empty with none of them is today's one
// muted line.
public struct EmptyState: View {
    let empty: Empty
    let actions: [Action]
    let run: (Action) -> Void
    @Environment(\.stationPalette) private var palette
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationHints) private var hints

    let centred: Bool

    public init(_ empty: Empty, actions: [Action] = [], centred: Bool = false, run: @escaping (Action) -> Void = { _ in }) {
        self.empty = empty
        self.actions = actions
        self.centred = centred
        self.run = run
    }

    var full: Bool { centred || empty.glyph != nil || empty.sub != nil || !(empty.actions ?? []).isEmpty }

    var symbol: String? {
        guard let glyph = empty.glyph, case let .symbol(name) = GlyphView.drawn(glyph) else { return nil }
        return name
    }

    public var body: some View {
        if full { block } else { line }
    }

    var block: some View {
        let m = Token.Metric.EmptyParts.self
        let buttons = (empty.actions ?? []).compactMap { RowView.action($0, in: actions) }
        return VStack(spacing: 0) {
            if let symbol {
                Image(systemName: symbol).font(.system(size: m.symbol, weight: .light)).foregroundStyle(tokens.color.t3)
            }
            Text(empty.text ?? "").font(Token.Role.emptyBold.font).foregroundStyle(tokens.tone(Token.Role.emptyBold.tone))
                .multilineTextAlignment(.center)
                .padding(.top, m.boldTop)
            if let sub = empty.sub {
                Text(sub).font(Token.Role.emptyGrey.font).foregroundStyle(tokens.tone(Token.Role.emptyGrey.tone))
                    .multilineTextAlignment(.center)
                    .padding(.top, m.greyTop)
            }
            if !buttons.isEmpty {
                HStack(spacing: m.buttonsGap) {
                    ForEach(Array(buttons.enumerated()), id: \.offset) { _, a in
                        PanelButton(a.label ?? "", symbol: a.icon) { run(a) }
                    }
                }
                .padding(.top, m.buttonsTop)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(m.padding)
    }

    var line: some View {
        VStack(alignment: .leading, spacing: Token.Gap.tight) {
            Text(empty.text ?? "no \(empty.noun ?? "")").font(.system(size: Token.Space.text)).foregroundStyle(palette.muted)
            if hints, let hint = empty.hint { HintView(hint) }
        }
    }
}

// invariant: the HUD's capsule: black under the notch, its glyph and its
// words in white; the app's own window holds it.
public struct HUDCapsule: View {
    let words: String
    let symbol: String
    @Environment(\.stationTokens) private var tokens

    public init(_ words: Value, symbol: String) {
        self.words = Format.show(words)
        self.symbol = symbol
    }

    public init(_ notice: HUDNotice) { self.init(notice.words, symbol: notice.symbol) }

    public var body: some View {
        let m = Token.Metric.HUD.self
        HStack(spacing: m.gap) {
            Image(systemName: symbol).font(.system(size: m.icon, weight: .medium))
            Text(words).font(Token.Role.hud.font).lineLimit(1)
        }
        .foregroundStyle(tokens.tone(Token.Role.hud.tone))
        .padding(m.padding)
        .frame(height: m.height)
        .background(UnevenRoundedRectangle(bottomLeadingRadius: m.corner, bottomTrailingRadius: m.corner, style: .continuous).fill(tokens.color.hud))
    }
}
