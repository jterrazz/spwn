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

    public var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.button, label, kit: "AddRow") }

    @ViewBuilder var drawing: some View {
        let m = Token.Metric.Row.self
        Button(action: tap) {
            HStack(spacing: m.gap) {
                Image(systemName: Token.Symbol.plus)
                    .font(.system(size: Token.Metric.Accounts.plus, weight: .medium))
                    .foregroundStyle(tokens.color.accent)
                    .frame(width: m.accessory)
                Text(label).recorded(label, Token.Role.addRow).foregroundStyle(tokens.color.accent).lineLimit(1)
                Spacer(minLength: 0)
                if let trail {
                    Text(trail).recorded(trail, Token.Role.addRowEnd).foregroundStyle(tokens.tone(Token.Role.addRowEnd.tone)).lineLimit(1)
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
