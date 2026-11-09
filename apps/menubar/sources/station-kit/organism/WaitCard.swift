import SwiftUI

// invariant: SPEC A4: what an agent waits on you for: its own line, its
// detail, its PR and change, Open PR and Merge (the tint on the next
// step: Merge once the PR was opened), then the reply.
public struct WaitCard: View {
    let section: Section
    let actions: [Action]
    let opened: Bool
    let run: (Action) -> Void
    let send: () -> Void
    @Binding var reply: String
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now

    public init(_ section: Section, actions: [Action] = [], opened: Bool = false, reply: Binding<String>, run: @escaping (Action) -> Void = { _ in }, send: @escaping () -> Void = {}) {
        self.section = section
        self.actions = actions
        self.opened = opened
        _reply = reply
        self.run = run
        self.send = send
    }

    var message: Message? { section.items?.compactMap(\.message).first }

    func action(_ id: String) -> Action? { actions.first { $0.id == id } }

    // value: the step the tint is on: Open PR until the PR was opened, then
    // Merge, the next step.
    public static func tinted(opened: Bool) -> String { opened ? "merge" : "open-pr" }

    public var body: some View {
        let m = Token.Metric.Wait.self
        Card(flush: false) {
            VStack(alignment: .leading, spacing: m.gap) {
                head
                if let message {
                    Text(verbatim: message.title ?? "").font(Token.Role.waitTitle.font).foregroundStyle(tokens.tone(Token.Role.waitTitle.tone))
                        .fixedSize(horizontal: false, vertical: true)
                    if let body = message.body, !body.isEmpty {
                        Text(verbatim: body).font(Token.Role.waitBody.font).foregroundStyle(tokens.tone(Token.Role.waitBody.tone))
                            .lineSpacing(m.bodySpacing)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                    if let pr = message.pr { prLine(pr) }
                    buttons.padding(.top, m.buttonsTop)
                    if message.reply == true, let a = action("reply") {
                        ReplyField(a.label.map { $0 + Token.Glyph.ellipsis } ?? "", text: $reply, send: send)
                    }
                }
            }
        }
    }

    var head: some View {
        HStack(spacing: Token.Metric.Wait.headGap) {
            Image(systemName: Token.Symbol.waits).font(Token.Role.cardTitle.font)
            Text(verbatim: section.title ?? "").font(Token.Role.cardTitle.font)
            Spacer(minLength: 0)
            if let since = message?.since {
                Text(Format.show(since, now: now)).font(Token.Role.waitSince.font).foregroundStyle(tokens.tone(Token.Role.waitSince.tone))
            }
        }
        .foregroundStyle(tokens.color.orangeText)
    }

    func prLine(_ pr: PR) -> some View {
        let sep = Text(verbatim: Token.Glyph.sep)
        var line = Text(verbatim: "PR #\(pr.number ?? 0)")
        if let files = pr.files { line = Text("\(line)\(sep)\(Text(verbatim: "\(files) files"))") }
        if let plus = pr.plus, let minus = pr.minus {
            line = Text("\(line)\(sep)\(Text(verbatim: "+\(plus)").foregroundStyle(tokens.color.greenText)) \(Text(verbatim: "\(Token.Glyph.minus)\(minus)").foregroundStyle(tokens.color.redText))")
        }
        return HStack(spacing: Token.Metric.Wait.prGap) {
            Image(systemName: Token.Symbol.pullRequest)
            line
        }
        .font(Token.Role.waitPR.font)
        .foregroundStyle(tokens.tone(Token.Role.waitPR.tone))
        .lineLimit(1)
    }

    var buttons: some View {
        HStack(spacing: Token.Metric.Wait.buttonsGap) {
            if let open = action("open-pr") {
                PanelButton((open.label ?? "") + " " + Token.Glyph.outward, look: Self.tinted(opened: opened) == open.id ? .tint : .plain) { run(open) }
            }
            if let merge = action("merge") {
                PanelButton(merge.label ?? "", look: Self.tinted(opened: opened) == merge.id ? .tint : .plain) { run(merge) }
            }
        }
        .animation(tokens.motion(Token.Motion.tint), value: opened)
    }
}
