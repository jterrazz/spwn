import Foundation

public protocol StreamClock: Sendable {
    func sleep(_ seconds: Double) async throws
}

public struct SystemClock: StreamClock {
    public init() {}
    public func sleep(_ seconds: Double) async throws { try await Task.sleep(nanoseconds: UInt64(seconds * 1_000_000_000)) }
}

public struct Reply: Sendable {
    public var ok: Bool
    // value: the connection ended before any answer.
    public var closed: Bool
    public var result: [String: AnySendable]
    public var message: String?
    public var hint: String?
    // value: the task of the call's last progress frame, what it was doing.
    public var task: String? = nil
    // value: the out of a call that answers a list, one entry per item.
    public var listed: [[String: AnySendable]] = []
    // value: no answer came in the time the call was given.
    public var late = false
}

public struct EngineHello: Equatable, Sendable {
    public var executable: String?
    public var page: Int?

    public var older: Bool { (page ?? 0) < pageRevision }
}

public struct AnySendable: @unchecked Sendable {
    public let value: Any
}

public enum StreamEvent: Sendable {
    case connected
    case page(Page)
    case tick
    case sample(LiveSample)
    case notice(kind: String, icon: String, text: String)
    case down
}

public enum Client {
    public static let silence = 45.0
    public static let backoffCap = 30.0

    static func readLine(_ socket: UnixSocket) async throws -> Data {
        try await withCheckedThrowingContinuation { (c: CheckedContinuation<Data, Error>) in
            DispatchQueue.global(qos: .utility).async {
                do { c.resume(returning: try socket.readLine()) } catch { c.resume(throwing: error) }
            }
        }
    }

    // value: the call's input: each answer sets the field its key's path
    // names (`Request.Lifetime`), and an answer written `K=v K=v` sets each of
    // those fields; a value written as JSON (`"5173"`, `["never-sleeps"]`)
    // is sent as that JSON, a number or true/false as one, except under the
    // keys of words, what the owner typed, sent as written.
    public static func input(_ answers: [String: String], fixed: [String: String]? = nil, words: Set<String> = []) -> [String: Any] {
        var out: [String: Any] = [:]
        func set(_ path: [String], _ value: Any, in dict: inout [String: Any]) {
            guard let head = path.first else { return }
            if path.count == 1 {
                dict[head] = value
                return
            }
            var inner = dict[head] as? [String: Any] ?? [:]
            set(Array(path.dropFirst()), value, in: &inner)
            dict[head] = inner
        }
        func put(_ key: String, _ value: String) {
            set(key.split(separator: ".").map(String.init), words.contains(key) ? value : typed(value), in: &out)
        }
        for (k, v) in (fixed ?? [:]).sorted(by: { $0.key < $1.key }) { put(k, v) }
        for (k, v) in answers.sorted(by: { $0.key < $1.key }) {
            if !words.contains(k), v.contains("=") {
                for pair in v.split(separator: " ") {
                    let kv = pair.split(separator: "=", maxSplits: 1).map(String.init)
                    if kv.count == 2 { put(kv[0], kv[1]) }
                }
            } else {
                put(k, v)
            }
        }
        return out
    }

    static func typed(_ value: String) -> Any {
        if let first = value.first, first == "\"" || first == "[" || first == "{",
           let json = try? JSONSerialization.jsonObject(with: Data(value.utf8), options: [.fragmentsAllowed]) {
            return json
        }
        if let n = Int(value) { return n }
        if value == "true" || value == "false" { return value == "true" }
        return value
    }

    // invariant: one call is one station.api/v2 frame on its own connection;
    // its progress frames are read past, and its answer is the out or the
    // error frame that ends it. A connection that ends first is closed.
    // A call given a time ends, unanswered, once it is spent: it is late.
    public static func call(path: String, name: String, input: [String: Any] = [:], target: Target? = nil, within: Double? = nil) async -> Reply {
        var body: [String: Any] = ["v": 2, "id": "1", "call": name]
        if !input.isEmpty { body["in"] = input }
        if let target {
            var t: [String: Any] = [:]
            if let n = target.name { t["name"] = n }
            if let on = target.on { t["on"] = on }
            body["target"] = t
        }
        guard let socket = try? UnixSocket.connect(path), let data = try? JSONSerialization.data(withJSONObject: body, options: [.sortedKeys]) else {
            return Reply(ok: false, closed: false, result: [:], message: "the station engine is not running", hint: "station engine start")
        }
        let spent = Seen()
        if let within {
            DispatchQueue.global(qos: .utility).asyncAfter(deadline: .now() + within) {
                spent.bump()
                socket.shutdownBoth()
            }
        }
        do {
            try socket.write(line: data)
            var task: String?
            while true {
                let line = try await readLine(socket)
                let obj = (try JSONSerialization.jsonObject(with: line)) as? [String: Any] ?? [:]
                if let progress = obj["progress"] {
                    task = ((progress as? [String: Any])?["task"] as? String).flatMap { $0.isEmpty ? nil : $0 } ?? task
                    continue
                }
                if let error = obj["error"] as? [String: Any] {
                    return Reply(ok: false, closed: false, result: [:], message: error["message"] as? String, hint: error["hint"] as? String, task: task)
                }
                let result = (obj["out"] as? [String: Any] ?? [:]).mapValues { AnySendable(value: $0) }
                let listed = (obj["out"] as? [[String: Any]] ?? []).map { $0.mapValues { AnySendable(value: $0) } }
                return Reply(ok: true, closed: false, result: result, message: nil, hint: nil, task: task, listed: listed)
            }
        } catch {
            if spent.value > 0 {
                return Reply(ok: false, closed: false, result: [:], message: "the station engine did not answer in \(Int(within ?? 0)) s", hint: nil, late: true)
            }
            return Reply(ok: false, closed: true, result: [:], message: "the station engine closed the connection", hint: "station engine start")
        }
    }

