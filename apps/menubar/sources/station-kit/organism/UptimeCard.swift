import SwiftUI

// invariant: SPEC B5.2's Last 30 days: its title and how available it
// was, a cell a day (today last, each with its tooltip), then the ends of
// the span in the page's words.
public struct UptimeCard: View {
    let section: Section
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now

    public init(_ section: Section) { self.section = section }

    var uptime: Uptime? { (section.items ?? []).compactMap(\.uptime).first }

    public var body: some View {
        let m = Token.Metric.Uptime.self
        let ends = section.notes
        Card(flush: false) {
            HStack(alignment: .firstTextBaseline) {
                Text(section.title ?? "").font(Token.Role.serviceTitle.font).foregroundStyle(tokens.tone(Token.Role.serviceTitle.tone))
                Spacer(minLength: 0)
                Text(FactsLine.words(section.caption ?? [], now: now))
                    .font(Token.Role.serviceValue.font)
                    .foregroundStyle(tokens.tone(Token.Role.serviceValue.tone))
            }
            .lineLimit(1)
            UptimeCells(uptime?.days ?? []).padding(.top, m.cellsTop)
            if ends.count == 2 {
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
