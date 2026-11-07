import SwiftUI

public struct TokenSheet: View {
    @Environment(\.stationTokens) private var tokens

    public init() {}

    static func swatches(_ c: Token.Colors) -> [(String, Color)] {
        [
            ("panel", c.panel), ("panelSolid", c.panelSolid), ("panelEdge", c.panelEdge), ("sheetSolid", c.sheetSolid),
            ("menu", c.menu), ("card", c.card), ("hero", c.hero), ("card2", c.card2), ("hair", c.hair),
            ("t1", c.t1), ("t2", c.t2), ("t3", c.t3), ("t4", c.t4), ("q", c.q), ("qq", c.qq), ("key", c.key), ("track", c.track),
            ("accent", c.accent), ("accentText", c.accentText), ("green", c.green), ("greenText", c.greenText),
            ("orange", c.orange), ("orangeText", c.orangeText), ("red", c.red), ("redText", c.redText), ("indigo", c.indigo),
            ("scrim", c.scrim), ("scrimFind", c.scrimFind), ("ring", c.ring), ("hud", c.hud),
        ]
    }

    public var body: some View {
        let c = tokens.color
        let inner = Token.Metric.Panel.width - Token.Metric.Panel.body.leading - Token.Metric.Panel.body.trailing
        let columns = Array(repeating: GridItem(.fixed((inner - Token.Metric.Tiles.gap) / 2), spacing: Token.Metric.Tiles.gap, alignment: .leading), count: 2)
        VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
            LazyVGrid(columns: columns, alignment: .leading, spacing: Token.Metric.Tiles.gap) {
                ForEach(Array(Self.swatches(c).enumerated()), id: \.offset) { _, swatch in
                    HStack(spacing: Token.Metric.Row.gap) {
                        RoundedRectangle(cornerRadius: Token.Metric.Marks.logoRadius, style: .continuous)
                            .fill(swatch.1)
                            .overlay(RoundedRectangle(cornerRadius: Token.Metric.Marks.logoRadius, style: .continuous).strokeBorder(c.hair, lineWidth: Token.Metric.Card.contrastStroke))
                            .frame(width: Token.Metric.Marks.logoChip, height: Token.Metric.Marks.logoChip)
                        Text(swatch.0).font(Token.Role.address.font).foregroundStyle(c.t2)
                    }
                }
            }
            LazyVGrid(columns: Array(repeating: GridItem(.fixed(Token.Metric.Marks.logoChip), spacing: Token.Metric.Tiles.gap), count: Token.Metric.Chart.uptimeCells / Token.Metric.Tiles.columns), alignment: .leading, spacing: Token.Metric.Tiles.gap) {
                ForEach(Token.Brand.allCases, id: \.self) { brand in
                    let chip = brand.chip(tokens.scheme)
                    RoundedRectangle(cornerRadius: Token.Metric.Marks.logoRadius, style: .continuous)
                        .fill(LinearGradient(colors: chip.fill, startPoint: .topLeading, endPoint: .bottomTrailing))
                        .overlay(Circle().fill(chip.mark).frame(width: Token.Metric.Marks.dot, height: Token.Metric.Marks.dot))
                        .frame(width: Token.Metric.Marks.logoChip, height: Token.Metric.Marks.logoChip)
                }
            }
            VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
                ForEach(Array(Token.Role.all.enumerated()), id: \.offset) { _, role in
                    Text(role.0).font(role.1.font).tracking(role.1.tracking).foregroundStyle(tokens.tone(role.1.tone))
                        .background(role.1.tone == .hud ? c.hud : Color.clear)
                }
            }
            .padding(Token.Metric.Card.padding)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(RoundedRectangle(cornerRadius: Token.Metric.Card.radius, style: .continuous).fill(c.card))
        }
        .padding(Token.Metric.Panel.body)
        .frame(width: Token.Metric.Panel.width, alignment: .leading)
        .background(tokens.color.panelSolid)
    }
}
