import SwiftUI

// invariant: SPEC A4: a machine's gauges, three equal columns, each
// opening its Activity.
public struct GaugeStrip: View {
    let section: Section
    let open: () -> Void

    public init(_ section: Section, open: @escaping () -> Void = {}) {
        self.section = section
        self.open = open
    }

    public var body: some View {
        let m = Token.Metric.Gauges.self
        HStack(spacing: m.gap) {
            ForEach(Array((section.items ?? []).compactMap(\.gauge).enumerated()), id: \.offset) { _, gauge in
                Button(action: open) { GaugeTile(gauge) }.buttonStyle(.plain)
            }
        }
    }
}
