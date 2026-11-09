import SwiftUI

// invariant: SPEC B5.5's Last lines card: its title, then the lines the
// console kept before the server went down, the cause line in t1, the
// full log behind its action (SPEC A9 1-7).
public struct LogCard: View {
    let section: Section
    let actions: [Action]
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ section: Section, actions: [Action] = [], run: @escaping (Action) -> Void = { _ in }) {
        self.section = section
        self.actions = actions
        self.run = run
    }

    var log: Log? { (section.items ?? []).compactMap(\.log).first }

    public var body: some View {
        Card(flush: false) {
            Text(section.title ?? "").font(Token.Role.cardTitle.font).foregroundStyle(tokens.tone(Token.Role.cardTitle.tone))
                .lineLimit(1)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.bottom, Token.Metric.Health.logTitle)
            if let log { LogLines(log, more: RowView.action(log.more, in: actions), run: run) }
        }
    }
}
