import SwiftUI

public struct PageView: View {
    let page: Page
    let width: CGFloat
    @Environment(\.stationPalette) private var palette

    public init(_ page: Page, width: CGFloat = Token.Space.width) {
        self.page = page
        self.width = width
    }

    var inner: CGFloat { width - Token.Space.margin * 2 }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Space.gap) {
            HStack {
                Text("station \(page.command ?? page.id ?? "")").font(.system(size: Token.Space.text, weight: .bold)).foregroundStyle(palette.text)
                Spacer()
                Text([page.context?.machine, page.context?.kind].compactMap { $0 }.joined(separator: Token.Glyph.sep))
                    .font(.system(size: Token.Space.small)).foregroundStyle(palette.muted)
            }
            if let strip = page.strip { StripView(strip) }
            Divider()
            if let error = page.error { ErrorLine(error) }
            ForEach(Array((page.sections ?? []).filter { $0.role != "lifted" }.enumerated()), id: \.offset) { _, s in
                SectionView(s, width: inner, hintButtons: true)
            }
            ForEach(Array((page.hints ?? []).enumerated()), id: \.offset) { _, h in HintView(h) }
        }
        .padding(Token.Space.margin)
        .frame(width: width, alignment: .topLeading)
        .stationPageGlass()
    }
}
