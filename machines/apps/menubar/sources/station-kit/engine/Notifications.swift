import Foundation

public protocol Poster: Sendable {
    func post(_ text: String)
}

public enum Notifications {
    public static let watch = "engine.events"

    // invariant: SPEC §10.4: a notification is an event of the engine's
    // events watch, which streams the closed list alone, posted in the
    // engine's words, once per event, or a failed call (failedCall); a
    // notice of the page watch draws the status line and is never posted.
    public static func posted(_ event: StreamEvent, from watch: String) -> String? {
        guard watch == Self.watch, case .notice(let kind, _, let text) = event, !kind.isEmpty, !text.isEmpty else { return nil }
        return text
    }

    // invariant: SPEC §10.4's eighth event, a call started from the menu bar
    // failed: what the call was doing, then the engine's message.
    public static func failedCall(_ doing: String, _ message: String) -> String {
        "\(doing) failed: \(message)"
    }

    // value: the result an item of a call's list ends in when it failed.
    public static let failed = "broken"

    // invariant: MB-16: a call that answers its items, some of them failed,
    // posts one notification: the item that failed and why, or how many and
    // which; nil when every item is done.
    public static func failedItems(_ listed: [[String: AnySendable]]) -> String? {
        let failed = listed.filter { $0["Result"]?.value as? String == Self.failed }
        guard let first = failed.first else { return nil }
        let doing = doing(first["Verb"]?.value as? String ?? "")
        let names = failed.compactMap { $0["Name"]?.value as? String }
        if failed.count == 1 {
            return failedCall("\(doing) \(names.first ?? "")", first["Detail"]?.value as? String ?? "")
        }
        return failedCall("\(doing) \(failed.count) apps", names.joined(separator: ", "))
    }

    static func doing(_ verb: String) -> String {
        let stem = verb.hasSuffix("e") ? String(verb.dropLast()) : verb
        return stem.prefix(1).uppercased() + stem.dropFirst() + "ing"
    }
}