    // invariant: a watch held is a connection kept open until the task that
    // holds it is cancelled: the engine counts it while it stays open (the
    // popover's `engine.watch`, SPEC §6.3), never longer.
    public static func hold(path: String, watch: String, input: [String: Any] = [:]) async {
        var body: [String: Any] = ["v": 2, "id": "1", "watch": watch]
        if !input.isEmpty { body["in"] = input }
        guard let socket = try? UnixSocket.connect(path), let data = try? JSONSerialization.data(withJSONObject: body, options: [.sortedKeys]),
              (try? socket.write(line: data)) != nil else { return }
        await withTaskCancellationHandler {
                while !Task.isCancelled, (try? await readLine(socket)) != nil {}
        } onCancel: {
            socket.shutdownBoth()
        }
        socket.shutdownBoth()
    }

    public static func hello(path: String) async -> EngineHello? {
        let reply = await call(path: path, name: "engine.hello")
        guard reply.ok else { return nil }
        return EngineHello(executable: reply.result["executable"]?.value as? String, page: reply.result["page"]?.value as? Int)
    }

    public static func executable(path: String) async -> String? {
        await hello(path: path)?.executable
    }
}

final class Seen: @unchecked Sendable {
    private let lock = NSLock()
    private var n = 0
    func bump() { lock.lock(); n += 1; lock.unlock() }
    var value: Int { lock.lock(); defer { lock.unlock() }; return n }
}

public final class Subscription: @unchecked Sendable {
    let path: String
    let watch: String
    let pages: [String]
    let clock: StreamClock
    let emit: @Sendable (StreamEvent) -> Void

    public static let pageWatch = "page"

    public init(path: String, watch: String = Subscription.pageWatch, pages: [String] = ["menubar"], clock: StreamClock = SystemClock(), emit: @escaping @Sendable (StreamEvent) -> Void) {
        self.path = path
        self.watch = watch
        self.pages = pages
        self.clock = clock
        self.emit = emit
    }

    var request: [String: Any] {
        var body: [String: Any] = ["v": 2, "id": "1", "watch": watch]
        if watch == Self.pageWatch { body["in"] = ["pages": pages] }
        return body
    }

    // invariant: one subscribed connection at a time; a connection that ends,
    // or a page watch that says nothing for 45 s (it ticks every 15 s), is
    // dropped and made again after 1, 2, 4 up to 30 s, the wait starting over
    // once a connection streamed again. The events watch is quiet until the
    // engine tells something, so only its end drops it.
    public func run() async {
        var backoff = 1.0
        while !Task.isCancelled {
            if let socket = try? UnixSocket.connect(path), await stream(socket) {
                backoff = 1
            }
            emit(.down)
            do { try await clock.sleep(backoff) } catch { return }
            backoff = min(backoff * 2, Client.backoffCap)
        }
    }

    func stream(_ socket: UnixSocket) async -> Bool {
        guard let request = try? JSONSerialization.data(withJSONObject: request, options: [.sortedKeys]),
              (try? socket.write(line: request)) != nil else { return false }
        let seen = Seen()
        let ticks = watch == Self.pageWatch
        return await withTaskGroup(of: Void.self) { group in
            if ticks {
                group.addTask { [clock] in
                    while true {
                        let before = seen.value
                        do { try await clock.sleep(Client.silence) } catch { return }
                        if seen.value == before {
                            socket.shutdownBoth()
                            return
                        }
                    }
                }
            }
            var streamed = false
            await withTaskCancellationHandler {
                while let line = try? await Client.readLine(socket) {
                    seen.bump()
                    guard let frame = (try? JSONSerialization.jsonObject(with: line)) as? [String: Any] else { continue }
                    if frame["error"] != nil { break }
                    guard let obj = frame["out"] as? [String: Any] else { continue }
                    if obj["stream"] != nil {
                        streamed = true
                        emit(.connected)
                        continue
                    }
                    switch obj["event"] as? String {
                    case "page":
                        if let raw = obj["page"], let data = try? JSONSerialization.data(withJSONObject: raw), let page = try? Page.decode(data) {
                            emit(.page(page))
                        }
                    case "tick":
                        emit(.tick)
                    case "sample":
                        if let sample = LiveSample.decode(obj) { emit(.sample(sample)) }
                    case "notice":
                        let n = obj["notice"] as? [String: Any]
                        emit(.notice(kind: n?["kind"] as? String ?? "", icon: n?["icon"] as? String ?? "", text: n?["text"] as? String ?? ""))
                    default:
                        break
                    }
                }
            } onCancel: {
                socket.shutdownBoth()
            }
            socket.shutdownBoth()
            group.cancelAll()
            return streamed
        }
    }
}
