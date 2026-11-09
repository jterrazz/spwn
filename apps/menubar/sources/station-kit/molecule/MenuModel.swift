import SwiftUI

// value: one entry of a menu, as the page says it: its words, its symbol,
// its key, whether it is red, picked or off, its submenu, and the verb it
// runs; nil words draw a separator.
public struct MenuItem: Equatable, Sendable {
    public var id: String
    public var title: String
    public var icon: String?
    public var key: String?
    public var destructive: Bool
    public var enabled: Bool
    public var checked: Bool
    public var submenu: [MenuItem]
    public var action: Action?
    // value: the link a QR item shows, the one its row opens or copies.
    public var link: String?

    public var separator: Bool { id == MenuModel.separator }

    static let line = MenuItem(id: MenuModel.separator, title: "", destructive: false, enabled: false, checked: false, submenu: [])
}

// invariant: SPEC A5: every menu is first a value, built by one pure
// function from the page's actions, so the native menu and its drawn
// preview are one model.
public struct MenuModel: Equatable, Sendable {
    public var header: String?
    public var items: [MenuItem]

    public static let separator = "-"
    static let title = "title:"
    static let menu = "menu"
    static let qr = "qr:"
    static let links = ["open-port:", "copy-port:", "copy:"]

    public static let login = "app:login"
    public static let sparkline = "app:sparkline"
    public static let quit = "app:quit"

    public var isEmpty: Bool { items.allSatisfy(\.separator) }

    // value: the menu of a list of action ids: a "title:" entry first is
    // its header, "-" a separator, each other id the action it names.
    public static func list(_ ids: [String], actions: [Action], checked: Set<String> = []) -> MenuModel {
        var ids = ids[...]
        var header: String?
        if let first = ids.first, first.hasPrefix(title) {
            header = action(first, in: actions)?.label
            ids = ids.dropFirst()
        }
        return MenuModel(header: header, items: ids.compactMap { item($0, actions: actions, checked: checked) })
    }

    static func item(_ id: String, actions: [Action], checked: Set<String>) -> MenuItem? {
        if id == separator { return .line }
        guard let a = action(id, in: actions) else { return nil }
        let sub = (a.submenu ?? []).compactMap { item($0, actions: actions, checked: checked) }
        let asks = a.confirm != nil || a.form != nil || a.sheet != nil || id.hasPrefix(qr)
        let words = (a.label ?? "") + (asks && !(a.label ?? "").hasSuffix(Token.Glyph.ellipsis) ? Token.Glyph.ellipsis : "")
        return MenuItem(id: id, title: words, icon: a.icon, key: a.key, destructive: a.destructive == true || a.confirm?.danger == true,
                        enabled: a.disabled == nil, checked: a.picked == true || checked.contains(id), submenu: sub, action: a, link: link(of: a, in: actions))
    }

    static func action(_ id: String, in actions: [Action]) -> Action? { actions.first { $0.id == id } }

    // value: the link a QR entry shows: its row's own link, opened or
    // copied.
    static func link(of a: Action, in actions: [Action]) -> String? {
        guard let id = a.id, id.hasPrefix(qr) else { return nil }
        let key = String(id.dropFirst(qr.count).prefix { $0 != "@" })
        for prefix in links {
            if let run = action(prefix + key, in: actions)?.run, let url = run.open ?? run.copy { return url }
        }
        return nil
    }

    // value: a row's right-click menu: the ids its row lists.
    public static func context(_ row: Row, actions: [Action]) -> MenuModel {
        list(row.menu ?? [], actions: actions)
    }

    // value: a tile's right-click menu: the menu its switch's row names.
    public static func tile(_ tile: Tile, actions: [Action]) -> MenuModel {
        guard let key = action(tile.toggle ?? "", in: actions)?.rows?.first,
              let head = action(menu + ":" + key, in: actions) else { return MenuModel(header: nil, items: []) }
        return list(head.submenu ?? [], actions: actions)
    }

    // value: a page's More menu.
    public static func page(_ page: Page) -> MenuModel {
        let actions = page.actions ?? []
        return list(action(menu, in: actions)?.submenu ?? [], actions: actions)
    }

    // value: the station menu: Home's More menu, the app's own switches
    // checked as they stand.
    public static func station(_ page: Page, loginItemOn: Bool, sparklineOn: Bool) -> MenuModel {
        let actions = page.actions ?? []
        let on = Set([loginItemOn ? login : nil, sparklineOn ? sparkline : nil].compactMap { $0 })
        return list(action(menu, in: actions)?.submenu ?? [], actions: actions, checked: on)
    }

    // value: the glyphs a key is said by in a menu: "⇧⌘N", "⌘↩", "↩".
    public static func keys(_ key: String) -> String {
        key.split(separator: "+").map { part -> String in
            switch part {
            case "cmd": Token.MenuKey.command
            case "shift": Token.MenuKey.shift
            case "opt", "alt": Token.MenuKey.option
            case "ctrl": Token.MenuKey.control
            case "enter", "return": Token.MenuKey.enter
            default: part.uppercased()
            }
        }.joined()
    }

    // value: the key a native menu item answers, and its modifiers.
    public static func shortcut(_ key: String) -> KeyboardShortcut? {
        let parts = key.split(separator: "+").map(String.init)
        guard let last = parts.last else { return nil }
        var mods: EventModifiers = []
        for part in parts.dropLast() {
            switch part {
            case "cmd": mods.insert(.command)
            case "shift": mods.insert(.shift)
            case "opt", "alt": mods.insert(.option)
            case "ctrl": mods.insert(.control)
            default: return nil
            }
        }
        if last == "enter" || last == "return" { return KeyboardShortcut(.return, modifiers: mods) }
        guard last.count == 1, let c = last.first else { return nil }
        return KeyboardShortcut(KeyEquivalent(c), modifiers: mods)
    }
}

// value: what picking a menu's entry does: the app's own switch, a link
// opened or copied, a QR shown, the row's page opened when the verb runs
// nothing, else the verb as any gesture runs it (its sheet or question
// first).
public struct MenuPick {
    let handlers: PopoverHandlers

    public init(_ handlers: PopoverHandlers) {
        self.handlers = handlers
    }

    public func callAsFunction(_ item: MenuItem, nav: String?) {
        switch item.id {
        case MenuModel.login: return handlers.loginItem(!handlers.loginItemOn)
        case MenuModel.sparkline: return handlers.sparkline(!handlers.sparklineOn)
        case MenuModel.quit: return handlers.quit()
        default: break
        }
        guard let a = item.action else { return }
        if let link = item.link { return handlers.toggleQR(link) }
        if let open = a.run?.open, let url = URL(string: open) { return handlers.open(url) }
        if let text = a.run?.copy { return handlers.copy(text) }
        if a.run == nil, a.form == nil, a.sheet == nil, a.confirm == nil {
            if let nav { handlers.push(nav) }
            return
        }
        perform(a, handlers)
    }
}

extension EnvironmentValues {
    @Entry public var stationMenuPick: MenuPick? = nil
}
