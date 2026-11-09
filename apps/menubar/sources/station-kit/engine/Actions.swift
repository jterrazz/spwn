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
    // owner typed, which the call checks against the machine's own. An
    // answer the page fixed is the call's own (its yes under a question the
    // alert asks typed): the name typed then only opened the alert's button.
    public static let typedKey = "Confirm"

    public static func filled(_ action: Action, answers: [String: String]) -> [String: String] {
        var out = action.form.map { Answers($0, answers).sent } ?? [:]
        let fixed = action.run?.in?[typedKey] != nil
        if action.confirm?.type != nil, !fixed, let typed = answers[typedKey] { out[typedKey] = typed }
        for (key, value) in answers where action.run?.in?[key] != nil && key != typedKey { out[key] = value }
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
        if !(run.and ?? []).isEmpty || !Self.checks(action).isEmpty { return await chain([run] + (run.and ?? []), of: action, answers: answers) }
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

    // invariant: SPEC A5 Clean up: a sheet's calls are made in turn, each
    // sent only what its In names, narrowed to what the owner left checked;
    // a call left with nothing is not made, and the first refusal stops the
    // rest and is the status.
    func chain(_ runs: [Run], of action: Action, answers: [String: String]) async -> Status? {
        var status: Status?
        for (step, input) in Self.steps(runs, of: action, answers: answers) {
            guard let name = step.call else { continue }
            let reply = await Client.call(path: socket, name: name, input: Client.input([:], fixed: input), target: step.target)
            if !reply.ok {
                let message = reply.message ?? name
                status = Self.failure(message, reply.hint)
                status?.posted = Notifications.failedCall(reply.task ?? action.label ?? name, message)
                break
            }
        }
        if !reread.isEmpty { _ = await Client.call(path: socket, name: Self.refresh, input: ["pages": reread]) }
        return status
    }

    // value: the calls a sheet makes as the owner left it checked, in
    // turn, each with what it is sent; a call left with nothing is not
    // among them.
    public static func steps(_ runs: [Run], of action: Action, answers: [String: String]) -> [(Run, [String: String])] {
        let sent = action.form.map { Answers($0, answers).sent } ?? answers
        let checks = checks(action)
        return runs.compactMap { step in narrowed(step.in ?? [:], by: sent, checks: checks).map { (step, $0) } }
    }

    // value: the keys of a sheet's checkboxes that its calls name: what
    // the owner leaves unchecked there is never sent.
    static func checks(_ action: Action) -> Set<String> {
        let keys = Set(([action.run].compactMap { $0 } + (action.run?.and ?? [])).flatMap { ($0.in ?? [:]).keys })
        let multi = action.form.map { Answers($0).fields.filter { $0.kind == "multiselect" }.compactMap(\.key) } ?? []
        return keys.intersection(multi)
    }

    // value: a call's fixed input narrowed by the answers of the same key:
    // a list keeps what is checked, a single value stays only while it is
    // checked, and a checkbox answer left empty keeps nothing; nil when
    // nothing is left to send.
    public static func narrowed(_ fixed: [String: String], by sent: [String: String], checks: Set<String> = []) -> [String: String]? {
        var out = fixed
        for (key, value) in fixed {
            guard let answer = sent[key] else {
                if checks.contains(key) { return nil }
                continue
            }
            let checked = items(answer) ?? [answer]
            if let list = items(value) {
                let kept = list.filter(checked.contains)
                if kept.isEmpty { return nil }
                out[key] = Answers.list(kept)
            } else if !checked.contains(value) {
                return nil
            }
        }
        return out
    }

    static func items(_ text: String) -> [String]? {
        guard text.first == "[", let data = text.data(using: .utf8) else { return nil }
        return try? JSONSerialization.jsonObject(with: data) as? [String]
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
