import SwiftUI

// invariant: SPEC A4: a page's Needs you: its label and count, then its
// need rows, the section's Top before the fold, the rest behind the
// fold's own words, unfolded in place.
public struct NeedsCard: View {
    let section: Section
    let actions: [Action]
    let open: Bool
    let fold: () -> Void
    let run: (Action, Row) -> Void

    public init(_ section: Section, actions: [Action] = [], open: Bool = false, fold: @escaping () -> Void = {}, run: @escaping (Action, Row) -> Void = { _, _ in }) {
        self.section = section
        self.actions = actions
        self.open = open
        self.fold = fold
        self.run = run
    }

    public static func shown(_ section: Section, open: Bool) -> [Row] {
        let rows = section.rows
        guard !open, let top = section.top, top > 0, section.rest != nil else { return rows }
        return Array(rows.prefix(top))
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
            CardLabel(section.title ?? "", value: section.caption ?? [])
            Card {
                ForEach(Array(Self.shown(section, open: open).enumerated()), id: \.offset) { _, row in
                    NeedRow(row, actions: actions) { run($0, row) }
                }
                if let rest = section.rest?.row {
                    FoldButton(open ? rest.detail?.text ?? "" : rest.label ?? "", open: open, tap: fold)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
        }
    }
}
