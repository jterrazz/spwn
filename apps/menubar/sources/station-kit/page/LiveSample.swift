import Foundation

// value: one tick of the engine's `sample` frame: each live key's value,
// as the page that lists the key draws it, and the keys told late.
public struct LiveSample: Equatable, Sendable {
    public var at: String
    public var values: [String: Value]
    public var stale: [String]

    public init(at: String, values: [String: Value], stale: [String] = []) {
        self.at = at
        self.values = values
        self.stale = stale
    }

    static func decode(_ obj: [String: Any]) -> LiveSample? {
        guard let raw = obj["values"] as? [String: Any] else { return nil }
        var values: [String: Value] = [:]
        for (key, v) in raw {
            if let data = try? JSONSerialization.data(withJSONObject: v), let value = try? JSONDecoder().decode(Value.self, from: data) {
                values[key] = value
            }
        }
        return LiveSample(at: obj["at"] as? String ?? "", values: values, stale: obj["stale"] as? [String] ?? [])
    }
}
