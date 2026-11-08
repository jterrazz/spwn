import SwiftUI

// invariant: SPEC A4: what an agent does now: its title and task, Full
// log, then its last steps; once finished, its Last steps with Clean up.
public struct NowCard: View {
    let section: Section
    let actions: [Action]
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ section: Section, actions: [Action] = [], run: @escaping (Action) -> Void = { _ in }) {
        self.section = section
        self.actions = actions
        self.run = run
    }

    var steps: [Step] { (section.items ?? []).compactMap(\.steps).flatMap { $0 } }

    // value: the card's own verb: Full log while the agent lives, Clean up
    // once it finished.
    var verb: Action? {
        actions.first { $0.id == "full-log" } ?? actions.first { $0.rows?.contains(section.id ?? "") == true }
    }

    public var body: some View {
        let m = Token.Metric.NowCard.self
        Card(flush: false) {
            VStack(alignment: .leading, spacing: m.gap) {
                HStack(alignment: .firstTextBaseline, spacing: m.headGap) {
                    Text(verbatim: section.title ?? "").font(Token.Role.cardTitle.font).foregroundStyle(tokens.tone(Token.Role.cardTitle.tone))
                    if let task = section.caption?.first?.text {
                        Text(verbatim: Token.Glyph.sep.trimmingCharacters(in: .whitespaces) + " " + task)
                            .font(Token.Role.cardTask.font).foregroundStyle(tokens.tone(Token.Role.cardTask.tone)).lineLimit(1)
                    }
                    Spacer(minLength: 0)
                    if let verb { link(verb) }
                }
                if !steps.isEmpty { StepSlots(steps) }
            }
        }
    }

    @ViewBuilder
    func link(_ a: Action) -> some View {
        if a.id == "full-log" {
            Button { run(a) } label: {
                HStack(spacing: Token.Metric.Log.moreGap) {
                    Text(a.label ?? "")
                    Text(verbatim: Token.Glyph.outward)
                }
                .font(Token.Role.link.font)
                .foregroundStyle(tokens.tone(Token.Role.link.tone))
            }
            .buttonStyle(.plain)
        } else {
            VerbButton(a.label ?? "") { run(a) }
        }
    }
}
