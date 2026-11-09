import SwiftUI

// invariant: SPEC A4: a machine's page by its kind, and an agent's: a
// Mac's, a workspace's and an agent's each on its own page frame; a
// server's on its own, but for its Activity, drawn on a Mac's frame.
struct MachinePage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers

    @MainActor
    static func view(_ state: PanelState, handlers: PopoverHandlers, scrolls: Bool) -> AnyView? {
        guard let page = state.shown, page.header != nil else { return nil }
        let id = page.context?.machine ?? ""
        switch (page.id, page.context?.kind) {
        case (PageID.agent, _): return AnyView(AgentPage(page: page, state: state, handlers: handlers, scrolls: scrolls).id(id))
        case (PageID.machine, "mac"): return AnyView(MacPage(page: page, state: state, handlers: handlers, scrolls: scrolls).id(id))
        case (PageID.machine, "server") where state.segment == ActivityWindow.segment:
            return AnyView(MacPage(page: page, state: state, handlers: handlers, scrolls: scrolls).id(id))
        case (PageID.machine, "workspace"): return AnyView(WorkspacePage(page: page, state: state, handlers: handlers, scrolls: scrolls).id(id))
        default: return nil
        }
    }

    var body: some View {
        if page.context?.kind == "server" {
            ServerPage(page: page, state: state, handlers: handlers)
        } else {
            MachineContent(page: page, state: state, handlers: handlers)
        }
    }
}
