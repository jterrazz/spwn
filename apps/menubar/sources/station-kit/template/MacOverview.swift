import SwiftUI

// invariant: SPEC A4 A Mac's Overview: each section by its card.
struct MacOverview: View {
    let section: Section
    let page: Page
    let unfolded: Set<String>
    let fold: (String) -> Void
    let run: (Action, Row?) -> Void
    let open: () -> Void
    @Environment(\.stationTokens) private var tokens

    var id: String { section.id ?? "" }

    var body: some View {
        if id == "vms" {
            VMsCard(section, actions: page.actions ?? [], run: run)
        } else {
            card
        }
    }

    @ViewBuilder
    var card: some View {
        switch section.card {
        case "gauges": GaugeStrip(section, open: open)
        case "care":
            CareCard(section, actions: page.actions ?? [], open: unfolded.contains("upkeep"), fold: { fold("upkeep") }) { run($0, nil) }
        case "empty":
            VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                ForEach(Array((section.items ?? []).compactMap(\.empty).enumerated()), id: \.offset) { _, empty in
                    Card(flush: false) { EmptyState(empty, actions: page.actions ?? []) { run($0, nil) } }
                }
            }
        default:
            if id == TodoBadge.section {
                NeedsCard(section, actions: page.actions ?? [], open: unfolded.contains(id), fold: { fold(id) }, run: { run($0, $1) })
            } else if section.notes.isEmpty {
                SegmentSection(section: section)
            } else {
                ForEach(Array(section.notes.enumerated()), id: \.offset) { _, note in
                    Text(note).font(Token.Role.footnote.font).foregroundStyle(tokens.tone(Token.Role.footnote.tone))
                        .lineSpacing(Token.Metric.Footnote.lineSpacing)
                        .fixedSize(horizontal: false, vertical: true)
                        .padding(Token.Metric.Footnote.padding)
                }
            }
        }
    }
}
