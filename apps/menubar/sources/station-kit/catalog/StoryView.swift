import SwiftUI

extension Story {
    public var connection: Connection {
        switch ui.connection {
        case "stopped": .stopped
        case "connecting": .connecting
        case "not-installed": .notInstalled
        default: .live
        }
    }

    public var state: PanelState {
        let bottom = pages.first
        let over = pages.count > 1 ? pages.last : nil
        var state: PanelState
        if over == nil, let bottom, bottom.id == PageID.switcher || bottom.id == PageID.machine {
            state = PanelState(page: nil, connection: connection, pushed: bottom)
        } else {
            state = PanelState(page: bottom, connection: connection, pushed: over)
        }
        let ui = ui
        state.confirming = ui.alert
        state.qr = ui.popover
        state.menu = ui.menu?.id == "station"
        state.search = ui.find?.query ?? ""
        state.scrub = ui.scrub
        if let sheet = ui.sheet, let answers = sheet.answers { state.answers[sheet.id] = answers }
        return state
    }
}

public struct StoryView: View {
    let story: Story
    let state: PanelState
    let handlers: PopoverHandlers
    @Environment(\.colorScheme) private var scheme

    public init(_ story: Story, state: PanelState? = nil, handlers: PopoverHandlers = PopoverHandlers()) {
        self.story = story
        self.state = state ?? story.state
        self.handlers = handlers
    }

    static func panel(_ page: Page) -> Bool { page.id == PageID.home || page.id == PageID.switcher || page.id == PageID.machine }

    public var body: some View {
        if story.ui.statusItem == true {
            MenuBarIconView(MenuBarIcon.of(connection: state.connection, page: state.page), scheme: scheme)
        } else if state.pushed != nil || state.page.map(Self.panel) == true {
            Popover(state, handlers: handlers, scrolls: false)
        } else if let own = MenuPage.view(state, handlers: handlers) {
            own
        } else if let page = state.page {
            PageView(page)
        }
    }
}
