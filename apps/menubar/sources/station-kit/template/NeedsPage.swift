import SwiftUI

// invariant: SPEC A4 Needs you: the way back, its title and its facts;
// one card of every need with its verb, most urgent first, then the
// upkeep under its label; with none, the one empty card. A verb that
// runs says its busy words until the page says its outcome.
struct NeedsPage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    @Environment(\.stationStill) private var still

    var running: String? { state.running ?? state.settling }

    var body: some View {
        Panel(insets: .page, bounded: scrolls, key: "\(state.openings)/\(page.id ?? "")", sheet: state.confirming) {
            MachineHeader(page, segment: nil, back: handlers.back) { perform($0, handlers) }
        } content: {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                ForEach(Array((page.sections ?? []).filter { $0.id != "summary" }.enumerated()), id: \.offset) { _, s in
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

    @ViewBuilder
    func section(_ s: Section) -> some View {
        if let empty = (s.items ?? []).compactMap(\.empty).first {
            Card(flush: false) { EmptyState(empty, centred: true) }
        } else {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
                if s.id != TodoBadge.section { CardLabel(s.title ?? "", value: s.caption ?? []) }
                Card {
                    ForEach(Array(s.rows.enumerated()), id: \.offset) { _, row in
                        NeedRow(row, actions: page.actions ?? [], running: running) { verb(page, $0, handlers) }
                    }
                }
            }
        }
    }
}
