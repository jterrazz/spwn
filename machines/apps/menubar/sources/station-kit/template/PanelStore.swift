import Observation
import SwiftUI

// value: what the panel draws in its place: Home, the page pushed over it,
// or Find over either.
public enum PanelRoute: Equatable, Sendable {
    case home, pushed, find

    public static func of(_ state: PanelState) -> PanelRoute {
        if state.find != nil { return .find }
        return state.pushed == nil ? .home : .pushed
    }
}

// invariant: the panel's state observed by slice: the engine's link, Home,
// the page pushed over it, and the rest of the panel's own state (`ui`);
// the live values are LiveStore's. A view is told only of the slices it
// read, so a change of one draws again only the views that read it.
// `state` is the panel's whole state, and writing it writes back only the
// slices that changed.
@MainActor
@Observable
public final class PanelStore {
    public private(set) var link: Connection
    public private(set) var home: Page?
    public private(set) var pages: Page?
    public private(set) var ui: PanelState
    public private(set) var route: PanelRoute
    // value: Home and the page pushed as they last stood, read without
    // being followed by the views that draw neither (a page pushed opens
    // Home's Nth Mac on cmd+N: the one Home listed when that page was last
    // drawn).
    @ObservationIgnored public private(set) var homeSeen: Page?
    @ObservationIgnored public private(set) var pagesSeen: Page?

    public init(_ state: PanelState) {
        link = state.connection
        home = state.page
        pages = state.pushed
        homeSeen = state.page
        pagesSeen = state.pushed
        ui = Self.rest(state)
        route = PanelRoute.of(state)
    }

    static func rest(_ state: PanelState) -> PanelState {
        var rest = state
        rest.connection = .connecting
        rest.page = nil
        rest.pushed = nil
        return rest
    }

    public var state: PanelState {
        get {
            var whole = ui
            whole.connection = link
            whole.page = home
            whole.pushed = pages
            return whole
        }
        set {
            if link != newValue.connection { link = newValue.connection }
            if home != newValue.page {
                home = newValue.page
                homeSeen = newValue.page
            }
            if pages != newValue.pushed {
                pages = newValue.pushed
                pagesSeen = newValue.pushed
            }
            let rest = Self.rest(newValue)
            if ui != rest { ui = rest }
            let route = PanelRoute.of(newValue)
            if self.route != route { self.route = route }
        }
    }
}

private struct EvaluatedKey: EnvironmentKey {
    static let defaultValue: @MainActor (String) -> Void = { _ in }
}

extension EnvironmentValues {
    // value: told the name of each of the store's views whose body is
    // drawn again (`route`, `home`, `pushed`, `find`): the count of what a
    // change draws, for the tests.
    public var stationEvaluated: @MainActor (String) -> Void {
        get { self[EvaluatedKey.self] }
        set { self[EvaluatedKey.self] = newValue }
    }
}

// invariant: the app's panel over its store: the route alone picks the
// view, and each view reads the slices it draws, so Home's changes never
// draw a page pushed over it, nor a page pushed's Home.
public struct StorePanel: View {
    let store: PanelStore
    let handlers: PopoverHandlers
    @Environment(\.stationEvaluated) private var evaluated

    public init(store: PanelStore, handlers: PopoverHandlers) {
        self.store = store
        self.handlers = handlers
    }

    public var body: some View {
        let _ = evaluated("route")
        SwiftUI.Group {
            switch store.route {
            case .home: HomeSlice(store: store, handlers: handlers)
            case .pushed: PushedSlice(store: store, handlers: handlers)
            case .find: FindSlice(store: store, handlers: handlers)
            }
        }
        .recorded(nil)
    }
}

// value: Home: the link, Home and the panel's own state.
struct HomeSlice: View {
    let store: PanelStore
    let handlers: PopoverHandlers
    @Environment(\.stationEvaluated) private var evaluated

    var body: some View {
        let _ = evaluated("home")
        Popover(drawn, handlers: handlers).recorded(nil)
    }

    var drawn: PanelState {
        var state = store.ui
        state.connection = store.link
        state.page = store.home
        return state
    }
}

// value: a page pushed over Home: the link, the page and the panel's own
// state; Home as it last stood.
struct PushedSlice: View {
    let store: PanelStore
    let handlers: PopoverHandlers
    @Environment(\.stationEvaluated) private var evaluated

    var body: some View {
        let _ = evaluated("pushed")
        Popover(drawn, handlers: handlers).recorded(nil)
    }

    var drawn: PanelState {
        var state = store.ui
        state.connection = store.link
        state.page = store.homeSeen
        state.pushed = store.pages
        return state
    }
}

// value: Find, over Home or a page pushed: the link and the panel's own
// state, which holds its query and index; the pages under it as they last
// stood.
struct FindSlice: View {
    let store: PanelStore
    let handlers: PopoverHandlers
    @Environment(\.stationEvaluated) private var evaluated

    var body: some View {
        let _ = evaluated("find")
        Popover(drawn, handlers: handlers).recorded(nil)
    }

    var drawn: PanelState {
        var state = store.ui
        state.connection = store.link
        state.page = store.homeSeen
        state.pushed = store.pagesSeen
        return state
    }
}
