import SwiftUI

// invariant: SPEC A4: a machine's page by its kind: a Mac's and a
// server's on their own page frame, a workspace's on its own.
struct MachinePage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers

    @MainActor
    static func view(_ state: PanelState, handlers: PopoverHandlers, scrolls: Bool) -> AnyView? {
        guard let page = state.shown, page.id == PageID.machine, page.header != nil else { return nil }
        switch page.context?.kind {
        case "mac": return AnyView(MacPage(page: page, state: state, handlers: handlers, scrolls: scrolls).id(page.context?.machine ?? ""))
        case "server": return AnyView(ServerPage(page: page, state: state, handlers: handlers, scrolls: scrolls).id(page.context?.machine ?? ""))
        default: return nil
        }
    }

    var body: some View {
        WorkspacePage(page: page, state: state, handlers: handlers)
    }
}
