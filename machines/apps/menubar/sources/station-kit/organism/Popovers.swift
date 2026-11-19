import SwiftUI

public struct QRPopover: View {
    let url: URL
    let row: Row
    let actions: [Action]
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationPalette) private var palette

    static let scan = "Scan with your phone"
    static let tailnet = "Opens on any device of your tailnet"
    static let anyone = "Anyone can open it"
    static let copy = "Copy"
    static let open = "Open"

    public init(url: URL, row: Row, actions: [Action], run: @escaping (Action) -> Void) {
        self.url = url
        self.row = row
        self.actions = actions
        self.run = run
    }

    public static func verbs(of row: Row, in actions: [Action]) -> (copy: Action?, open: Action?, stop: Action?) {
        let mine = actions.filter { $0.rows?.contains(row.id ?? "") == true }
        return (mine.first { $0.run?.copy != nil }, mine.first { $0.run?.open != nil }, mine.first { $0.id?.hasPrefix(stopSharing) == true })
    }

    public static let stopSharing = "stop-sharing:"

    public static func reach(_ row: Row) -> (link: String, line: String, public: Bool) {
        let (address, route) = PortRow.address(row.detail ?? Value())
        guard let route else { return (address, tailnet, false) }
        return (address, anyone + Token.Glyph.sep + route, true)
    }

    public var body: some View {
        let m = Token.Metric.Popover.self
        let verbs = Self.verbs(of: row, in: actions)
        let said = Self.reach(row)
        HStack(alignment: .top, spacing: m.column) {
            QRView(url.absoluteString, size: m.qr)
                .padding(m.qrInset)
                .background(RoundedRectangle(cornerRadius: m.qrRadius, style: .continuous).fill(palette.qrLight))
            VStack(alignment: .leading, spacing: m.lines) {
                Text(Self.scan).font(Token.Role.popoverTitle.font).foregroundStyle(tokens.tone(Token.Role.popoverTitle.tone))
                Text(verbatim: said.link)
                    .font((said.public ? Token.Role.popoverDomain : Token.Role.popoverLink).font)
                    .foregroundStyle(tokens.tone(Token.Role.popoverLink.tone))
                    .lineLimit(1)
                Text(said.line).font(Token.Role.popoverReach.font).foregroundStyle(tokens.tone(Token.Role.popoverReach.tone))
                    .fixedSize(horizontal: false, vertical: true)
                HStack(spacing: m.buttons) {
                    if let copy = verbs.copy { PanelButton(Self.copy, symbol: Token.Symbol.copy) { run(copy) } }
                    if let open = verbs.open { PanelButton(Self.open + " " + Token.Glyph.outward) { run(open) } }
                }
                .padding(.top, m.buttons)
                if let stop = verbs.stop, !said.public {
                    Button { run(stop) } label: {
                        Text((stop.label ?? "") + Token.Glyph.ellipsis).font(Token.Role.popoverDomain.font).foregroundStyle(tokens.color.redText)
                    }
                    .buttonStyle(.plain)
                    .padding(.top, m.stopTop)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(m.paddingQR)
        .frame(width: m.widthQR)
    }
}

public struct InfoPopover: View {
    let text: String
    @Environment(\.stationTokens) private var tokens

    public init(_ text: String) { self.text = text }

    public var body: some View {
        let m = Token.Metric.Popover.self
        Text(text)
            .font(Token.Role.popoverBody.font)
            .foregroundStyle(tokens.tone(Token.Role.popoverBody.tone))
            .lineSpacing(m.lines)
            .fixedSize(horizontal: false, vertical: true)
            .padding(m.padding)
            .frame(width: m.width, alignment: .leading)
    }
}

public struct TooltipPreview: View {
    let words: String
    @Environment(\.stationTokens) private var tokens

    public init(_ words: String) { self.words = words }

    public var body: some View {
        let m = Token.Metric.Tooltip.self
        let shape = RoundedRectangle(cornerRadius: m.radius, style: .continuous)
        Text(words)
            .font(Token.Role.tooltip.font)
            .foregroundStyle(tokens.tone(Token.Role.tooltip.tone))
            .fixedSize(horizontal: false, vertical: true)
            .padding(m.padding)
            .background(shape.fill(tokens.color.panelSolid))
            .overlay(shape.strokeBorder(tokens.color.hair, lineWidth: Token.Metric.Row.hairline))
            .frame(maxWidth: m.maxWidth, alignment: .leading)
            .fixedSize()
    }
}
