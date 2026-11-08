import SwiftUI

// invariant: SPEC A4: a machine's page by its kind: a Mac's on its own
// page frame, a workspace's and a server's on theirs; a workspace's
// Accounts on the frame a Mac's segments are drawn in.
struct MachinePage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers

    @MainActor
    static func view(_ state: PanelState, handlers: PopoverHandlers, scrolls: Bool) -> AnyView? {
        guard let page = state.shown, page.id == PageID.machine, page.header != nil else { return nil }
        if page.context?.kind != "mac", state.segment != "accounts" { return nil }
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
