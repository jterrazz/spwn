import SwiftUI

// invariant: a segment's section drawn plainly: its label, then its rows,
// links and empty lines in one card.
struct SegmentSection: View {
    let section: Section

    var body: some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
            CardLabel(section.title ?? "", value: section.caption ?? [])
            Card(flush: false) {
                ItemsView(section.items ?? [], columns: section.columns ?? [], width: Token.Metric.Panel.width)
            }
        }
    }
}
