import SwiftUI

public struct UptimeCells: View {
    let days: [Day]
    @Environment(\.stationTokens) private var tokens

    public init(_ days: [Day]) { self.days = days }

    func fill(_ mark: String?) -> Color {
        let c = Token.Metric.Chart.self
        return switch mark {
        case "ok": tokens.color.t2.opacity(c.uptimeShare)
        case "restart": tokens.color.orange.opacity(c.uptimeMarkShare)
        case "down": tokens.color.red.opacity(c.uptimeMarkShare)
        default: tokens.color.t4
        }
    }

    public var body: some View {
        let c = Token.Metric.Chart.self
        let shown = Array(days.suffix(c.uptimeCells))
        HStack(spacing: c.uptimeGap) {
            ForEach(Array(shown.enumerated()), id: \.offset) { _, day in
                RoundedRectangle(cornerRadius: c.uptimeRadius, style: .continuous)
                    .fill(fill(day.mark))
                    .help(day.tip ?? "")
            }
        }
        .frame(height: c.uptimeHeight)
        .accessibilityElement(children: .ignore)
    }
}
