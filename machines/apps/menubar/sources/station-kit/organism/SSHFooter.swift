import SwiftUI

// invariant: SPEC A4's footer of a VM's page: its ssh in mono with its
// copy, then Open in Terminal, the big neutral button (SPEC A9 1-10);
// while the VM cannot answer, why in place of the ssh, the button
// waiting.
public struct SSHFooter: View {
    let footer: Footer
    let actions: [Action]
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ footer: Footer, actions: [Action] = [], run: @escaping (Action) -> Void = { _ in }) {
        self.footer = footer
        self.actions = actions
        self.run = run
    }

    var buttons: [Action] { (footer.buttons ?? []).compactMap { RowView.action($0, in: actions) } }
    var copy: Action? { buttons.first { $0.run?.copy != nil } }
    var terminal: Action? { buttons.first { $0.run?.terminal == true } }

    public var body: some View {
        let m = Token.Metric.SSHFooter.self
        HStack(spacing: Token.Metric.Panel.blocks) {
            if let words = footer.copy {
                Button { if let copy { run(copy) } } label: {
                    HStack(spacing: m.copyGap) {
                        Text(verbatim: words.text ?? "").font(Token.Role.footerCopy.font).foregroundStyle(tokens.tone(Token.Role.footerCopy.tone))
                        Image(systemName: Token.Symbol.copy).font(.system(size: m.copyGlyph)).foregroundStyle(tokens.color.t3)
                    }
                    .lineLimit(1)
                }
                .buttonStyle(.plain)
                .accessibilityLabel(copy?.label ?? "")
            } else if let note = footer.note {
                Text(note).font(Token.Role.footerNote.font).foregroundStyle(tokens.tone(Token.Role.footerNote.tone)).lineLimit(1)
            }
            Spacer(minLength: 0)
            if let terminal {
                let off = terminal.disabled != nil
                PanelButton(terminal.label ?? "", symbol: terminal.icon, big: true) { run(terminal) }
                    .disabled(off)
                    .opacity(off ? m.disabledShare : 1)
                    .help(terminal.disabled ?? "")
            }
        }
    }
}
