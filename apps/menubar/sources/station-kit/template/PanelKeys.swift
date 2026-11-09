import Foundation

public enum PanelFocus: Hashable, Sendable {
    case panel
    case row(String)
    case control(String)

    static let controlPrefix = "control:"

    public init(_ said: String) {
        self = said.hasPrefix(Self.controlPrefix) ? .control(String(said.dropFirst(Self.controlPrefix.count))) : .row(said)
    }

    public var lit: PanelFocus? { self == .panel ? nil : self }

    public static let menu = PanelFocus.control("menu")
    public static let find = PanelFocus.control("find")
    public static let cancel = PanelFocus.control("cancel")
    public static let confirm = PanelFocus.control("confirm")
    public static let typed = PanelFocus.control("field:typed")

    public static func field(_ key: String) -> PanelFocus { .control("field:\(key)") }
    public static func more(_ group: String) -> PanelFocus { .control("more:\(group)") }
}

public enum PanelRegion: Equatable, Sendable {
    case page, sheet, alert, find, menu

    public static func of(_ state: PanelState) -> PanelRegion {
        if state.menu { return .menu }
        if let action = PanelKeys.asking(state) {
            if action.form != nil || SheetLayer.layered(action) { return .sheet }
            if action.confirm != nil { return .alert }
        }
        return state.find != nil ? .find : .page
    }
}

public enum PanelLayer: Equatable, Sendable { case menu, popover, find, sheet }

public enum PanelEffect: Equatable, Sendable {
    case none
    case close
    case dismiss(PanelLayer)
    case pop
    case home
    case find(String)
    case move(Int)
    case walk(Int)
    case open(String)
    case run(String)
    case segment(String)
    case answer(Bool)
    case pick(alternate: Bool)
}

public enum PanelOrder {
    public static func walk(_ state: PanelState) -> [PanelFocus] {
        switch PanelRegion.of(state) {
        case .menu: return []
        case .find: return find(state)
        case .alert:
            let typed = PanelKeys.asking(state)?.confirm?.type != nil
            return (typed ? [PanelFocus.typed] : []) + [.cancel, .confirm]
        case .sheet:
            let fields = (PanelKeys.asking(state)?.form?.fields ?? []).compactMap(\.key).map(PanelFocus.field)
            return fields + [.cancel, .confirm]
        case .page:
            guard let home = PanelKeys.home(state) else { return arrows(state) }
            let header = (home.header?.buttons ?? []).map(PanelFocus.control)
            return header + [.menu] + arrows(state) + [.find]
        }
    }

    public static func arrows(_ state: PanelState) -> [PanelFocus] {
        switch PanelRegion.of(state) {
        case .find: return find(state)
        case .page: break
        default: return []
        }
        guard let shown = state.shown else { return [] }
        guard let home = PanelKeys.home(state) else { return rows(of: shown, segment: state.segment).map { .row($0.id ?? "") } }
        let hero = (home.section(HomePage.hero)?.rows ?? []).dropFirst().filter { $0.nav != nil || $0.verb != nil }.map { PanelFocus.row($0.id ?? "") }
        let tiles = (home.section(HomeCards.tiles)?.items ?? []).compactMap { $0.tile?.nav }.map(PanelFocus.control)
        let cards = HomeCards.navigable(home, unfolded: Set(state.unfolded)).map { PanelFocus.row($0.id ?? "") }
        return hero + tiles + cards
    }

    static func rows(of page: Page, segment: String?) -> [Row] {
        let picked = segment ?? MacPage.picked(page)
        func rows(_ items: [Item]) -> [Row] {
            items.flatMap { item -> [Row] in
                if let row = item.row { return [row] }
                if let group = item.group { return (group.head.map { [$0] } ?? []) + rows(group.items ?? []) }
                return []
            }
        }
        let sections = page.segments == nil ? page.sections ?? [] : MacPage.sections(page, segment: picked)
        return sections.flatMap { rows($0.items ?? []) }.filter { $0.id != nil && ($0.nav != nil || $0.verb != nil) }
    }

    static func find(_ state: PanelState) -> [PanelFocus] {
        guard let query = state.find else { return [] }
        let groups = FindIndex(state.findIndex ?? Page()).results(query: query.query, recent: state.recent, open: Set(query.open))
        return FindIndex.lines(groups).map { line in
            switch line {
            case let .entry(e): .row(e.row.id ?? e.title)
            case let .more(group, _): .more(group)
            }
        }
    }

    public static func follow(_ focus: PanelFocus?, in order: [PanelFocus]) -> PanelFocus? {
        guard let focus, focus != .panel else { return nil }
        return order.contains(focus) ? focus : nil
    }

    public static func walked(_ focus: PanelFocus?, by step: Int, in order: [PanelFocus]) -> PanelFocus? {
        guard !order.isEmpty else { return nil }
        guard let focus, let at = order.firstIndex(of: focus) else { return order[step > 0 ? 0 : order.count - 1] }
        return order[((at + step) % order.count + order.count) % order.count]
    }

    public static func moved(_ focus: PanelFocus?, by step: Int, in order: [PanelFocus]) -> PanelFocus? {
        let at = focus.flatMap { order.firstIndex(of: $0) }
        return HomeKeys.moved(at, by: step, count: order.count).map { order[$0] }
    }
}

