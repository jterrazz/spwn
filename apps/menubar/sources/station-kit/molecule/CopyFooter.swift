import SwiftUI

// invariant: SPEC A4: a page's footer: its command as a mono text button
// that copies it, then its one big button.
public struct CopyFooter: View {
    let footer: Footer
    let actions: [Action]
    let copy: (String) -> Void
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ footer: Footer, actions: [Action] = [], copy: @escaping (String) -> Void = { _ in }, run: @escaping (Action) -> Void = { _ in }) {
        self.footer = footer
        self.actions = actions
        self.copy = copy
        self.run = run
    }

    // invariant: the command is never cut: when it and the button do not
    // fit one line, the command takes its own line above the button.
    public var body: some View {
        ViewThatFits(in: .horizontal) {
            HStack(spacing: Token.Gap.pair) {
                command.fixedSize()
                Spacer(minLength: 0)
                buttons
            }
            VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                command
                HStack { Spacer(minLength: 0); buttons }
            }
        }
    }

    @ViewBuilder
    var command: some View {
        let m = Token.Metric.CopyFooter.self
        HStack(spacing: 0) {
            if let command = footer.copy?.text {
                Button { copy(command) } label: {
                    HStack(spacing: m.gap) {
                        Text(verbatim: command).font(Token.Role.copyCommand.font).lineLimit(1)
                        Image(systemName: Token.Symbol.copy).font(.system(size: m.glyph, weight: .regular))
                    }
                    .foregroundStyle(tokens.tone(Token.Role.copyCommand.tone))
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
            }
        }
    }

    var buttons: some View {
        ForEach(Array((footer.buttons ?? []).compactMap { id in actions.first { $0.id == id } }.enumerated()), id: \.offset) { _, a in
            PanelButton(a.label ?? "", symbol: Token.Symbol.terminal, big: true) { run(a) }.fixedSize()
        }
    }
}
