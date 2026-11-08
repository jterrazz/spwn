import SwiftUI

// invariant: a workspace's page, as today's machine page draws it until
// its own lot rebuilds it.
struct WorkspacePage: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers

    var body: some View { MachineContent(page: page, state: state, handlers: handlers) }
}
