import SwiftUI

// invariant: the live values the app holds: by key, the last tick's, a key
// told late kept at its last value and marked stale; a page's value with a
// live key shows the store's, never one the app computed.
public final class LiveStore: ObservableObject, @unchecked Sendable {
    @Published public private(set) var values: [String: Value] = [:]

    public init() {}

    public func take(_ sample: LiveSample) {
        var next = values
        for (key, v) in sample.values {
            var v = v
            if sample.stale.contains(key) { v.stale = true }
            next[key] = v
        }
        values = next
    }

    // value: what a page's value shows now: its live key's last value, led
    // out of its series, else the value as the page sent it.
    public func now(_ value: Value?) -> Value? {
        guard let key = value?.live, let got = values[key] else { return value }
        var lead = got.kind == "spark" ? (got.items?.first ?? got) : got
        if got.stale == true { lead.stale = true }
        lead.live = key
        return lead
    }

    // value: the series a value's live key carries, else the page's own.
    public func points(_ value: Value?, else spark: [Int]?) -> [Int]? {
        guard let key = value?.live, let got = values[key], got.kind == "spark" else { return spark }
        return got.points
    }
}

private struct LiveKey: EnvironmentKey {
    static let defaultValue = LiveStore()
}

extension EnvironmentValues {
    public var stationLive: LiveStore {
        get { self[LiveKey.self] }
        set { self[LiveKey.self] = newValue }
    }
}

// value: a view drawn from the live store in the environment, again on each
// tick it takes.
public struct LiveRead<Content: View>: View {
    @Environment(\.stationLive) private var store
    let content: (LiveStore) -> Content

    public init(@ViewBuilder _ content: @escaping (LiveStore) -> Content) { self.content = content }

    public var body: some View { Following(store: store, content: content) }

    struct Following: View {
        @ObservedObject var store: LiveStore
        let content: (LiveStore) -> Content
        var body: some View { content(store) }
    }
}
