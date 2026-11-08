import SwiftUI

// invariant: a card's label: its title, and the caption's words at its
// end in t3, or in their signal's colour when they carry one.
public struct CardLabel: View {
    let title: String
    let value: [Value]
    let about: About?
    let tap: (About) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now

    public init(_ title: String, value: [Value] = [], about: About? = nil, tap: @escaping (About) -> Void = { _ in }) {
        self.title = title
        self.value = value
        self.about = about
        self.tap = tap
    }

    public var body: some View {
        let m = Token.Metric.CardLabel.self
        HStack(alignment: .firstTextBaseline) {
            if let about {
                HStack(alignment: .firstTextBaseline, spacing: m.aboutGap) { name; aboutButton(about) }
            } else {
                name
            }
            Spacer(minLength: 0)
            if !value.isEmpty {
                Text(FactsLine.words(value, now: now))
                    .font(Token.Role.sectionValue.font)
                    .foregroundStyle(value.first?.signal.map(tokens.signalText) ?? tokens.tone(Token.Role.sectionValue.tone))
            }
        }
        .lineLimit(1)
        .frame(height: m.height)
        .padding(.horizontal, m.inset)
    }

    var name: some View {
        Text(title).font(Token.Role.sectionLabel.font).foregroundStyle(tokens.tone(Token.Role.sectionLabel.tone))
    }

    func aboutButton(_ about: About) -> some View {
        let m = Token.Metric.CardLabel.self
        return Button { tap(about) } label: {
            Image(systemName: Token.Symbol.about).font(.system(size: m.aboutGlyph))
                .foregroundStyle(tokens.tone(Token.Role.sectionValue.tone))
                .frame(width: m.about, height: m.about)
                .contentShape(Circle())
        }
        .buttonStyle(.plain)
        .focusEffectDisabled()
        .alignmentGuide(.firstTextBaseline) { $0[VerticalAlignment.center] + m.aboutRise }
        .help(about.label ?? "")
        .accessibilityLabel(about.label ?? "")
    }
}

// invariant: a card: its fill and radius, flush around rows (4) or padded
// around anything else.
public struct Card<Content: View>: View {
    let flush: Bool
    let content: Content

    public init(flush: Bool = true, @ViewBuilder content: () -> Content) {
        self.flush = flush
        self.content = content()
    }

    public var body: some View {
        let m = Token.Metric.Card.self
        VStack(spacing: 0) { content }
            .padding(flush ? EdgeInsets(top: m.flush, leading: m.flush, bottom: m.flush, trailing: m.flush) : m.padding)
            .frame(maxWidth: .infinity, alignment: .leading)
            .modifier(CardFill(lit: false, radius: m.radius))
    }
}
