import SwiftUI

// invariant: the row that adds one more to its card (an account shared):
// the plus and its words in the accent, what it promises at its end.
public struct AddRow: View {
    let label: String
    let trail: String?
    let tap: () -> Void
    @Environment(\.stationTokens) private var tokens
    @State private var hovering = false

    public init(_ label: String, trail: String? = nil, tap: @escaping () -> Void = {}) {
        self.label = label
        self.trail = trail
        self.tap = tap
    }

    public var body: some View {
        let m = Token.Metric.Row.self
        Button(action: tap) {
            HStack(spacing: m.gap) {
                Image(systemName: Token.Symbol.plus)
                    .font(.system(size: Token.Metric.Accounts.plus, weight: .medium))
                    .foregroundStyle(tokens.color.accent)
                    .frame(width: m.accessory)
                Text(label).font(Token.Role.addRow.font).foregroundStyle(tokens.color.accent).lineLimit(1)
                Spacer(minLength: 0)
                if let trail {
                    Text(trail).font(Token.Role.addRowEnd.font).foregroundStyle(tokens.tone(Token.Role.addRowEnd.tone)).lineLimit(1)
                }
            }
            .padding(.horizontal, m.padding)
            .frame(minHeight: m.height)
            .background {
                if hovering { RoundedRectangle(cornerRadius: m.radius, style: .continuous).fill(tokens.color.q) }
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .onHover { hovering = $0 }
        .animation(tokens.motion(Token.Motion.tint), value: hovering)
    }
}
