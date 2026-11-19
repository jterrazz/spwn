import SwiftUI

// invariant: SPEC A4 Awake: one line of a Mac's Awake card: its sun or
// moon in the accessory column, its policy and what depends on it, its
// switch; an asleep Mac's line is dimmed and has none.
public struct AwakeMac: View {
    let row: Row
    let on: Bool
    let waiting: Bool
    let toggle: (() -> Void)?
    let label: String
    @Environment(\.stationTokens) private var tokens

    public init(_ row: Row, on: Bool, waiting: Bool = false, label: String = "", toggle: (() -> Void)? = nil) {
        self.row = row
        self.on = on
        self.waiting = waiting
        self.label = label
        self.toggle = toggle
    }

    var dim: Bool { row.signal == "asleep" }

    // value: a line's sun or moon, inked as its name is, the board's
    // monochrome lead; any other glyph draws as its signal says.
    @ViewBuilder
    static func lead(_ glyph: Glyph, ink: Color) -> some View {
        if case let .symbol(name) = GlyphView.drawn(glyph), glyph.signal == "quiet" {
            Image(systemName: name).font(.system(size: Token.Metric.Marks.glyph, weight: .medium))
                .foregroundStyle(ink)
                .frame(width: Token.Metric.Marks.glyphSlot, height: Token.Metric.Marks.glyphSlot)
        } else {
            GlyphView(glyph)
        }
    }
    var sub: Value? { row.detail.flatMap { $0.isNone ? nil : $0 } }

    public var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.listitem, label, kit: "AwakeMac", states: ViewState.of([.on: on, .waiting: waiting, .dimmed: dim])) }

    @ViewBuilder var drawing: some View {
        let m = Token.Metric.Row.self
        HStack(spacing: m.gap) {
            SwiftUI.Group {
                if let glyph = row.glyph { Self.lead(glyph, ink: tokens.tone(Token.Role.rowName.tone)) } else { Color.clear }
            }
            .frame(width: m.accessory)
            VStack(alignment: .leading, spacing: m.subGap) {
                Text(row.label ?? "").font(Token.Role.rowName.font).foregroundStyle(tokens.tone(Token.Role.rowName.tone)).lineLimit(1)
                if let sub { FactsLine(sub, role: .rowSub) }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            if waiting {
                PendingDot()
            } else if let toggle {
                Switch(on: on, tap: toggle)
                    .padding(.trailing, Token.Metric.Care.switchTrail)
                    .accessibilityLabel(label.isEmpty ? row.label ?? "" : label)
            }
        }
        .opacity(dim ? m.dimShare : 1)
        .padding(.horizontal, m.padding)
        .frame(minHeight: sub == nil ? m.height : m.twoLines)
    }
}
