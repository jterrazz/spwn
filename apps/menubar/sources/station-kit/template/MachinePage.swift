import SwiftUI

// invariant: SPEC A4: a machine's page that has its header (a Mac's, a
// running workspace's) on the page frame its segments are picked in; a
// page without one, a server's or a stopped workspace's, on its own.
struct MachinePage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers

    @MainActor
    static func view(_ state: PanelState, handlers: PopoverHandlers, scrolls: Bool) -> AnyView? {
        guard let page = state.shown, page.id == PageID.machine, page.header != nil else { return nil }
        return AnyView(MacPage(page: page, state: state, handlers: handlers, scrolls: scrolls).id(page.context?.machine ?? ""))
    }

    var body: some View {
        if page.context?.kind == "server" {
            ServerPage(page: page, state: state, handlers: handlers)
        } else {
            WorkspacePage(page: page, state: state, handlers: handlers)
        }
    }
}
