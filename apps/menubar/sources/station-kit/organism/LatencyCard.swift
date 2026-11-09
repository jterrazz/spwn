import SwiftUI

// invariant: SPEC B5.4's latency card: the site its check asks, its
// latency now on the right, the check's p50 and peak under them, then
// the last hour's runs as one line scaled to its own range, between
// the ends of the hour.
public struct LatencyCard: View {
    let section: Section
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now

    public init(_ section: Section) { self.section = section }

    var chart: PageChart? { (section.items ?? []).compactMap(\.chart).first }

    // value: the check's runs in one line: its name, then its p50 and peak.
    var runs: Value? {
        guard let fact = (section.items ?? []).compactMap(\.fact).first else { return nil }
        var line = Value(kind: "list")
        line.items = [Value(kind: "text", text: fact.key ?? "")] + (fact.value?.items ?? fact.value.map { [$0] } ?? [])
        return line
    }

    public var body: some View {
        let m = Token.Metric.Latency.self
        Card(flush: false) {
            HStack(alignment: .firstTextBaseline) {
                Text(section.title ?? "").font(Token.Role.cardTitle.font).foregroundStyle(tokens.tone(Token.Role.cardTitle.tone))
                Spacer(minLength: 0)
                Text(FactsLine.words(section.caption ?? [], now: now))
                    .font(Token.Role.cardValue.font)
                    .foregroundStyle(tokens.tone(Token.Role.cardValue.tone))
            }
            .lineLimit(1)
            if let runs {
                FactsLine(runs, role: .cardSub)
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
                    .font(Token.Role.cardSub.font)
                    .foregroundStyle(tokens.color.t3)
                    .lineLimit(1)
                    .padding(.top, m.endsTop)
                }
            }
        }
    }
}
