import SwiftUI

// invariant: a card's label: its title, and the caption's words at its
// end in t3, or in their signal's colour when they carry one.
public struct CardLabel: View {
    let title: String
    let value: [Value]
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now

    public init(_ title: String, value: [Value] = []) {
        self.title = title
        self.value = value
    }

    public var body: some View {
        let m = Token.Metric.CardLabel.self
        HStack(alignment: .firstTextBaseline) {
            Text(title).font(Token.Role.sectionLabel.font).foregroundStyle(tokens.tone(Token.Role.sectionLabel.tone))
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
