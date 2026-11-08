import SwiftUI

// invariant: SPEC B5.4's service card: the host its check asks, its
// latency now on the right, the check's p50 and peak under them, then
// the last hour's runs as one line scaled to its own range, between
// the ends of the hour.
public struct ServiceCard: View {
    let section: Section
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now

    public init(_ section: Section) { self.section = section }

    var chart: PageChart? { (section.items ?? []).compactMap(\.chart).first }

    public var body: some View {
        let m = Token.Metric.Service.self
        Card(flush: false) {
            HStack(alignment: .firstTextBaseline) {
                Text(section.title ?? "").font(Token.Role.serviceTitle.font).foregroundStyle(tokens.tone(Token.Role.serviceTitle.tone))
                Spacer(minLength: 0)
                Text(FactsLine.words(section.caption ?? [], now: now))
                    .font(Token.Role.serviceValue.font)
                    .foregroundStyle(tokens.tone(Token.Role.serviceValue.tone))
            }
            .lineLimit(1)
            ForEach(Array(section.notes.enumerated()), id: \.offset) { _, note in
                Text(note).font(Token.Role.serviceSub.font).foregroundStyle(tokens.tone(Token.Role.serviceSub.tone))
                    .lineLimit(1)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.top, m.subTop)
            }
            if let chart {
                Spark(chart.series?.first?.points ?? [], height: m.chart).padding(.top, m.chartTop)
                if let ends = chart.ends, ends.count == 2 {
                    HStack {
                        Text(ends[0])
                        Spacer(minLength: 0)
                        Text(ends[1])
                    }
                    .font(Token.Role.serviceSub.font)
                    .foregroundStyle(tokens.color.t3)
                    .lineLimit(1)
                    .padding(.top, m.endsTop)
                }
            }
        }
    }
}
