import SwiftUI

// invariant: SPEC A4 An agent: its header, then what it waits on you for
// or what it does, then the workspace it runs on; its terminal is the
// footer while it lives. Whether its PR was opened and the reply typed are
// the client's own.
struct AgentPage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    @State private var reply: String
    @State private var opened: Bool
    @Environment(\.stationStill) private var still

    init(page: Page, state: PanelState, handlers: PopoverHandlers, scrolls: Bool) {
        self.page = page
        self.state = state
        self.handlers = handlers
        self.scrolls = scrolls
        _reply = State(initialValue: state.reply)
        _opened = State(initialValue: state.prOpened.contains(page.context?.machine ?? ""))
    }

    var body: some View {
        Panel(insets: .page, bounded: scrolls, key: "\(state.openings)/agent/\(page.context?.machine ?? "")", sheet: state.confirming, layers: AlertLayer.layers(page, state, handlers)) {
            MachineHeader(page, segment: nil, back: handlers.back, run: tap, open: handlers.push)
        } content: {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                ForEach(Array(MacPage.sections(page, segment: "").enumerated()), id: \.offset) { _, s in
                    section(s)
                }
                if state.confirming != nil { SheetFor(page: page, state: state, handlers: handlers) }
            }
        } footer: {
            PageFooter(page: page, state: state, handlers: handlers)
        }
        .background {
            BackKey(state: state, handlers: handlers)
            if !still, BackKey.gesture(state) == .back { EscapeKey(back: handlers.back) }
        }
        .environment(\.stationHints, false)
    }

    func tap(_ action: Action) {
        if action.id == "open-pr" { opened = true }
        perform(action, handlers)
    }

    func send() {
        guard let a = page.action("reply"), !reply.trimmingCharacters(in: .whitespaces).isEmpty else { return }
        handlers.run(a, ["Text": reply])
        reply = ""
    }

    @ViewBuilder
    func section(_ s: Section) -> some View {
        switch s.card {
        case "wait": WaitCard(s, actions: page.actions ?? [], opened: opened, reply: $reply, run: tap, send: send)
        case "now": NowCard(s, actions: page.actions ?? [], run: tap)
        default:
            VMsCard(s, actions: page.actions ?? [], open: handlers.push) { action, row in
                if action.run == nil, let nav = row?.nav { handlers.push(nav) } else { tap(action) }
            }
        }
    }
}
