import SwiftUI

// invariant: SPEC A4: a machine's Ports segment, each section by its card;
// a share whose QR is shown unfolds it under its row.
struct MacPorts: View {
    let section: Section
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers
    let run: (Action) -> Void

    var body: some View {
        PortsSection(section, actions: page.actions ?? [], run: run, qr: handlers.toggleQR, about: { _ in handlers.toggleAbout(section.id ?? "") }, shown: state.qr, opened: state.about != nil && state.about == section.id)
    }
}
