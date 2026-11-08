import SwiftUI

// invariant: SPEC A4, B1: a Mac's page: its header, then the sections of
// the segment picked; the segment and what is unfolded are the client's
// own, so a pick redraws the page it holds and asks the engine nothing.
struct MacPage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    @State private var segment: String
    @State private var unfolded: Set<String>
    @Environment(\.stationStill) private var still

    init(page: Page, state: PanelState, handlers: PopoverHandlers, scrolls: Bool) {
        self.page = page
        self.state = state
        self.handlers = handlers
        self.scrolls = scrolls
        _segment = State(initialValue: state.segment ?? Self.picked(page))
        _unfolded = State(initialValue: Set(state.unfolded))
    }

    static func picked(_ page: Page) -> String {
        page.segments?.first { $0.picked == true }?.id ?? page.segments?.first?.id ?? ""
    }

    // value: the sections a segment shows: its own, and those of every
    // segment; the summary is the header's.
    static func sections(_ page: Page, segment: String) -> [Section] {
        (page.sections ?? []).filter { $0.id != "summary" && ($0.segment == nil || $0.segment == segment) }
    }

    var body: some View {
        Panel(insets: .page, bounded: scrolls, key: "\(state.openings)/\(page.context?.machine ?? "")/\(segment)", sheet: state.confirming) {
            header
        } content: {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                ForEach(Array(Self.sections(page, segment: segment).enumerated()), id: \.offset) { _, s in
                    section(s)
                }
                if state.confirming != nil { SheetFor(page: page, state: state, handlers: handlers) }
            }
        } footer: {
            if let status = state.status { StatusBar(status.line, failed: status.failed) }
        }
        .background {
            BackKey(state: state, handlers: handlers)
            if !still, BackKey.gesture(state) == .back { EscapeKey(back: handlers.back) }
        }
        .environment(\.stationHints, false)
    }

    var header: MachineHeader {
        MachineHeader(page, segment: segment, pick: { segment = $0 }, back: handlers.back, run: tap)
    }

    func tap(_ action: Action) { perform(action, handlers) }

    // value: a row's verb: its call, or the page its row opens when the
    // verb runs nothing.
    func run(_ action: Action, _ row: Row?) {
        if action.run == nil, let nav = row?.nav {
            handlers.push(nav)
        } else {
            tap(action)
        }
    }

    func fold(_ id: String) {
        if unfolded.contains(id) { unfolded.remove(id) } else { unfolded.insert(id) }
    }

    @ViewBuilder
    func section(_ s: Section) -> some View {
        switch s.segment {
        case "activity": MacActivity(section: s)
        case "ports": MacPorts(section: s, page: page, state: state, handlers: handlers, run: tap)
        case "accounts": MacAccounts(section: s)
        default: MacOverview(section: s, page: page, unfolded: unfolded, fold: fold, run: run, open: { segment = "activity" })
        }
    }
}
