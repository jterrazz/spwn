import SwiftUI

public enum Connection: Equatable, Sendable {
    case connecting, live, stopped, notInstalled
}

public enum PageID {
    public static let home = "menubar"
    public static let switcher = "menubar.switcher"
    public static let machine = "menubar.machine"

    // value: the watch name of one machine's page (SPEC §10.3).
    public static func of(_ machine: String) -> String { "\(Self.machine):\(machine)" }

    // value: the page a call made now may have changed: the page pushed,
    // else Home.
    public static func shown(_ pushed: String?) -> String { pushed ?? home }
}

public struct PanelState: Equatable, Sendable {
    public var page: Page?
    public var connection: Connection
    public var running: String?
    public var confirming: String?
    public var status: Status?
    public var qr: String?
    public var menu = false
    public var answers: [String: [String: String]] = [:]
    // value: the page pushed over Home: the switcher, or one machine's.
    public var pushed: Page?
    // value: the machines last opened, newest first: the menu bar's own.
    public var recent: [String] = []
    public var search = ""
    // value: the words of the way back, the page under the one pushed;
    // nil when Home is under it.
    public var back: String?
    // value: the call that returned and whose page is not read again yet:
    // its control stays pending until the page that says its outcome comes.
    public var settling: String?
    // value: how many times the panel opened: each opening starts at its
    // top.
    public var openings = 0
    // value: Awake's switches as the owner last set them, until the page
    // read again says the same (`Lid`, `Battery`: "true" or "false").
    public var held: [String: String] = [:]
    // value: the page the status line was said on, by its watch; nil is
    // Home.
    public var statusOn: String?
    // value: the answers of the call that runs or settles, so its page
    // shows what it asked at once (the port being shared).
    public var sent: [String: String] = [:]
    // value: the VMs Stop everything stops, as Home listed them when it
    // was asked: each says Stopping until its line says it stopped.
    public var stopping: [String] = []
    public var scrub: Int?
    // value: the UI state a page keeps on the client: the segment picked
    // on a machine page, and what is unfolded there.
    public var segment: String?
    public var unfolded: [String] = []
    // value: the row the pointer or the keyboard lights when the panel is
    // drawn, by its id.
    public var lit: String?
    // value: Activity's window last picked, by machine: the menu bar's own.
    public var windows: [String: String] = [:]
    // value: the verbs that wait on what they started (a sign-in while
    // Terminal runs the login): each reads its busy words until the page
    // no longer offers it.
    public var pending: [String] = []

    public init(page: Page?, connection: Connection, running: String? = nil, confirming: String? = nil, status: Status? = nil, qr: String? = nil, pushed: Page? = nil, recent: [String] = []) {
        self.page = page
        self.connection = connection
        self.running = running
        self.confirming = confirming
        self.status = status
        self.qr = qr
        self.pushed = pushed
        self.recent = recent
    }

    // value: the page the owner sees, the pushed one over Home.
    public var shown: Page? { pushed ?? page }

    // value: the verbs drawn busy: the one running, and those pending.
    public var busy: Set<String> { Set(pending + [running].compactMap { $0 }) }

    // invariant: a status line is about the page it was said on: once
    // another page is shown it is gone, never carried onto the next one.
    public mutating func say(_ status: Status?, on watch: String?) {
        self.status = status
        statusOn = watch
    }

    public mutating func showing(_ watch: String?) {
        if watch != statusOn { status = nil }
    }
}

