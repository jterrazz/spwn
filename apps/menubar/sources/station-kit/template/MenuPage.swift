import SwiftUI

// value: the menu bar's own pages of SPEC §10.2 a page id names; any other
// page draws as PageView.
public enum MenuPage {
    public static let ports = Navigation.ports
    public static let awake = Navigation.awake
    public static let activity = Navigation.activity
    public static let accounts = Navigation.accounts

    @MainActor
    public static func view(_ state: PanelState, handlers: PopoverHandlers = PopoverHandlers(), scrolls: Bool = true) -> AnyView? {
        guard let page = state.page else { return nil }
        if page.id == accounts { return AnyView(AccountsPage(state, handlers: handlers, scrolls: scrolls)) }
        let body: AnyView
        switch page.id {
        case ports: body = AnyView(PortsPage(state, handlers: handlers))
        case awake: body = AnyView(AwakePage(state, handlers: handlers))
        case activity: body = AnyView(ActivityPage(state, handlers: handlers))
        default: return nil
        }
        return AnyView(MenuPageFrame(state: state, handlers: handlers, scrolls: scrolls) { body })
    }
}

struct MenuPageFrame<Content: View>: View {
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    let content: Content

    init(state: PanelState, handlers: PopoverHandlers, scrolls: Bool, @ViewBuilder content: () -> Content) {
        self.state = state
        self.handlers = handlers
        self.scrolls = scrolls
        self.content = content()
    }

    var body: some View {
        Panel(insets: .flush, bounded: scrolls, key: "\(state.openings)/\(state.page?.id ?? "")", sheet: state.confirming) {
            VStack(alignment: .leading, spacing: Token.Tile.section) {
                content
            }
            .padding(Token.Space.margin)
        } footer: {
            if let status = state.status {
                StatusBar(status.line, failed: status.failed).padding(EdgeInsets(top: 0, leading: Token.Space.margin, bottom: Token.Space.margin, trailing: Token.Space.margin))
            }
        }
        .background { BackKey(state: state, handlers: handlers) }
        .environment(\.stationHints, false)
    }
}

extension Page {
    // value: the actions a row offers, in their priority.
    public func actions(for row: String) -> [Action] {
        (actions ?? []).filter { $0.on == "row" && $0.rows?.contains(row) == true && $0.run != nil }.sorted { ($0.priority ?? 0) < ($1.priority ?? 0) }
    }

    public func summary() -> Row? { section("summary")?.items?.first?.row }
}

extension Section {
    // value: a row's cell under the column a key names; the first column
    // is the row's label.
    public func cell(_ row: Row, _ key: String) -> Value? {
        guard let i = columns?.firstIndex(where: { $0.key == key }), i > 0, let cells = row.cells, i - 1 < cells.count else { return nil }
        let v = cells[i - 1]
        return v.isNone ? nil : v
    }

    public var rows: [Row] { (items ?? []).compactMap(\.row) }
    public var notes: [String] { (items ?? []).compactMap { $0.note?.text } }
}

// invariant: an action is run as SPEC §10.2 says: a confirmation or a form
// opens its sheet first, any other runs at once.
struct Act {
    let state: PanelState
    let handlers: PopoverHandlers

    func tap(_ action: Action) {
        if action.confirm != nil || action.form != nil {
            handlers.confirm(action)
        } else {
            handlers.run(action, state.answers[action.id ?? ""] ?? [:])
        }
    }

    func open(_ action: Action) -> Bool { state.confirming == action.id }

    @ViewBuilder
    func sheet(_ action: Action) -> some View {
        if open(action), let page = state.page {
            SheetFor(page: page, state: state, handlers: handlers)
        }
    }

    func label(_ action: Action) -> String {
        (action.label ?? "") + (action.form != nil ? Token.Glyph.ellipsis : "")
    }
}
