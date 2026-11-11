import SwiftUI

// invariant: SPEC A4 Home's footer: the Find capsule, its glass, its
// words and its ⌘K cap; a click or ⌘K opens Find.
public struct FindCapsule: View {
    let open: () -> Void
    @Environment(\.stationTokens) private var tokens
    @State private var hovering = false

    public init(open: @escaping () -> Void = {}) { self.open = open }

    public var body: some View {
        let m = Token.Metric.Find.self
        let shape = RoundedRectangle(cornerRadius: m.fieldRadius, style: .continuous)
        Button(action: open) {
            HStack(spacing: Token.FindCapsule.gap) {
                Image(systemName: Token.Symbol.search).font(.system(size: Token.FindCapsule.glyph, weight: .medium))
                Text(Token.FindCapsule.words).font(Token.Role.findResult.font).lineLimit(1)
                Spacer(minLength: 0)
                KeyCap(Token.FindCapsule.key)
            }
            .foregroundStyle(tokens.color.t2)
            .padding(.horizontal, Token.FindCapsule.padding)
            .frame(height: m.field)
            .background(shape.fill(hovering ? tokens.color.card2 : tokens.color.card))
            .overlay { shape.strokeBorder(tokens.cardEdge, lineWidth: Token.Metric.Card.edge) }
            .contentShape(shape)
        }
        .buttonStyle(.plain)
        .keyboardShortcut("k", modifiers: .command)
        .onHover { hovering = $0 }
        .panelFocus(.find, look: .control, radius: m.fieldRadius)
        .animation(tokens.motion(Token.Motion.tint), value: hovering)
        .accessibilityLabel(Text(Token.FindCapsule.words))
    }
}
