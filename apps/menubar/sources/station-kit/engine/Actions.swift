import Foundation

public struct Status: Equatable, Sendable {
    public var line: String
    public var failed: Bool
    // value: the notification a failed call started from the menu bar posts
    // (SPEC §10.4); nil for every other outcome.
    public var posted: String?
    public init(_ line: String, failed: Bool, posted: String? = nil) {
        self.line = line
        self.failed = failed
        self.posted = posted
    }
}

public struct ActionRunner: Sendable {
    public var executable: String
    public var socket: String
    public var launcher: Launcher
    public var terminal: String
    // value: the pages watched that a call may have changed, read again once
    // it returns.
    public var reread: [String]
    // value: how long Stop everything waits for the engine to end, in
    // seconds, before it says the engine did not stop: the engine stops
    // its VMs first, each given 60 s to power off and 10 s more to be
    // stopped hard.
    public var stopWithin: Double = 90

    // value: the call that ends the engine answering it: its connection's
    // end, with no answer, is its success.
    public static let endsTheEngine = "engine.stop"
    public static let refresh = "engine.refresh"
    public static let awake = "awake.keep"

    public init(executable: String, socket: String, launcher: Launcher = ProcessLauncher(), terminal: String = "Terminal", reread: [String] = []) {
        self.executable = executable
        self.socket = socket
        self.launcher = launcher
        self.terminal = terminal
        self.reread = reread
    }

    // value: the key a typed confirmation's answer goes under: the name the
    // owner typed, which the call checks against the machine's own.
    public static let typedKey = "Confirm"

    public static func filled(_ action: Action, answers: [String: String]) -> [String: String] {
        var out = action.form.map { Answers($0, answers).sent } ?? [:]
        if action.confirm?.type != nil, let typed = answers[typedKey] { out[typedKey] = typed }
        for (key, value) in answers where action.run?.in?[key] != nil { out[key] = value }
        return out
    }

    // value: the answers the owner typed: a text field's, and a typed
    // confirmation's name.
    static func words(_ action: Action) -> Set<String> {
        var out = Set(action.form.map { Answers($0).fields.filter { $0.kind == "text" }.compactMap(\.key) } ?? [])
        if action.confirm?.type != nil { out.insert(typedKey) }
        return out
    }

    // invariant: the outcome is the status line: nil when the action did what
    // it said, else the engine's own message and hint; a failed call also
    // names the notification it posts. A call goes over the socket, and once
    // it returns, answered or refused, the pages it may have changed are read
    // again; only a terminal action starts a program, the terminal.
    public func run(_ action: Action, answers: [String: String] = [:]) async -> Status? {
        guard let run = action.run else { return nil }
        if let name = run.call {
            let reply = await Client.call(path: socket, name: name, input: Client.input(Self.filled(action, answers: answers), fixed: run.in, words: Self.words(action)), target: run.target,
                                          within: name == Self.endsTheEngine ? stopWithin : nil)
            if reply.late {
                var status = Self.failure("The engine did not stop in \(Int(stopWithin)) s", "station engine stop")
                status.posted = Notifications.failedCall(action.label ?? name, "the engine did not stop")
                return status
            }
            if !reread.isEmpty, !reply.closed {
                _ = await Client.call(path: socket, name: Self.refresh, input: ["pages": reread])
            }
            if reply.ok, let posted = Notifications.failedItems(reply.listed) {
                return Status(posted, failed: true, posted: posted)
            }
            if reply.ok, action.on == "row", reply.task != nil || Self.saysDone(action) { return Self.done(action, name) }
            if reply.ok, let then = run.then, then.terminal == true {
                let said = action.form.map { Answers($0, answers) }
                let words = (then.argv ?? []).map { said?.fill($0) ?? $0 }
                let open = Terminal.opening(terminal, executable: executable, words: words)
                let done = await launcher.run(open.path, open.args, environment: [:])
                return done.code == 0 ? nil : Self.failure("\(terminal) did not open station", nil)
            }
            if reply.ok || (reply.closed && name == Self.endsTheEngine) { return nil }
            let message = reply.message ?? name
            var status = Self.failure(message, reply.hint)
            status.posted = Notifications.failedCall(reply.task ?? action.label ?? name, message)
            return status
        }
        guard run.terminal == true else { return nil }
        let open = Terminal.opening(terminal, executable: executable, words: run.argv ?? [])
        let done = await launcher.run(open.path, open.args, environment: [:])
        return done.code == 0 ? nil : Self.failure("\(terminal) did not open station", nil)
    }

    // value: a call that worked as a task (Update all) says it is done, so
    // a call whose outcome the page does not show is never silent.
    // value: a row's call whose line may stay as it was once it ran (an
    // update the world does not clear at once): it says it is done.
    static func saysDone(_ action: Action) -> Bool { action.id?.hasPrefix(updateAll) == true }
    public static let updateAll = "update-all"

    public static func done(_ action: Action, _ name: String) -> Status {
        Status("\(action.label ?? name): done", failed: false)
    }

    static func failure(_ message: String, _ hint: String?) -> Status {
        Status(hint.map { "\(message)  \(Token.Glyph.arrow) \($0)" } ?? message, failed: true)
    }
}
