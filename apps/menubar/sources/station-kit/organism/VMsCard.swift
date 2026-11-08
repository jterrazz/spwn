import SwiftUI

// invariant: SPEC A4: a Mac's VMs: its label and the memory they hold,
// one row per VM in the page's order; with none, the empty card that
// offers the first.
public struct VMsCard: View {
    let section: Section
    let actions: [Action]
    let run: (Action, Row?) -> Void

    public init(_ section: Section, actions: [Action] = [], run: @escaping (Action, Row?) -> Void = { _, _ in }) {
        self.section = section
        self.actions = actions
        self.run = run
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
            CardLabel(section.title ?? "", value: section.caption ?? [])
            if let empty = (section.items ?? []).compactMap(\.empty).first {
                Card { EmptyState(empty, actions: actions) { run($0, nil) } }
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