public struct PopoverHandlers {
    public var run: (Action, [String: String]) -> Void
    public var confirm: (Action?) -> Void
    public var answer: (String, String, String) -> Void
    public var open: (URL) -> Void
    public var toggleQR: (String) -> Void
    public var toggleMenu: () -> Void
    public var loginItem: (Bool) -> Void
    public var loginItemOn: Bool
    public var sparkline: (Bool) -> Void
    public var sparklineOn: Bool
    public var quit: () -> Void
    public var search: (String) -> Void
    public var copy: (String) -> Void
    // value: navigation: open a machine's page, the switcher, a page by its
    // watch name, Activity's window, back to Home.
    public var openMachine: (String) -> Void
    public var openSwitcher: () -> Void
    public var push: (String) -> Void
    public var window: (String) -> Void
    public var back: () -> Void
    // value: what the panel does once an action hands the owner to another
    // app (a link opened, a terminal): it closes, as a menu does.
    public var close: () -> Void
    public var scrub: (Int?) -> Void = { _ in }
    // value: the window picked on a machine's Activity, to remember.
    public var remember: (String, String) -> Void = { _, _ in }

    public init(run: @escaping (Action, [String: String]) -> Void = { _, _ in }, confirm: @escaping (Action?) -> Void = { _ in },
                answer: @escaping (String, String, String) -> Void = { _, _, _ in }, open: @escaping (URL) -> Void = { _ in },
                toggleQR: @escaping (String) -> Void = { _ in }, toggleMenu: @escaping () -> Void = {}, loginItem: @escaping (Bool) -> Void = { _ in }, loginItemOn: Bool = false,
                sparkline: @escaping (Bool) -> Void = { _ in }, sparklineOn: Bool = false, quit: @escaping () -> Void = {},
                openMachine: @escaping (String) -> Void = { _ in }, openSwitcher: @escaping () -> Void = {}, search: @escaping (String) -> Void = { _ in },
                copy: @escaping (String) -> Void = { _ in }, window: @escaping (String) -> Void = { _ in },
                push: @escaping (String) -> Void = { _ in }, back: @escaping () -> Void = {}, close: @escaping () -> Void = {}) {
        self.run = run
        self.confirm = confirm
        self.answer = answer
        self.open = open
        self.toggleQR = toggleQR
        self.toggleMenu = toggleMenu
        self.loginItem = loginItem
        self.loginItemOn = loginItemOn
        self.sparkline = sparkline
        self.sparklineOn = sparklineOn
        self.quit = quit
        self.openMachine = openMachine
        self.openSwitcher = openSwitcher
        self.search = search
        self.copy = copy
        self.window = window
        self.push = push
        self.back = back
        self.close = close
    }
}

extension PopoverHandlers {
    // invariant: a gesture that swaps the page (open, back) acts once the
    // click that made it has ended: a page torn down under its own click
    // leaves the next click spent on the page that is gone.
    public func deferred(_ later: @escaping (@escaping () -> Void) -> Void) -> PopoverHandlers {
        var h = self
        let (push, openMachine, openSwitcher, back, window, confirm) = (self.push, self.openMachine, self.openSwitcher, self.back, self.window, self.confirm)
        h.push = { page in later { push(page) } }
        h.openMachine = { name in later { openMachine(name) } }
        h.openSwitcher = { later { openSwitcher() } }
        h.back = { later { back() } }
        h.window = { flag in later { window(flag) } }
        h.confirm = { action in later { confirm(action) } }
        return h
    }
}

public enum TodoBadge {
    public static let section = "needs-you"

    public static func todo(_ page: Page) -> (mark: Mark, n: Int)? {
        guard let s = page.section(section), page.state != "loading" || s.counts?.contains(where: { $0.mark != .pending }) == true else { return nil }
        let counts = (s.counts ?? []).filter { $0.mark != .pending && $0.mark != .ok }
        let n = counts.reduce(0) { $0 + ($1.n ?? 0) }
        let worst = counts.compactMap(\.mark).max { $0.rank < $1.rank } ?? .ok
        return (worst, n)
    }

    public static func text(_ page: Page) -> String? {
        guard let (_, n) = todo(page) else { return nil }
        return n == 0 ? "all good" : "\(n) to do"
    }
}

public struct Popover: View {
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    public init(_ state: PanelState, handlers: PopoverHandlers = PopoverHandlers(), scrolls: Bool = true) {
        self.state = state
        self.handlers = handlers
        self.scrolls = scrolls
    }

