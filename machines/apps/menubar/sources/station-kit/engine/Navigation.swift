import Foundation

// invariant: SPEC §10.3: a page pushed over Home is the engine's watch of
// that page, held only while it is on top: opening another pushes it over
// the one shown, going back drops the top and watches the one under it
// again. A page opened is drawn at once as it was last seen, across back and
// across the panel's closings (the last `kept` pages seen), then as its
// watch tells it; a page a watch that ended tells is never shown.
// invariant: ADR-069: a page drawn as last seen gives way only to a whole
// page of its watch (`Page.whole`): the pages the watch tells while it is
// still reading are held, the newest drawn once `patience` passed without
// a whole one, so an opening draws the page seen, then at most the page
// read, never a page half read between the two.
public final class Navigation: @unchecked Sendable {
    public static let ports = "menubar.ports"
    public static let awake = "menubar.awake"
    public static let activity = "menubar.activity"
    public static let accounts = "menubar.accounts"

    // value: the watch of a page of one subject: a Mac's ports
    // (`menubar.ports:mac-mini`), the activity over a window
    // (`menubar.activity:6m`).
    public static func page(_ id: String, of subject: String) -> String { "\(id):\(subject)" }

    let socket: String
    let clock: StreamClock
    let shown: @Sendable (String, Page?) -> Void
    private let lock = NSLock()
    private var task: Task<Void, Never>?
    private var stack: [String] = []
    private var seen: [String: Page] = [:]
    private var order: [String] = []
    private var picked: String?
    // value: the watch whose page is drawn as last seen until a whole one
    // comes, the page it told meanwhile, and the count of holds begun, so a
    // patience that ran out ends only its own hold.
    private var holding: String?
    private var held: Page?
    private var holds = 0
    // value: called with Activity's window each time the owner picks one,
    // so the app keeps it for the next visit.
    public var remember: (@Sendable (String) -> Void)?
    // value: called with each tick of the live keys the watched page lists.
    public var sampled: (@Sendable (LiveSample) -> Void)?

    public init(socket: String, clock: StreamClock = SystemClock(), window: String? = nil, shown: @escaping @Sendable (String, Page?) -> Void) {
        self.socket = socket
        self.clock = clock
        self.picked = window
        self.shown = shown
    }

    // value: how many pages are kept as last seen, the least recent dropped.
    public static let kept = 8
    // value: how long a page drawn as last seen waits for a whole page of
    // its watch before the page read so far replaces it: past the slowest
    // whole machine page the bench met on a lone Mac (576 ms).
    public static let patience = 0.75

    public var watched: String? { lock.withLock { stack.last } }

    // value: the page a watch last told, kept the least recent first out.
    private func keep(_ page: Page, as watch: String) {
        seen[watch] = page
        order.removeAll { $0 == watch }
        order.append(watch)
        while order.count > Self.kept { seen[order.removeFirst()] = nil }
    }

    // value: the watches open, Home's first pushed one first.
    public var path: [String] { lock.withLock { stack } }

    // value: the words of the way back: the page under the top, by its
    // title; nil when Home is under it.
    public var backWords: String? {
        lock.withLock { () -> String? in
            guard stack.count > 1 else { return nil }
            let below = stack[stack.count - 2]
            return Self.title(below, seen[below])
        }
    }

    public static func title(_ watch: String, _ page: Page?) -> String {
        let id = watch.split(separator: ":", maxSplits: 1).first.map(String.init) ?? watch
        switch id {
        case ports: return "Ports"
        case awake: return "Awake"
        case activity: return "Activity"
        case accounts: return "Accounts"
        default:
            if let title = page?.header?.title ?? page?.section("machine")?.title, !title.isEmpty { return title }
            return watch.split(separator: ":", maxSplits: 1).dropFirst().first.map(String.init) ?? watch
        }
    }

    // value: Activity opens over the window last picked.
    func windowed(_ page: String) -> String {
        guard page == Self.activity, let picked = lock.withLock({ picked }) else { return page }
        return Self.page(Self.activity, of: picked)
    }

    public func open(_ page: String) {
        let page = windowed(page)
        let last = lock.withLock { () -> Page? in
            if stack.last != page { stack.append(page) }
            return seen[page]
        }
        if let last { hold(page, last) }
        watch(page)
    }

    // value: a page drawn as last seen, held until its watch tells a whole
    // page or `patience` ran out.
    func hold(_ page: String, _ last: Page) {
        let hold = lock.withLock { () -> Int in
            holds += 1
            holding = page
            held = nil
            return holds
        }
        shown(page, last)
        let clock = self.clock
        Task.detached { [weak self] in
            guard (try? await clock.sleep(Navigation.patience)) != nil else { return }
            self?.waited(hold)
        }
    }

    func waited(_ hold: Int) {
        let (page, last) = lock.withLock { () -> (String?, Page?) in
            guard holds == hold, let page = holding else { return (nil, nil) }
            defer { holding = nil; held = nil }
            return (page, held)
        }
        guard let page, let last, watched == page else { return }
        lock.withLock { keep(last, as: page) }
        shown(page, last)
    }

    // value: a page its watch told: drawn, or held while a page seen waits
    // for a whole one.
    func told(_ p: Page, on page: String) {
        let drawn = lock.withLock { () -> Bool in
            guard stack.last == page else { return false }
            if holding == page, !p.whole {
                held = p
                return false
            }
            if holding == page { holding = nil; held = nil }
            keep(p, as: page)
            return true
        }
        if drawn { shown(page, p) }
    }

    // invariant: the window picked replaces Activity on top, never a level
    // more to go back through.
    public func window(_ flag: String) {
        let page = Self.page(Self.activity, of: flag)
        lock.withLock {
            picked = flag
            if let top = stack.last, top.hasPrefix(Self.activity) { stack.removeLast() }
            stack.append(page)
        }
        remember?(flag)
        watch(page)
    }

    func watch(_ page: String) {
        let sub = Subscription(path: socket, pages: [page], clock: clock) { [weak self] event in
            if case .sample(let s) = event, let self, self.watched == page {
                self.sampled?(s)
                return
            }
            guard case .page(let p) = event, let self else { return }
            self.told(p, on: page)
        }
        lock.withLock {
            task?.cancel()
            task = Task.detached { await sub.run() }
        }
    }

    // invariant: back drops the top; the page under it is drawn at once as it
    // was last seen and watched again, or Home is shown.
    public func back() {
        let (was, under, page) = lock.withLock { () -> (String?, String?, Page?) in
            task?.cancel()
            task = nil
            let was = stack.popLast()
            let under = stack.last
            return (was, under, under.flatMap { seen[$0] })
        }
        if let under, let page {
            hold(under, page)
            watch(under)
        } else if let under {
            shown(under, nil)
            watch(under)
        } else if let was {
            shown(was, nil)
        }
    }

    // value: every page dropped at once, its last seen kept: the panel
    // closed shows Home next.
    public func home() {
        let was = lock.withLock { () -> String? in
            task?.cancel()
            task = nil
            holding = nil
            held = nil
            defer { stack = [] }
            return stack.last
        }
        if let was { shown(was, nil) }
    }

    // value: the popover's handlers with its navigation: open a page, pick
    // Activity's window, go back a level.
    public func wire(_ handlers: PopoverHandlers) -> PopoverHandlers {
        var h = handlers
        h.push = { [weak self] in self?.open($0) }
        h.window = { [weak self] in self?.window($0) }
        h.back = { [weak self] in self?.back() }
        h.home = { [weak self] in self?.home() }
        return h
    }
}
