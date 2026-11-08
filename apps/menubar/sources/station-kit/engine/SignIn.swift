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
