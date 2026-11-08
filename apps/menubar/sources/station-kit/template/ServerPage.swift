import SwiftUI

// invariant: SPEC A4, B1: a server's page: its header, then the sections
// of the segment picked, then its ssh footer; the segment and what is
// unfolded are the client's own, so a pick redraws the page it holds and
// asks the engine nothing.
struct ServerPage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    @State private var segment: String
    @State private var unfolded: Set<String>
    @Environment(\.stationTokens) private var tokens

    init(page: Page, state: PanelState, handlers: PopoverHandlers, scrolls: Bool) {
        self.page = page
        self.state = state
        self.handlers = handlers
        self.scrolls = scrolls
        _segment = State(initialValue: state.segment ?? MacPage.picked(page))
        _unfolded = State(initialValue: Set(state.unfolded))
    }

    var actions: [Action] { page.actions ?? [] }

    var body: some View {
        Panel(insets: .page, bounded: scrolls, key: "\(state.openings)/\(page.context?.machine ?? "")/\(segment)", sheet: state.confirming) {
            MachineHeader(page, segment: segment, pick: { segment = $0 }, back: handlers.back, run: tap, go: handlers.push)
        } content: {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                ForEach(Array(MacPage.sections(page, segment: segment).enumerated()), id: \.offset) { _, s in
                    section(s)
                }
                if state.confirming != nil { SheetFor(page: page, state: state, handlers: handlers) }
            }
        } footer: {
            if let status = state.status {
                StatusBar(status.line, failed: status.failed)
            } else if let footer = page.footer {
                SSHFooter(footer, actions: actions, run: tap)
            }
        }
        .background {
            BackKey(state: state, handlers: handlers)
            EscapeKey(back: handlers.back)
        }
        .environment(\.stationHints, false)
    }

    func tap(_ action: Action) { perform(action, handlers) }

    func fold(_ id: String) {
        if unfolded.contains(id) { unfolded.remove(id) } else { unfolded.insert(id) }
    }

    @ViewBuilder
    func section(_ s: Section) -> some View {
        if s.segment == "activity" {
            MacActivity(section: s)
        } else {
            switch s.card {
            case "health": HealthCard(s, actions: actions, open: unfolded.contains(s.id ?? ""), fold: { fold(s.id ?? "") }, run: tap)
            case "chart": LatencyCard(s)
            case "uptime": UptimeCard(s)
            case "log": LogCard(s, actions: actions, run: tap)
            case "gauges":
                if Self.waiting(s) { SkeletonGauges(s) } else { GaugeStrip(s, open: { segment = "activity" }) }
            case "flush": ports(s)
            default: footnote(s)
            }
        }
    }

    // value: the gauges wait as skeletons while none of them is read yet
    // (a restart, M17).
    static func waiting(_ s: Section) -> Bool {
        let gauges = (s.items ?? []).compactMap(\.gauge)
        return !gauges.isEmpty && gauges.allSatisfy { $0.value?.kind == "pending" }
    }

    func ports(_ s: Section) -> some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
            CardLabel(s.title ?? "", value: s.caption ?? [])
            if let empty = (s.items ?? []).compactMap(\.empty).first {
                Card { EmptyState(empty, actions: actions, run: tap) }
            } else {
                Card {
                    ForEach(Array(s.rows.enumerated()), id: \.offset) { i, row in
                        if i > 0 { RowSeparator() }
                        PortRow(row, actions: actions, run: tap)
                    }
                }
            }
        }
    }

    func footnote(_ s: Section) -> some View {
        ForEach(Array(s.notes.enumerated()), id: \.offset) { _, note in
            Text(note).font(Token.Role.toolingNote.font).foregroundStyle(tokens.tone(Token.Role.toolingNote.tone))
                .lineSpacing(Token.Metric.Footnote.lineSpacing)
                .fixedSize(horizontal: false, vertical: true)
                .padding(Token.Metric.Footnote.padding)
        }
    }
}
