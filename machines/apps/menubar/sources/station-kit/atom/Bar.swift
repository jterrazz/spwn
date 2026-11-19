import SwiftUI

public struct Bar: View {
    public enum Kind: Equatable, Sendable {
        case plain(Double)
        case memory(Double)
        case capacity(os: Double, vms: [Double], total: Double)
    }

    let kind: Kind
    @Environment(\.stationTokens) private var tokens

    public init(_ kind: Kind) { self.kind = kind }

    public enum Level: Equatable, Sendable { case calm, warn, critical }

    public static func level(_ used: Double) -> Level {
        used > Token.Metric.Chart.critical ? .critical : used > Token.Metric.Chart.warn ? .warn : .calm
    }

    public var used: Double {
        switch kind {
        case .plain(let share), .memory(let share): min(max(share, 0), 1)
        case .capacity(let os, let vms, let total): total > 0 ? min(max((os + vms.reduce(0, +)) / total, 0), 1) : 0
        }
    }

    var height: CGFloat {
        switch kind {
        case .plain: Token.Metric.Chart.plainBar
        case .memory: Token.Metric.Chart.memoryBar
        case .capacity: Token.Metric.Chart.capacityBar
        }
    }

    func ink(_ calm: Color) -> Color {
        switch Self.level(used) {
        case .calm: calm
        case .warn: tokens.color.orange
        case .critical: tokens.color.red
        }
    }

    var segments: [(share: Double, color: Color)] {
        let c = Token.Metric.Chart.self
        switch kind {
        case .plain(let share): return [(min(max(share, 0), 1), tokens.color.accent)]
        case .memory(let share): return [(min(max(share, 0), 1), ink(tokens.color.accent))]
        case .capacity(let os, let vms, let total):
            guard total > 0 else { return [] }
            let level = Self.level(used)
            let first = [(os / total, tokens.color.t3)]
            return first + vms.enumerated().map { i, vm in
                let calm = tokens.color.accent.opacity(c.vmShares[i % c.vmShares.count])
                return (vm / total, level == .calm ? calm : ink(calm))
            }
        }
    }

    public var body: some View {
        let track: Color = if case .capacity = kind { tokens.color.qq } else { tokens.color.track }
        let round: CGFloat = if case .capacity = kind { 0 } else { height / 2 }
        GeometryReader { geo in
            HStack(spacing: 0) {
                ForEach(Array(segments.enumerated()), id: \.offset) { _, s in
                    RoundedRectangle(cornerRadius: round, style: .continuous).fill(s.color).frame(width: geo.size.width * max(s.share, 0))
                }
                Spacer(minLength: 0)
            }
        }
        .frame(height: height)
        .background(track)
        .clipShape(Capsule())
        .accessibilityHidden(true)
    }
}
