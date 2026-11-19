import SwiftUI

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
            content.systemContextMenu(model) { pick?($0, nav: nav) }
        }
    }
}

extension View {
    public func stationContextMenu(_ model: MenuModel, nav: String? = nil) -> some View {
        modifier(StationContextMenu(model: model, nav: nav))
    }
}
