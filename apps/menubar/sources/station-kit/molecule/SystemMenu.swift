import AppKit
import SwiftUI

// invariant: SPEC A5: a menu is handed to the system as an NSMenu built
// from its model, so what the preview draws the menu bar shows: the
// header a section's title, red what destroys, each key as a key
// equivalent, the picked entry checked, a popup's note in a grey column.
@MainActor
public enum SystemMenu {
    public static func make(_ model: MenuModel, pick: @escaping (MenuItem) -> Void) -> NSMenu {
        let menu = NSMenu()
        menu.autoenablesItems = false
        if let header = model.header { menu.addItem(.sectionHeader(title: header)) }
        let tab = noteStop(model.items)
        for item in model.items { menu.addItem(entry(item, tab: tab, pick: pick)) }
        return menu
    }

    static func entry(_ item: MenuItem, tab: CGFloat?, pick: @escaping (MenuItem) -> Void) -> NSMenuItem {
        if item.separator { return .separator() }
        let entry = NSMenuItem(title: item.title, action: #selector(Pick.fire(_:)), keyEquivalent: "")
        let target = Pick { pick(item) }
        entry.target = target
        entry.representedObject = target
        entry.isEnabled = item.enabled
        entry.state = item.checked ? .on : .off
        if let title = attributed(item, tab: tab) { entry.attributedTitle = title }
        if let icon = item.icon { entry.image = NSImage(systemSymbolName: icon, accessibilityDescription: nil) }
        if let key = item.key.flatMap(keyEquivalent) {
            entry.keyEquivalent = key.key
            entry.keyEquivalentModifierMask = key.modifiers
        }
        if !item.submenu.isEmpty {
            entry.action = nil
            entry.submenu = make(MenuModel(header: nil, items: item.submenu), pick: pick)
        }
        return entry
    }

    // value: the words an entry is drawn in when the plain title cannot say
    // them: red when it destroys, its note after a right-aligned stop.
    public static func attributed(_ item: MenuItem, tab: CGFloat?) -> NSAttributedString? {
        guard item.destructive || item.note != nil else { return nil }
        let font = NSFont.menuFont(ofSize: 0)
        let ink = item.destructive ? Token.SystemMenu.destructive : item.enabled ? Token.SystemMenu.text : Token.SystemMenu.off
        let out = NSMutableAttributedString(string: item.title, attributes: [.font: font, .foregroundColor: ink])
        if let note = item.note, let tab {
            let style = NSMutableParagraphStyle()
            style.tabStops = [NSTextTab(textAlignment: .right, location: tab)]
            out.append(NSAttributedString(string: "\t" + note, attributes: [.font: font, .foregroundColor: Token.SystemMenu.note]))
            out.addAttribute(.paragraphStyle, value: style, range: NSRange(location: 0, length: out.length))
        }
        return out
    }

    // value: where a popup's grey column ends: past its widest words and
    // widest note, a gap between.
    static func noteStop(_ items: [MenuItem]) -> CGFloat? {
        let noted = items.filter { $0.note != nil }
        guard !noted.isEmpty else { return nil }
        let font = NSFont.menuFont(ofSize: 0)
        func width(_ s: String) -> CGFloat { (s as NSString).size(withAttributes: [.font: font]).width }
        let words = items.map { width($0.title) }.max() ?? 0
        let notes = noted.compactMap { $0.note.map(width) }.max() ?? 0
        return ceil(words + Token.SystemMenu.noteGap + notes)
    }

    // value: the key equivalent and modifiers a page's key names: "cmd+n",
    // "shift+cmd+n", "enter", "opt+enter".
    public static func keyEquivalent(_ key: String) -> (key: String, modifiers: NSEvent.ModifierFlags)? {
        let parts = key.split(separator: "+").map(String.init)
        guard let last = parts.last else { return nil }
        var mods: NSEvent.ModifierFlags = []
        for part in parts.dropLast() {
            switch part {
            case "cmd": mods.insert(.command)
            case "shift": mods.insert(.shift)
            case "opt", "alt": mods.insert(.option)
            case "ctrl": mods.insert(.control)
            default: return nil
            }
        }
        if last == "enter" || last == "return" { return ("\r", mods) }
        guard last.count == 1 else { return nil }
        return (last.lowercased(), mods)
    }

    // value: where a menu opens from the view it hangs from, in that view's
    // flipped space: right-aligned under a button, or a popup's picked entry
    // over its field.
    public enum Placement: Equatable {
        case under
        case over
    }

    public static func origin(_ placement: Placement, menu: NSSize, in bounds: CGRect) -> NSPoint {
        switch placement {
        case .under: NSPoint(x: bounds.maxX - menu.width, y: bounds.maxY + Token.SystemMenu.drop)
        case .over: NSPoint(x: bounds.minX, y: bounds.minY)
        }
    }

    static func popUp(_ menu: NSMenu, _ placement: Placement, in view: NSView) {
        let picked = placement == .over ? menu.items.first { $0.state == .on } : nil
        menu.popUp(positioning: picked, at: origin(picked == nil ? .under : .over, menu: menu.size, in: view.bounds), in: view)
    }
}

final class Pick: NSObject {
    let run: () -> Void

