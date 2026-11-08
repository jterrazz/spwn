import SwiftUI

// invariant: SPEC A4 Accounts of All machines: its header (the way back,
// the key, its facts), each section as its accounts card, then its
// footer: the full list in Terminal and the account shared.
public struct AccountsPage: View {
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool

    public init(_ state: PanelState, handlers: PopoverHandlers = PopoverHandlers(), scrolls: Bool = true) {
        self.state = state
        self.handlers = handlers
        self.scrolls = scrolls
    }

    var page: Page { state.page ?? Page() }

    func tap(_ action: Action) { perform(action, handlers) }

    public var body: some View {
        Panel(insets: .page, bounded: scrolls, key: "\(state.openings)/\(page.id ?? "")", sheet: state.confirming) {
            MachineHeader(page, segment: nil, back: handlers.back, run: tap)
        } content: {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                ForEach(Array(MacPage.sections(page, segment: "").enumerated()), id: \.offset) { _, s in
                    AccountsSection(s, actions: page.actions ?? [], busy: state.busy) { a, _ in tap(a) }
                }
                if state.confirming != nil { SheetFor(page: page, state: state, handlers: handlers) }
            }
        } footer: {
            VStack(spacing: Token.Metric.Accounts.footerGap) {
                if let status = state.status { StatusBar(status.line, failed: status.failed) }
                ListFooter(page: page, run: tap)
            }
        }
        .background {
            BackKey(state: state, handlers: handlers)
        }
        .environment(\.stationHints, false)
    }
}

// invariant: a fleet page's footer: its first button as a link that
// leaves for Terminal, the others as the page's buttons.
struct ListFooter: View {
    let page: Page
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens

    var buttons: [Action] { (page.footer?.buttons ?? []).compactMap { page.action($0) } }

    var body: some View {
        if let lead = buttons.first {
            HStack(spacing: Token.Metric.Accounts.footerGap) {
                Button { run(lead) } label: {
                    HStack(spacing: Token.Metric.Accounts.linkGap) {
                        Text(lead.label ?? "")
                        if let icon = lead.icon { Image(systemName: icon).font(Token.Role.rowSub.font) }
                    }
                    .font(Token.Role.footerLink.font)
                    .foregroundStyle(tokens.tone(Token.Role.footerLink.tone))
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                Spacer(minLength: 0)
                ForEach(Array(buttons.dropFirst().enumerated()), id: \.offset) { _, a in
                    PanelButton((a.label ?? "") + (a.form != nil ? Token.Glyph.ellipsis : ""), symbol: a.icon, big: true) { run(a) }
                }
            }
        }
    }
}
