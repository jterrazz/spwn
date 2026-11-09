import SwiftUI

// invariant: SPEC A4, B1: a workspace's page: its header, then the
// sections of the segment picked, its agents first on its Overview; the
// segment is the client's own, so a pick asks the engine nothing.
struct WorkspacePage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    @State private var segment: String
    @State private var window: String
    @State private var unfolded: Set<String>
    @Environment(\.stationStill) private var still

    init(page: Page, state: PanelState, handlers: PopoverHandlers, scrolls: Bool) {
        self.page = page
        self.state = state
        self.handlers = handlers
        self.scrolls = scrolls
        _segment = State(initialValue: state.segment ?? MacPage.picked(page))
        _window = State(initialValue: ActivityWindow.remembered(state, page))
        _unfolded = State(initialValue: Set(state.unfolded))
    }

    var body: some View {
        Panel(insets: .page, bounded: scrolls, key: "\(state.openings)/\(page.context?.machine ?? "")/\(segment)", sheet: state.confirming) {
            MachineHeader(page, segment: segment, pick: { segment = $0 }, back: handlers.back, run: tap, open: handlers.push)
        } content: {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                if segment == ActivityWindow.segment {
                    MachineActivity(page: page, sections: MacPage.sections(page, segment: segment), window: window, pick: pickWindow,
                                    scrub: state.scrub, onScrub: handlers.scrub, run: run, open: handlers.push)
                } else {
                    ForEach(Array(MacPage.sections(page, segment: segment).enumerated()), id: \.offset) { _, s in
                        section(s)
                    }
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

    func tap(_ action: Action) { perform(action, handlers) }

    func run(_ action: Action, _ row: Row?) {
        if action.run == nil, let nav = row?.nav {
            handlers.push(nav)
        } else {
            tap(action)
        }
    }

    func pickWindow(_ flag: String) {
        window = flag
        handlers.remember(page.context?.machine ?? "", flag)
    }

    func fold(_ id: String) {
        if unfolded.contains(id) { unfolded.remove(id) } else { unfolded.insert(id) }
    }

    @ViewBuilder
    func section(_ s: Section) -> some View {
        switch (s.segment, s.id) {
        case ("ports", _), (_, "ports"): MacPorts(section: s, page: page, state: state, handlers: handlers, run: tap)
        case ("accounts", _): MacAccounts(section: s, page: page, busy: state.busy, run: run)
        case (_, "agents"): WorkspaceAgents(section: s, actions: page.actions ?? [], run: run)
        case (_, "stopped"):
            ForEach(Array((s.items ?? []).compactMap(\.empty).enumerated()), id: \.offset) { _, empty in
                Card(flush: false) { EmptyState(empty, actions: page.actions ?? []) { run($0, nil) } }
            }
        default: MacOverview(section: s, page: page, unfolded: unfolded, fold: fold, run: run, open: { segment = "activity" })
        }
    }
}

// invariant: SPEC A4: a workspace's Agents card: its label, its counts
// and the round + that starts one more, then one row per agent; with none,
// the one line that offers the first.
struct WorkspaceAgents: View {
    let section: Section
    let actions: [Action]
    let run: (Action, Row?) -> Void
    @Environment(\.stationTokens) private var tokens

    var start: Action? { actions.first { $0.id == "new-agent" } }

    var body: some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
            HStack(spacing: Token.Metric.Wait.headGap) {
                CardLabel(section.title ?? "", value: section.caption ?? [])
                if let start, section.card != "empty" {
                    RoundButton(Token.Symbol.plus, size: .small) { run(start, nil) }.accessibilityLabel(start.label ?? "")
                }
            }
            if let empty = section.items?.first?.empty {
                Card(flush: false) {
                    HStack {
                        Text(verbatim: empty.text ?? "").font(Token.Role.emptyGrey.font).foregroundStyle(tokens.tone(Token.Role.emptyGrey.tone))
                        Spacer(minLength: 0)
                        if let start { VerbButton(start.label ?? "") { run(start, nil) } }
                    }
                }
            } else {
                Card {
                    RowStack(section.rows, columns: section.columns ?? [], actions: actions) { action in
                        run(action, section.rows.first { $0.verb == action.id || $0.quick == action.id })
                    }
                }
            }
        }
    }
}


// value: a pushed page's footer: its copy line and big button when the
// page has them, then the status line.
struct PageFooter: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers

    var body: some View {
        VStack(spacing: Token.Metric.Panel.blocks) {
            if let footer = page.footer, footer.kind == "ssh" {
                CopyFooter(footer, actions: page.actions ?? [], copy: handlers.copy) { perform($0, handlers) }
            }
            if let status = state.status { StatusBar(status.line, failed: status.failed) }
        }
    }
}
