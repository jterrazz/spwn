import Foundation

// invariant: SPEC §10.3: a page pushed over Home is the engine's watch of
// that page, held only while it is on top: opening another pushes it over
// the one shown, going back drops the top and watches the one under it
// again, drawn at once as it was last seen; a page of a watch that ended is
// never shown.
public final class Navigation: @unchecked Sendable {
    public static let ports = "menubar.ports"
    public static let awake = "menubar.awake"
    public static let activity = "menubar.activity"

    // value: the watch of a page of one subject: a Mac's ports
    // (`menubar.ports:jterrazz-mac-mini`), the activity over a window
    // (`menubar.activity:6m`).
    public static func page(_ id: String, of subject: String) -> String { "\(id):\(subject)" }

    let socket: String
    let clock: StreamClock
    let shown: @Sendable (String, Page?) -> Void
    private let lock = NSLock()
    private var task: Task<Void, Never>?
    private var stack: [String] = []
    private var seen: [String: Page] = [:]
    private var picked: String?
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

    public var watched: String? { lock.withLock { stack.last } }

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
        case PageID.switcher: return "All machines"
        default:
            if let title = page?.section("machine")?.title, !title.isEmpty { return title }
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
        lock.withLock {
            if stack.last != page { stack.append(page) }
        }
        watch(page)
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
            guard case .page(let p) = event, let self, self.watched == page else { return }
            self.lock.withLock { self.seen[page] = p }
            self.shown(page, p)
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
            if let was { seen[was] = nil }
            let under = stack.last
            return (was, under, under.flatMap { seen[$0] })
        }
        if let under {
            shown(under, page)
            watch(under)
        } else if let was {
            shown(was, nil)
        }
    }

    // value: every page dropped at once: the panel closed shows Home next.
    public func home() {
        let was = lock.withLock { () -> String? in
            task?.cancel()
            task = nil
            defer {
                stack = []
                seen = [:]
            }
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
        return h
    }
}