    init(_ run: @escaping () -> Void) {
        self.run = run
    }

    @objc func fire(_ sender: NSMenuItem) { run() }
}

// invariant: a button that opens its menu on the first click, even while
// the panel is not yet key, and lays the menu where the page draws it.
final class MenuOpenerView: NSView {
    var menuOf: () -> NSMenu? = { nil }
    var placement: SystemMenu.Placement = .under
    var context = false

    override var isFlipped: Bool { true }

    override func acceptsFirstMouse(for event: NSEvent?) -> Bool { true }

    override func hitTest(_ point: NSPoint) -> NSView? {
        guard context else { return super.hitTest(point) }
        guard let e = NSApp.currentEvent else { return nil }
        let asks = e.type == .rightMouseDown || e.type == .rightMouseUp || e.type == .leftMouseDown && e.modifierFlags.contains(.control)
        return asks ? super.hitTest(point) : nil
    }

    override func mouseDown(with event: NSEvent) {
        guard let menu = menuOf() else { return }
        if context || event.modifierFlags.contains(.control) {
            NSMenu.popUpContextMenu(menu, with: event, for: self)
        } else {
            SystemMenu.popUp(menu, placement, in: self)
        }
    }

    override func rightMouseDown(with event: NSEvent) {
        guard let menu = menuOf() else { return super.rightMouseDown(with: event) }
        NSMenu.popUpContextMenu(menu, with: event, for: self)
    }
}

struct MenuOpener: NSViewRepresentable {
    let menu: () -> NSMenu?
    let placement: SystemMenu.Placement
    let rightClick: Bool

    func makeNSView(context: NSViewRepresentableContext<MenuOpener>) -> MenuOpenerView { MenuOpenerView() }

    func updateNSView(_ view: MenuOpenerView, context: NSViewRepresentableContext<MenuOpener>) {
        view.menuOf = menu
        view.placement = placement
        view.context = rightClick
    }
}

// value: the opener laid over a view in the app; drawn still, the view
// alone, as the offscreen renderer draws no AppKit view.
struct SystemMenuOverlay: ViewModifier {
    let opener: MenuOpener
    @Environment(\.stationStill) private var still

    func body(content: Content) -> some View {
        if still { content } else { content.overlay(opener) }
    }
}

extension View {
    // value: this view opens the system menu of its model on a click, as a
    // button does, still drawn as itself.
    public func systemMenu(_ model: MenuModel, _ placement: SystemMenu.Placement = .under, pick: @escaping (MenuItem) -> Void) -> some View {
        modifier(SystemMenuOverlay(opener: MenuOpener(menu: { model.isEmpty ? nil : SystemMenu.make(model, pick: pick) }, placement: placement, rightClick: false)))
    }

    // value: this view's right-click menu, the system's, built from its
    // model; every other click goes through to the view.
    public func systemContextMenu(_ model: MenuModel, pick: @escaping (MenuItem) -> Void) -> some View {
        modifier(SystemMenuOverlay(opener: MenuOpener(menu: { model.isEmpty ? nil : SystemMenu.make(model, pick: pick) }, placement: .under, rightClick: true)))
    }
}