public enum PanelKeys {
    public static let shortcuts = ["cmd+[", "cmd+0", "cmd+1", "cmd+2", "cmd+3", "cmd+4", "cmd+5", "cmd+6", "cmd+7", "cmd+8", "cmd+9",
                                   "cmd+i", "cmd+p", "cmd+u", "cmd+e", "cmd+t", "shift+cmd+n", "cmd+enter"]

    static let pages = ["cmd+i": Navigation.activity, "cmd+p": Navigation.ports, "cmd+u": Navigation.accounts, "cmd+e": Navigation.awake]

    static func asking(_ state: PanelState) -> Action? {
        guard let id = state.confirming else { return nil }
        return (state.find != nil ? state.findIndex : state.shown)?.action(id)
    }

    static func home(_ state: PanelState) -> Page? {
        guard state.pushed == nil, let page = state.page, page.id == PageID.home, page.header != nil else { return nil }
        return page
    }

    static func row(_ id: String, in page: Page) -> (row: Row, section: String?)? {
        func find(_ items: [Item]) -> Row? {
            for item in items {
                if let r = item.row, r.id == id { return r }
                if let g = item.group {
                    if let h = g.head, h.id == id { return h }
                    if let r = find(g.items ?? []) { return r }
                }
            }
            return nil
        }
        for s in page.sections ?? [] {
            if let r = find(s.items ?? []) { return (r, s.id) }
        }
        return nil
    }

    static func keyed(_ key: String, in page: Page?) -> Action? {
        (page?.actions ?? []).first { $0.key == key && $0.on != "row" && !($0.id ?? "").contains("@") && $0.run != nil }
    }

    public static func effect(_ key: String, _ state: PanelState, focus: PanelFocus?) -> PanelEffect {
        let region = PanelRegion.of(state)
        if key == "esc" || key == "cmd+[" {
            if state.menu { return .dismiss(.menu) }
            if state.qr != nil || state.about != nil { return .dismiss(.popover) }
            switch region {
            case .sheet, .alert: return .dismiss(.sheet)
            case .find: return .dismiss(.find)
            default: return state.pushed != nil ? .pop : .close
            }
        }
        switch region {
        case .menu:
            return .none
        case .sheet, .alert:
            return asked(key, state, region: region, focus: focus)
        case .find:
            switch key {
            case "down": return .move(1)
            case "up": return .move(-1)
            case "enter": return .pick(alternate: false)
            case "cmd+enter": return .pick(alternate: true)
            default: return .none
            }
        case .page:
            return paged(key, state, focus: focus)
        }
    }

    static func asked(_ key: String, _ state: PanelState, region: PanelRegion, focus: PanelFocus?) -> PanelEffect {
        switch key {
        case "tab": return .walk(1)
        case "backtab": return .walk(-1)
        case "enter", "space":
            if focus == .cancel { return .answer(false) }
            if focus == .confirm || region == .sheet { return .answer(true) }
            return .answer(asking(state)?.confirm.map(AlertCard.yesIsDefault) ?? true)
        default: return .none
        }
    }

    static func paged(_ key: String, _ state: PanelState, focus: PanelFocus?) -> PanelEffect {
        let home = home(state)
        let shown = state.shown
        switch key {
        case "cmd+k": return .find("")
        case "down": return .move(1)
        case "up": return .move(-1)
        case "tab": return .walk(1)
        case "backtab": return .walk(-1)
        case "left": return state.pushed != nil ? .pop : .none
        case "enter", "space": return pressed(focus, state)
        case "cmd+enter":
            if case let .row(id)? = focus, let quick = shown.flatMap({ row(id, in: $0) })?.row.quick { return .run(quick) }
            guard shown?.id == PageID.agent || shown?.context?.kind == "workspace", let attach = keyed(key, in: shown) else { return .none }
            return .run(attach.id ?? "")
        case "cmd+n", "shift+cmd+n", "cmd+t":
            return keyed(key, in: shown).map { .run($0.id ?? "") } ?? .none
        case "cmd+0":
            return state.pushed != nil ? .home : .none
        case "cmd+i", "cmd+p", "cmd+u", "cmd+e":
            if let segment = shown?.segments?.first(where: { $0.key == key }) { return .segment(segment.id ?? "") }
            if let tile = (home?.section(HomeCards.tiles)?.items ?? []).compactMap(\.tile).first(where: { $0.key == key }), let nav = tile.nav { return .open(nav) }
            return pages[key].map { .open($0) } ?? .none
        default:
            if key.hasPrefix("cmd+"), let n = Int(key.dropFirst(4)), n > 0 {
                let macs = (state.page?.section(HomeCards.macs)?.rows ?? []).filter { $0.nav != nil }
                return n <= macs.count ? .open(macs[n - 1].nav ?? "") : .none
            }
            if home != nil, key.count == 1, key.first?.isLetter == true { return .find(key) }
            return .none
        }
    }

    static func pressed(_ focus: PanelFocus?, _ state: PanelState) -> PanelEffect {
        guard let shown = state.shown else { return .none }
        switch focus {
        case let .row(id)?:
            guard let (row, section) = row(id, in: shown) else { return .none }
            if section == HomePage.hero || section == TodoBadge.section, let verb = row.verb { return .run(verb) }
            return row.nav.map { .open($0) } ?? .none
        case let .control(id)?:
            if focus == .find { return .find("") }
            if focus == .menu { return .none }
            if shown.action(id) != nil { return .run(id) }
            return .open(id)
        default:
            return .none
        }
    }
}
