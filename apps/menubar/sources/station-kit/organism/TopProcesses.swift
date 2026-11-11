import SwiftUI

public struct TopProcesses: View {
    let section: Section
    let open: (String) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationSamples) private var samples

    public init(_ section: Section, open: @escaping (String) -> Void = { _ in }) {
        self.section = section
        self.open = open
    }

    public static let column = "cpu"

    func share(_ row: Row) -> Value? { section.cell(row, row.trail ?? Self.column) }

    public static func ranked(_ section: Section, shown said: (Value) -> Value) -> [Row] {
        let shown = section.rows.enumerated().map { i, row in
            (at: i, row: row, share: section.cell(row, row.trail ?? column).map { said($0).n ?? 0 } ?? 0)
        }
        return shown.sorted { $0.share != $1.share ? $0.share > $1.share : $0.at < $1.at }.map(\.row)
    }

    public var body: some View {
        LiveRead { store in drawn(Self.ranked(section) { Live.value(store.now($0) ?? $0, samples: samples) }) }
    }

    func drawn(_ rows: [Row]) -> some View {
        let m = Token.Metric.Row.self
        return VStack(spacing: 0) {
            ForEach(Array(rows.enumerated()), id: \.offset) { i, row in
                if i > 0 {
                    Rectangle().fill(tokens.color.hair).frame(height: m.hairline)
                        .padding(.leading, m.insetGlyph).padding(.trailing, m.insetEnd)
                }
                line(row)
            }
        }
        .padding(.vertical, Token.Metric.Card.flush)
        .modifier(CardFill(lit: false, radius: Token.Metric.Card.radius))
    }

    func line(_ row: Row) -> some View {
        let m = Token.Metric.Row.self
        return HStack(spacing: m.gap) {
            SwiftUI.Group {
                if let glyph = row.glyph { GlyphView(glyph) } else { Color.clear }
            }
            .frame(width: m.accessory, height: m.accessory)
            HStack(spacing: 0) {
                Text(row.label ?? "").font(Token.Role.rowName.font).foregroundStyle(tokens.tone(Token.Role.rowName.tone))
                if let detail = row.detail, !Format.show(detail).isEmpty {
                    Text(Token.Glyph.sep + Format.show(detail)).font(Token.Role.rowSuffix.font).foregroundStyle(tokens.tone(Token.Role.rowSuffix.tone))
                }
            }
            .lineLimit(1)
            Spacer(minLength: m.gap)
            if let share = share(row) { LiveText(share) }
            Image(systemName: Token.Symbol.forward)
                .font(.system(size: Token.Metric.Chart.ends, weight: .semibold))
                .foregroundStyle(tokens.color.t3)
                .opacity(row.nav == nil ? 0 : 1)
        }
        .padding(.horizontal, m.padding)
        .frame(minHeight: m.height)
        .contentShape(Rectangle())
        .onTapGesture { if let nav = row.nav { open(nav) } }
        .accessibilityElement(children: .combine)
        .accessibilityAddTraits(row.nav == nil ? [] : .isButton)
    }
}