    // value: the menu bar's own pages Home opens: This Mac's Activity over
    // 1 h, every Mac's Awake, every Mac's Ports.
    public static let pages: [(label: String, watch: String)] = [
        ("Activity", Navigation.activity), ("Awake", Navigation.awake), ("Ports", Navigation.ports),
    ]

    // invariant: the panel is the MenuBarExtra's own glass; nothing in it
    // is glass again (WWDC25: no glass on glass, none in the content).
    public var body: some View {
        if let machine = MachinePage.view(state, handlers: handlers, scrolls: scrolls) {
            machine
        } else if let own = pushedView {
            own
        } else {
            panel
        }
    }

    // value: a page of the menu bar's own (Ports, Awake, Activity) drawn in
    // place of Home while it is pushed.
    var pushedView: AnyView? {
        guard let pushed = state.pushed else { return nil }
        var shown = state
        shown.page = pushed
        return MenuPage.view(shown, handlers: handlers, scrolls: scrolls)
    }

    var key: String { "\(state.openings)/\(state.shown?.id ?? "")/\(state.shown?.context?.machine ?? "")" }

    // value: Home as SPEC A4 draws it, once its engine sends its header.
    var home: Page? {
        guard state.pushed == nil, let shown = state.shown, shown.id == PageID.home, shown.header != nil else { return nil }
        return shown
    }

    @ViewBuilder
    var panel: some View {
        if let home {
            Panel(insets: .page, bounded: scrolls, key: key, sheet: state.confirming) {
                HomeHeaderBar(page: home, state: state, handlers: handlers)
            } content: {
                VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                    if state.menu { menu }
                    HomePage(page: home, state: state, handlers: handlers)
                        .opacity(state.connection == .connecting ? Token.Fade.connecting : 1)
                        .disabled(state.connection == .connecting)
                }
            } footer: {
                if home.section(HomeCards.macs) != nil {
                    VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                        statusBar
                        FindCapsule(open: handlers.openSwitcher)
                    }
                } else {
                    statusBar
                }
            }
            .background { BackKey(state: state, handlers: handlers) }
            .environment(\.stationHints, false)
        } else {
            Panel(insets: .flush, bounded: scrolls, key: key, sheet: state.confirming) {
                page
            } footer: {
                statusLine
            }
            .background { BackKey(state: state, handlers: handlers) }
            .environment(\.stationHints, false)
        }
    }

    @ViewBuilder
    var statusBar: some View {
        if state.connection == .connecting {
            StatusBar("connecting\(Token.Glyph.ellipsis)", failed: false)
        } else if let status = state.status {
            StatusBar(status.line, failed: status.failed)
        }
    }

    var page: some View {
        VStack(alignment: .leading, spacing: Token.Tile.section) {
            header
            if state.menu { menu }
            if let shown = state.shown {
                SwiftUI.Group {
                    if shown.id == PageID.switcher {
                        SwitcherContent(page: shown, state: state, handlers: handlers)
                    } else if shown.id == PageID.machine {
                        MachinePage(page: shown, state: state, handlers: handlers)
                    } else {
                        HomePage(page: shown, state: state, handlers: handlers)
                    }
                }
                .opacity(state.connection == .connecting ? Token.Fade.connecting : 1)
                .disabled(state.connection == .connecting)
            } else {
                PendingDot(still: true)
            }
        }
        .padding(Token.Space.margin)
    }

    // invariant: the status line stays in sight under the panel's scroll,
    // never scrolled away with what it is about, and never drawn over it.
    @ViewBuilder
    var statusLine: some View {
        if state.connection == .connecting {
            StatusBar("connecting\(Token.Glyph.ellipsis)", failed: false).padding(EdgeInsets(top: 0, leading: Token.Space.margin, bottom: Token.Space.margin, trailing: Token.Space.margin))
        } else if let status = state.status {
            StatusBar(status.line, failed: status.failed).padding(EdgeInsets(top: 0, leading: Token.Space.margin, bottom: Token.Space.margin, trailing: Token.Space.margin))
        }
    }

    // invariant: Home and the switcher lead with the machine switcher,
    // This Mac's, its host's name and the settings; a machine's page leads
    // with the way back (its own header).
    @ViewBuilder
    var header: some View {
        if let pushed = state.pushed, pushed.id == PageID.machine {
            EmptyView()
        } else {
            HStack(alignment: .center, spacing: Token.Gap.pair) {
                Button(action: state.pushed == nil ? handlers.openSwitcher : handlers.back) {
                    SwitcherPill(open: state.pushed != nil)
                }
                .buttonStyle(.plain)
                .focusEffectDisabled()
                .keyboardShortcut("0", modifiers: .command)
                if state.pushed == nil {
                    Text(state.page?.context?.machine ?? "station").font(.system(size: Token.Size.sub)).foregroundStyle(Token.Ink.secondary).lineLimit(1).truncationMode(.middle)
                }
                Spacer(minLength: Token.Gap.small)
                if state.pushed == nil {
                    PlainIcon(Token.Symbol.settings, picked: state.menu, round: true, tap: handlers.toggleMenu)
                }
            }
            .padding(.horizontal, Token.Gap.tight).padding(.top, Token.Gap.tight)
        }
    }

    // invariant: the footer menu is drawn inline, never an AppKit menu: the
    // switches for launch at login and the icon's CPU line, and quit.
    var menu: some View {
        HStack(spacing: Token.Space.gap) {
            Toggle("Open at login", isOn: Binding(get: { handlers.loginItemOn }, set: { handlers.loginItem($0) }))
                .toggleStyle(.checkbox)
                .font(.system(size: Token.Space.small))
            Toggle("CPU in the menu bar", isOn: Binding(get: { handlers.sparklineOn }, set: { handlers.sparkline($0) }))
                .toggleStyle(.checkbox)
                .font(.system(size: Token.Space.small))
            Spacer()
            TextAction("Quit") { handlers.quit() }
        }
    }
}

