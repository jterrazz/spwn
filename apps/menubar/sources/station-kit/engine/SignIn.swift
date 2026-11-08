import Foundation

// invariant: SPEC B5.19: a verb that opens Terminal on a login and names
// its busy words waits on it: the page it is on is read again every 2 s,
// for at most 5 min, and the wait ends as soon as the page no longer
// offers the verb (the read says signed in).
public enum SignInWatch {
    public static let every: TimeInterval = 2
    public static let most: TimeInterval = 300

    public static func waits(_ action: Action) -> Bool { action.busy != nil && action.run?.terminal == true }

    public static func open(_ id: String, on page: Page?) -> Bool { page?.action(id) != nil }

    // value: the verb's id, now pending, when the verb waits and is not
    // waited on already; else nil, and nothing starts.
    public static func start(_ action: Action, pending: inout [String]) -> String? {
        guard waits(action), let id = action.id, !pending.contains(id) else { return nil }
        pending.append(id)
        return id
    }

    // value: how many times the page was asked for again before the wait
    // ended.
    @discardableResult
    public static func run(since start: Date, now: @Sendable () -> Date, sleep: @Sendable (TimeInterval) async -> Void,
                           open: @Sendable () async -> Bool, refresh: @Sendable () async -> Void) async -> Int {
        var asked = 0
        while await open() {
            await sleep(every)
            guard now().timeIntervalSince(start) < most, await open() else { break }
            await refresh()
            asked += 1
        }
        return asked
    }
}

// invariant: a sign-in waits on the page it was pressed on, read on its
// own watch, never on the page the panel shows: closing the panel, as
// Terminal takes the focus, or going to another page leaves it waiting,
// and a reopened Accounts still reads its busy words until the read says
// signed in.
public final class SignInWait: @unchecked Sendable {
    public let id: String
    public let watch: String
    private let lock = NSLock()
    private var page: Page?

    public init(id: String, watch: String, on page: Page?) {
        self.id = id
        self.watch = watch
        self.page = page
    }

    public func seen(_ page: Page) { lock.withLock { self.page = page } }

    public var open: Bool { lock.withLock { SignInWatch.open(id, on: page) } }

    // value: the engine's watch of the page waited on, each page it streams
    // taken as the wait's own; cancelled once the wait ends.
    public func follow(socket: String, clock: StreamClock = SystemClock()) -> Task<Void, Never> {
        let feed = Subscription(path: socket, pages: [watch], clock: clock) { [weak self] event in
            if case .page(let page) = event { self?.seen(page) }
        }
        return Task.detached { await feed.run() }
    }

    @discardableResult
    public func run(since start: Date, now: @Sendable () -> Date, sleep: @Sendable (TimeInterval) async -> Void,
                    refresh: @Sendable () async -> Void) async -> Int {
        await SignInWatch.run(since: start, now: now, sleep: sleep, open: { self.open }, refresh: refresh)
    }
}
