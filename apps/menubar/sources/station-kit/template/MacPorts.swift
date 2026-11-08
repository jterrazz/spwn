import SwiftUI

// invariant: SPEC A4: a machine's Ports segment, each section by its card.
struct MacPorts: View {
    let section: Section
    let page: Page
    let run: (Action) -> Void
    let qr: (String) -> Void

    var body: some View { PortsSection(section, actions: page.actions ?? [], run: run, qr: qr) }
}
