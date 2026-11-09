import SwiftUI

// invariant: SPEC A5: a menu drawn by the system from its model: each
// entry a Label with its symbol, its key, red when destructive, a toggle
// when it is a switch, a nested Menu for a submenu, the header a section's
// title; the same model the preview draws.
public struct MenuContent: View {
    let model: MenuModel
    let pick: (MenuItem) -> Void

    public init(_ model: MenuModel, pick: @escaping (MenuItem) -> Void) {
        self.model = model
        self.pick = pick
    }

    public var body: some View {
        if let header = model.header {
            SwiftUI.Section(header) { entries(model.items) }
        } else {
            entries(model.items)
        }
    }

    func entries(_ items: [MenuItem]) -> some View {
        ForEach(Array(items.enumerated()), id: \.offset) { _, item in
            entry(item)
        }
    }

    @ViewBuilder
    func entry(_ item: MenuItem) -> some View {
        if item.separator {
            Divider()
        } else if !item.submenu.isEmpty {
            Menu {
                MenuContent(MenuModel(header: nil, items: item.submenu), pick: pick)
            } label: {
                label(item)
            }
        } else if item.id == MenuModel.login || item.id == MenuModel.sparkline {
            Toggle(item.title, isOn: Binding(get: { item.checked }, set: { _ in pick(item) }))
        } else {
            Button(role: item.destructive ? .destructive : nil) { pick(item) } label: { label(item) }
                .keyboardShortcut(item.key.flatMap(MenuModel.shortcut))
                .disabled(!item.enabled)
        }
    }

    @ViewBuilder
    func label(_ item: MenuItem) -> some View {
        if let icon = item.icon {
            Label(item.title, systemImage: item.checked ? Token.Symbol.checkmark : icon)
        } else if item.checked {
            Label(item.title, systemImage: Token.Symbol.checkmark)
        } else {
            Text(item.title)
        }
    }
}

// value: a view's right-click menu, drawn natively from its model; it
// picks through the panel's own pick, and opens the page it names when a
// verb runs nothing.
struct StationContextMenu: ViewModifier {
    let model: MenuModel
    let nav: String?
    @Environment(\.stationMenuPick) private var pick

    func body(content: Content) -> some View {
        if model.isEmpty {
            content
        } else {
            content.contextMenu {
                MenuContent(model) { pick?($0, nav: nav) }
            }
        }
    }
}

extension View {
    public func stationContextMenu(_ model: MenuModel, nav: String? = nil) -> some View {
        modifier(StationContextMenu(model: model, nav: nav))
    }
}