extension Page {
    static let ThisMacWord = "This Mac"
}

// invariant: Escape goes back a level while a page is pushed and no sheet
// is open (a sheet's Cancel takes it then): a machine's page opened from
// Home goes back to Home; on Home it closes the panel, as macOS does.
// Each gesture is its own key, so the one drawn is never the one the
// page before it held.
struct BackKey: View {
    let state: PanelState
    let handlers: PopoverHandlers
    @Environment(\.stationStill) private var still

    enum Gesture: Equatable { case close, back }

    static func gesture(_ state: PanelState) -> Gesture? {
        guard state.confirming == nil else { return nil }
        return state.pushed == nil ? .close : .back
    }

    var body: some View {
        if !still {
            switch Self.gesture(state) {
            case .close: key(handlers.close).id(Gesture.close)
            case .back: key(handlers.back).id(Gesture.back)
            case nil: EmptyView()
            }
        }
    }

    func key(_ act: @escaping () -> Void) -> some View {
        Button("", action: act)
            .keyboardShortcut(.escape, modifiers: [])
            .opacity(0)
            .allowsHitTesting(false)
            .accessibilityHidden(true)
    }
}

// invariant: a pushed page with no field to type in holds the keys
// itself, so Escape reaches its way back before the panel's own close.
struct EscapeKey: View {
    let back: () -> Void
    @FocusState private var focused: Bool

    var body: some View {
        Color.clear.frame(width: Token.Detail.anchor, height: Token.Detail.anchor)
            .focusable()
            .focusEffectDisabled()
            .focused($focused)
            .onKeyPress(.escape) {
                back()
                return .handled
            }
            .onAppear { DispatchQueue.main.async { focused = true } }
            .accessibilityHidden(true)
    }
}
