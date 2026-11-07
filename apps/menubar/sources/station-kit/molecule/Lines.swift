import SwiftUI

// invariant: an agent's last steps: when, in one column, what it did, in
// one text x; the step running in t1, its diff after it.
public struct StepSlots: View {
    let steps: [Step]
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now

    public init(_ steps: [Step]) { self.steps = steps }

    func words(_ step: Step) -> String {
        [step.text, step.diff].compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
    }

    public var body: some View {
        let m = Token.Metric.Steps.self
        Grid(alignment: .leadingFirstTextBaseline, horizontalSpacing: m.columnGap, verticalSpacing: m.rowGap) {
            ForEach(Array(steps.enumerated()), id: \.offset) { _, step in
                let role: Token.Role = step.current == true ? .stepCurrent : .step
                GridRow {
                    Text(step.at.map { Format.show($0, now: now) } ?? "")
                        .font(Token.Role.stepAt.font)
                        .foregroundStyle(tokens.tone(Token.Role.stepAt.tone))
                        .lineLimit(1)
                        .fixedSize()
                    Text(words(step))
                        .font(role.font)
                        .foregroundStyle(tokens.tone(role.tone))
                        .lineLimit(1)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
        }
    }
}

// invariant: the last lines of a run, in mono: each line's time, its
// source in t1, what it said; the cause line (Log.Cause, counted from 1)
// wholly in t1; the full log behind its action.
public struct LogLines: View {
    let log: Log
    let more: Action?
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ log: Log, more: Action? = nil, run: @escaping (Action) -> Void = { _ in }) {
        self.log = log
        self.more = more
        self.run = run
    }

    // value: a line's time, its source and the rest, split at its first
    // two spaces; a line without them is all said.
    public static func parts(_ line: String) -> (at: String, source: String, said: String) {
        let words = line.split(separator: " ", maxSplits: 2, omittingEmptySubsequences: false).map(String.init)
        guard words.count == 3 else { return ("", "", line) }
        return (words[0], words[1], words[2])
    }

    func line(_ text: String, cause: Bool) -> Text {
        let p = Self.parts(text)
        let said = cause ? tokens.color.t1 : tokens.tone(Token.Role.logLine.tone)
        guard !p.at.isEmpty else { return Text(verbatim: p.said).foregroundStyle(said) }
        let at = Text(verbatim: p.at + " ").foregroundStyle(cause ? tokens.color.t1 : tokens.color.t3)
        let source = Text(verbatim: p.source + " ").font(Token.Role.logSource.font).foregroundStyle(tokens.tone(Token.Role.logSource.tone))
        return Text("\(at)\(source)\(Text(verbatim: p.said).foregroundStyle(said))")
    }

    public var body: some View {
        let m = Token.Metric.Log.self
        VStack(alignment: .leading, spacing: m.lineGap) {
            ForEach(Array((log.lines ?? []).enumerated()), id: \.offset) { i, text in
                line(text, cause: log.cause == i + 1)
                    .font(Token.Role.logLine.font)
                    .tracking(Token.Role.logLine.tracking)
                    .lineLimit(1)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
            if let more {
                Button { run(more) } label: {
                    HStack(spacing: m.moreGap) {
                        Text(more.label ?? "")
                        Text(verbatim: Token.Glyph.outward)
                    }
                    .font(Token.Role.link.font)
                    .foregroundStyle(tokens.tone(Token.Role.link.tone))
                }
                .buttonStyle(.plain)
                .padding(.top, m.moreTop)
            }
        }
    }
}
