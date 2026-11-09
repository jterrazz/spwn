import SwiftUI

// invariant: SPEC A4: the reply to an agent: a round field and the
// accent Send beside it; under \.stationStill the field is drawn, as the
// renderer cannot draw a TextField.
public struct ReplyField: View {
    let placeholder: String
    let send: () -> Void
    @Binding var text: String
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still

    public init(_ placeholder: String, text: Binding<String>, send: @escaping () -> Void = {}) {
        self.placeholder = placeholder
        _text = text
        self.send = send
    }

    public var body: some View {
        let m = Token.Metric.Reply.self
        HStack(spacing: m.gap) {
            field
                .padding(.horizontal, m.padding)
                .frame(height: m.height)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(RoundedRectangle(cornerRadius: m.radius, style: .continuous).fill(tokens.color.q))
                .overlay { if tokens.increased { RoundedRectangle(cornerRadius: m.radius, style: .continuous).strokeBorder(tokens.color.hair, lineWidth: Token.Metric.Card.contrastStroke) } }
            Button(action: send) {
                Image(systemName: Token.Symbol.send).font(.system(size: Token.Metric.Control.roundGlyph, weight: .semibold))
                    .foregroundStyle(tokens.onTint)
                    .frame(width: Token.Metric.Control.round, height: Token.Metric.Control.round)
                    .background(Circle().fill(tokens.color.accent))
                    .contentShape(Circle())
            }
            .buttonStyle(.plain)
            .accessibilityLabel(Text(verbatim: "Send"))
        }
    }

    @ViewBuilder
    var field: some View {
        if still {
            Text(text.isEmpty ? placeholder : text)
                .font((text.isEmpty ? Token.Role.replyHint : Token.Role.reply).font)
                .foregroundStyle(tokens.tone(text.isEmpty ? Token.Role.replyHint.tone : Token.Role.reply.tone))
                .lineLimit(1)
        } else {
            TextField(placeholder, text: $text)
                .textFieldStyle(.plain)
                .font(Token.Role.reply.font)
                .onSubmit(send)
        }
    }
}
