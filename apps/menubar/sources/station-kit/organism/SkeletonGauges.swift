import SwiftUI

// invariant: SPEC A4, M17: a machine's gauges while none is read yet
// (a server restarting): each its title over a figure and a bar drawn
// as placeholders, the shimmer crossing them every 1.2 s; still, the
// band rests mid-tile.
public struct SkeletonGauges: View {
    let section: Section
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still

    public init(_ section: Section) { self.section = section }

    public static let stillProgress = Token.Motion.skeletonStill

    public var body: some View {
        if still {
            strip(Self.stillProgress)
        } else {
            TimelineView(.animation) { context in
                let period = Token.Motion.skeletonPeriod
                strip(context.date.timeIntervalSinceReferenceDate.truncatingRemainder(dividingBy: period) / period)
            }
        }
    }

    func strip(_ progress: Double) -> some View {
        let m = Token.Metric.Gauges.self
        return HStack(spacing: m.gap) {
            ForEach(Array((section.items ?? []).compactMap(\.gauge).enumerated()), id: \.offset) { _, gauge in
                tile(gauge, progress: progress)
            }
        }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel((section.items ?? []).compactMap(\.gauge?.title).joined(separator: ", "))
    }

    func tile(_ gauge: PageGauge, progress: Double) -> some View {
        let m = Token.Metric.Gauges.self
        let g = Token.Metric.GaugeParts.self
        return VStack(alignment: .leading, spacing: 0) {
            Text(gauge.title ?? "")
                .font(Token.Role.gaugeTitle.font)
                .foregroundStyle(tokens.tone(Token.Role.gaugeTitle.tone))
                .lineLimit(1)
            Text(verbatim: "00.0 GB").font(Token.Role.gaugeValue.font)
                .modifier(Skeleton(progress: progress))
            Spacer(minLength: 0)
            Capsule().fill(tokens.color.track).frame(height: g.bar)
                .modifier(Skeleton(progress: progress))
                .frame(height: m.visual)
        }
        .padding(m.padding)
        .frame(maxWidth: .infinity, alignment: .leading)
        .frame(height: m.height)
        .modifier(CardFill(lit: false, radius: Token.Metric.Card.radius))
    }
}
