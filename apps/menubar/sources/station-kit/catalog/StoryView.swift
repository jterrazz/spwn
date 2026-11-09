import SwiftUI

extension Story {
    // value: the menu a story opens, built as the app builds it: the
    // station menu, the shown page's More menu, or a row's or a tile's.
    public var menu: MenuModel? {
        guard let open = ui.menu, let page = state.shown else { return nil }
        guard let key = open.row else {
            return open.id == "station" ? MenuModel.station(page, loginItemOn: ui.loginItem ?? false, sparklineOn: false) : MenuModel.page(page)
        }
        let actions = page.actions ?? []
        for section in page.sections ?? [] {
            for item in section.items ?? [] {
                for row in [item.row].compactMap({ $0 }) + (item.group?.items ?? []).compactMap(\.row) where row.id == key {
                    return MenuModel.context(row, actions: actions)
                }
                if let tile = item.tile, actions.first(where: { $0.id == tile.toggle })?.rows?.first == key {
                    return MenuModel.tile(tile, actions: actions)
                }
            }
        }
        return nil
    }

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
        if over == nil, let bottom, bottom.id == PageID.switcher || bottom.id == PageID.machine || bottom.id == PageID.agent {
            state = PanelState(page: nil, connection: connection, pushed: bottom)
        } else {
            state = PanelState(page: bottom, connection: connection, pushed: over)
        }
        let ui = ui
        state.confirming = ui.alert
        state.qr = ui.popover
        state.search = ui.find?.query ?? ""
        state.scrub = ui.scrub
        state.segment = ui.segment
        state.unfolded = ui.unfolded ?? []
        state.lit = ui.hover ?? ui.highlight
        if let window = ui.window, let machine = state.shown?.context?.machine { state.windows[machine] = window }
        state.pending = ui.pending ?? []
        state.prOpened = ui.prOpened ?? []
        state.reply = ui.reply ?? ""
        if let sheet = ui.sheet {
            state.confirming = sheet.id
            if let answers = sheet.answers { state.answers[sheet.id] = answers }
        }
        return state
    }
}

public struct StoryView: View {
    let story: Story
    let state: PanelState
    let handlers: PopoverHandlers
    let live: LiveStore?
    @Environment(\.colorScheme) private var scheme

    public init(_ story: Story, state: PanelState? = nil, handlers: PopoverHandlers = PopoverHandlers()) {
        self.story = story
        self.state = state ?? story.state
        self.handlers = handlers
        live = story.samples.isEmpty ? nil : Self.live(story.samples)
    }

    static func panel(_ page: Page) -> Bool { page.id == PageID.home || page.id == PageID.switcher || page.id == PageID.machine || page.id == PageID.agent }

    public var body: some View {
        SwiftUI.Group {
            if let live {
                staged.environment(\.stationSamples, story.samples).environment(\.stationLive, live)
            } else {
                staged
            }
        }
        .environment(\.stationMenuPick, MenuPick(handlers))
    }

    // invariant: SPEC C4: a menu story draws its menu's preview where the
    // menu opens, over the page it opens on.
    var staged: some View {
        drawn.overlayPreferenceValue(MenuAnchors.self) { anchors in
            if let open = story.ui.menu, let model = story.menu {
                GeometryReader { proxy in
                    let key = open.row.map(MenuAnchors.row) ?? MenuAnchors.page
                    if let anchor = anchors[key] {
                        MenuStage(model, child: open.submenu, anchor: proxy[anchor], page: open.row == nil, in: proxy.size)
                    }
                }
            }
        }
    }

    // value: the measure a CPU's live key starts with (`cpu:`, `cpuh:`).
    static let cpu = "cpu"

    // value: a story's samples as the engine's live frame would carry
    // them: a CPU's share in tenths of a percent, any other as its count.
    public static func live(_ samples: [String: Double]) -> LiveStore {
        let store = LiveStore()
        var values: [String: Value] = [:]
        for (key, sample) in samples {
            let cpu = key.hasPrefix(Self.cpu)
            var v = Value(kind: cpu ? "percent" : "count")
            v.n = Int64((cpu ? sample * Token.Live.tenths : sample).rounded())
            values[key] = v
        }
        store.take(LiveSample(at: "", values: values))
        return store
    }

    @ViewBuilder
    var drawn: some View {
        if story.ui.statusItem == true {
            MenuBarStrip(MenuBarIcon.of(connection: state.connection, page: state.page), pressed: story.ui.pressed == true, turn: story.freeze ?? 0)
        } else if state.pushed != nil || state.page.map(Self.panel) == true {
            Popover(state, handlers: handlers, scrolls: false)
        } else if let own = MenuPage.view(state, handlers: handlers, scrolls: false) {
            own
        } else if let page = state.page {
            PageView(page)
        }
    }
}

public struct MenuBarStrip: View {
    let icon: MenuBarIcon
    let pressed: Bool
    let turn: Double
    @Environment(\.colorScheme) private var scheme

    public init(_ icon: MenuBarIcon, pressed: Bool = false, turn: Double = 0) {
        self.icon = icon
        self.pressed = pressed
        self.turn = turn
    }

    public var body: some View {
        HStack(spacing: Token.Item.stripGap) {
            StatusItem(icon, scheme: scheme, turn: turn)
                .padding(.horizontal, Token.Item.pressedPad.width)
                .padding(.vertical, Token.Item.pressedPad.height)
                .background(pressed ? Token.Item.pressed(scheme) : .clear, in: RoundedRectangle(cornerRadius: Token.Item.pressedRadius))
                .accessibilityElement(children: .ignore)
                .accessibilityLabel(icon.label)
            ForEach(Token.Item.strip, id: \.self) { name in
                Image(systemName: name)
                    .font(.system(size: Token.Icon.glyph, weight: Token.Icon.weight))
                    .foregroundStyle(Token.Icon.ink(scheme))
            }
        }
        .padding(.trailing, Token.Stage.room)
        .frame(width: Token.Stage.strip.width, height: Token.Stage.strip.height, alignment: .trailing)
        .background(Token.Item.bar(scheme))
    }
}
