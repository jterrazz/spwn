import Foundation
import StationKit
import XCTest

final class TestClock: StreamClock, @unchecked Sendable {
    private let lock = NSLock()
    private var asked: [Double] = []
    private var waiters: [Int: CheckedContinuation<Void, Error>] = [:]
    private var next = 0

    var backoffs: [Double] { lock.withLock { asked.filter { $0 != Client.silence } } }

    func sleep(_ seconds: Double) async throws {
        lock.withLock { asked.append(seconds) }
        guard seconds == Client.silence else {
            if backoffs.count >= 3 { throw CancellationError() }
            return
        }
        let id = lock.withLock { next += 1; return next }
        try await withTaskCancellationHandler {
            try await withCheckedThrowingContinuation { (c: CheckedContinuation<Void, Error>) in
                lock.withLock { waiters[id] = c }
                if Task.isCancelled { lock.withLock { waiters.removeValue(forKey: id) }?.resume(throwing: CancellationError()) }
            }
        } onCancel: {
            lock.withLock { waiters.removeValue(forKey: id) }?.resume(throwing: CancellationError())
        }
    }

    func releaseSilence() {
        let all = lock.withLock { () -> [CheckedContinuation<Void, Error>] in
            defer { waiters.removeAll() }
            return Array(waiters.values)
        }
        all.forEach { $0.resume() }
    }
}

final class FakeEngine: @unchecked Sendable {
    let path: String
    let fd: Int32
    private let lock = NSLock()
    private var requests: [String] = []
    private var hungUp = false

    init(_ name: String) throws {
        try FileManager.default.createDirectory(atPath: "/private/tmp/station-work", withIntermediateDirectories: true)
        path = "/private/tmp/station-work/\(name)-\(getpid()).sock"
        unlink(path)
        fd = socket(AF_UNIX, SOCK_STREAM, 0)
        var addr = sockaddr_un()
        addr.sun_family = sa_family_t(AF_UNIX)
        let bytes = Array(path.utf8)
        withUnsafeMutableBytes(of: &addr.sun_path) { raw in
            raw.copyBytes(from: bytes)
            raw[bytes.count] = 0
        }
        let bound = withUnsafePointer(to: &addr) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) { bind(fd, $0, socklen_t(MemoryLayout<sockaddr_un>.size)) }
        }
        guard bound == 0, listen(fd, 4) == 0 else { throw SocketError.refused(errno) }
    }

    var received: [String] { lock.withLock { requests } }
    // value: the client closed its connection.
    var left: Bool { lock.withLock { hungUp } }

    // value: accepts one client, reads its request line, answers with these
    // bytes, then stops listening so a reconnect is refused.
    func serveOnce(_ answer: Data, thenClose: Bool, hangUp: Bool = false) {
        Thread.detachNewThread { [self] in
            let client = accept(fd, nil, nil)
            if thenClose {
                close(fd)
                unlink(path)
            }
            var line = Data()
            var byte: UInt8 = 0
            while read(client, &byte, 1) == 1, byte != 0x0A { line.append(byte) }
            lock.withLock { requests.append(String(decoding: line, as: UTF8.self)) }
            _ = answer.withUnsafeBytes { write(client, $0.baseAddress, $0.count) }
            while !hangUp, read(client, &byte, 1) == 1 {}
            lock.withLock { hungUp = true }
            close(client)
        }
    }

    func stop() {
        close(fd)
        unlink(path)
    }
}

final class Recorder: @unchecked Sendable {
    private let lock = NSLock()
    private var list: [StreamEvent] = []
    func add(_ e: StreamEvent) { lock.withLock { list.append(e) } }
    var events: [StreamEvent] { lock.withLock { list } }
}

struct RecordingLauncher: Launcher {
    let calls: Recorded
    func run(_ path: String, _ args: [String], environment: [String: String]) async -> Launched {
        calls.add([path] + args)
        return Launched(code: 0, stderr: "")
    }
}

final class Recorded: @unchecked Sendable {
    private let lock = NSLock()
    private var list: [[String]] = []
    func add(_ c: [String]) { lock.withLock { list.append(c) } }
    var all: [[String]] { lock.withLock { list } }
}

final class ClientTests: XCTestCase {
    func until(_ condition: @escaping () -> Bool) async {
        for _ in 0..<500 where !condition() {
            try? await Task.sleep(nanoseconds: 10_000_000)
        }
    }

    func testTheRecordedStreamIsReadInOrderThenSilenceReconnectsWithBackoff() async throws {
        let engine = try FakeEngine("mb-stream")
        defer { engine.stop() }
        engine.serveOnce(try Data(contentsOf: Tree.stream), thenClose: true)
        let clock = TestClock(), recorder = Recorder()
        let sub = Subscription(path: engine.path, clock: clock) { recorder.add($0) }
        let task = Task { await sub.run() }
        await until { recorder.events.count >= 7 }
        let kinds = recorder.events.prefix(7).map { e -> String in
            switch e {
            case .connected: "connected"
            case .incompatible(let why): "incompatible \(why)"
            case .page(let p): "page \(p.state ?? "")"
            case .tick: "tick"
            case .sample(let s): "sample \(s.at) \(s.values.keys.sorted().joined(separator: " "))\(s.stale.isEmpty ? "" : " stale")"
            case .notice(let kind, let icon, let text): "notice \(kind) \(icon): \(text)"
            case .down: "down"
            }
        }
        let keys = "cpu:mac-mini cpu:macbook-pro memory:macbook-pro net:macbook-pro"
        XCTAssertEqual(kinds, ["connected", "page loading", "page ready", "sample 2026-11-03T10:30:02Z \(keys)", "sample 2026-11-03T10:30:04Z \(keys) stale", "tick",
                               "notice awake_ended moon: Awake stopped on This Mac: battery 20 %"])
        XCTAssertEqual(engine.received, [#"{"id":"1","in":{"pages":["menubar"]},"v":2,"watch":"page"}"#])
        for _ in 0..<200 where clock.backoffs.isEmpty {
            clock.releaseSilence()
            try await Task.sleep(nanoseconds: 10_000_000)
        }
        await task.value
        XCTAssertEqual(clock.backoffs, [1, 2, 4])
    }

    func testARecordedStreamsSamplesReplayThroughTheKitOneValueByKeyTheLateOnesStale() async throws {
        let engine = try FakeEngine("mb-live")
        defer { engine.stop() }
        engine.serveOnce(try Data(contentsOf: Tree.stream), thenClose: true)
        let recorder = Recorder()
        let sub = Subscription(path: engine.path, clock: TestClock()) { recorder.add($0) }
        let task = Task { await sub.run() }
        await until { recorder.events.count >= 7 }
        task.cancel()
        let store = LiveStore()
        var page: Page?
        for event in recorder.events {
            if case .page(let p) = event { page = p }
            if case .sample(let s) = event {
                store.take(s)
                if s.stale.isEmpty {
                    let strip = try XCTUnwrap(page?.strip)
                    XCTAssertEqual([store.now(strip.cpu)?.kind, store.now(strip.cpu)?.n.map(String.init)], ["percent", "450"], "the tick's CPU is not what This Mac's CPU shows")
                    XCTAssertEqual(store.points(strip.cpu, else: strip.spark), [15, 45], "the CPU line is not the tick's series")
                    XCTAssertNotEqual(store.now(strip.cpu), strip.cpu, "the strip shows the page's CPU, not the tick's")
                }
            }
        }
        let strip = try XCTUnwrap(page?.strip)
        XCTAssertEqual(store.now(strip.memory)?.stale, true, "a key told late is not marked stale")
        XCTAssertEqual(store.now(Value(kind: "text", text: "not live")), Value(kind: "text", text: "not live"), "a value without a live key is not shown as the page sent it")
    }

    func testThePopoversWatchIsHeldOpenUntilThePopoverCloses() async throws {
        let engine = try answered("mb-watch", #"{"v":2,"id":"1","out":{"machines":[]}}"#)
        defer { engine.stop() }
        let held = Task { await Client.hold(path: engine.path, watch: "engine.watch") }
        await until { !engine.received.isEmpty }
        try await Task.sleep(nanoseconds: 100_000_000)
        XCTAssertFalse(engine.left, "the popover's watch ended while it was open")
        held.cancel()
        await held.value
        await until { engine.left }
        XCTAssertTrue(engine.left, "the popover closed and its watch stayed open")
        XCTAssertEqual(engine.received, [#"{"id":"1","v":2,"watch":"engine.watch"}"#])
    }

    func fixtureAction(_ id: String) throws -> Action {
        try XCTUnwrap(Tree.page("menubar.full").action(id) ?? Tree.page("menubar.empty").action(id))
    }

    func answered(_ name: String, _ frame: String, hangUp: Bool = false) throws -> FakeEngine {
        let engine = try FakeEngine(name)
        engine.serveOnce(Data(frame.utf8) + (frame.isEmpty ? [] : [0x0A]), thenClose: false, hangUp: hangUp)
        return engine
    }

    func testACallActionIsOneFrameWithItsFormsAnswersAndStartsNoProgram() async throws {
        let engine = try answered("mb-create", #"{"v":2,"id":"1","progress":{"task":"Booting api"}}"# + "\n" + #"{"v":2,"id":"1","out":{"VM":{"Name":"api"}}}"#)
        defer { engine.stop() }
        let calls = Recorded()
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path, launcher: RecordingLauncher(calls: calls)).run(try fixtureAction("new-vm"), answers: ["Request.Lifetime": "persistent"])
        XCTAssertNil(status)
        XCTAssertEqual(engine.received, [#"{"call":"machines.create-workspace","id":"1","in":{"Request":{"Image":"base-ubuntu","Lifetime":"persistent","Name":"linux-2","Network":"nat","Size":{"MemoryBytes":8589934592}}},"v":2}"#])
        XCTAssertEqual(calls.all, [])
    }

    // invariant: ADR-070's fit as a range: an engine is older only below the
    // oldest revision the app accepts, the app older only above the newest it
    // knows, and every revision between fits.
    func testTheVerdictIsARangeFromTheOldestAcceptedToTheNewestKnown() {
        let verdicts = [nil, 2, 3, 4, 5, 6].map { EngineIncompatibility.of(revision: $0, oldest: 3, newest: 5) }
        XCTAssertEqual(verdicts, [.engineOlder, .engineOlder, nil, nil, nil, .appOlder])
        XCTAssertLessThanOrEqual(pageOldest, pageRevision)
    }

    // invariant: the verdict comes in the watch's opening, before any page: an
    // engine at another page revision than the app's has none of its pages drawn.
    func testAWatchOpenedAtAnotherPageRevisionIsJudgedAndNoneOfItsPagesReachTheApp() async throws {
        let fixture = try JSONSerialization.jsonObject(with: Data(contentsOf: Tree.pages.appendingPathComponent("menubar.full.json")))
        let page = String(decoding: try JSONSerialization.data(withJSONObject: ["v": 2, "id": "1", "out": ["event": "page", "page": fixture]], options: [.sortedKeys]), as: UTF8.self)
        let openings: [(String, String, String)] = [
            ("mb-open-now", #""page":\#(pageRevision),"#, "connected page"),
            ("mb-open-old", #""page":\#(pageRevision - 1),"#, "incompatible engineOlder"),
            ("mb-open-v3", "", "incompatible engineOlder"),
            ("mb-open-new", #""page":\#(pageRevision + 1),"#, "incompatible appOlder"),
        ]
        for (name, revision, want) in openings {
            let engine = try answered(name, #"{"v":2,"id":"1","out":{\#(revision)"stream":"station.page/v1","pages":["menubar"]}}"# + "\n" + page + "\n" + page)
            defer { engine.stop() }
            let recorder = Recorder()
            let sub = Subscription(path: engine.path, clock: TestClock()) { recorder.add($0) }
            let task = Task { await sub.run() }
            await until { recorder.events.count >= (want.hasSuffix("page") ? 2 : 1) }
            try await Task.sleep(nanoseconds: 100_000_000)
            task.cancel()
            let seen = recorder.events.compactMap { e -> String? in
                switch e {
                case .connected: "connected"
                case .incompatible(let why): "incompatible \(why)"
                case .page: "page"
                default: nil
                }
            }
            XCTAssertEqual(seen.prefix(2).joined(separator: " "), want, name)
        }
    }

    func testAFailingCallsMessageAndHintMakeTheStatusLine() async throws {
        let engine = try answered("mb-fail", #"{"v":2,"id":"1","error":{"type":"failed","code":"internal","message":"api did not stop","hint":"station machines"}}"#)
        defer { engine.stop() }
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path).run(try fixtureAction("awake-off"))
        XCTAssertEqual(engine.received, [#"{"call":"awake.off","id":"1","v":2}"#])
        XCTAssertEqual(status, Status("api did not stop  → station machines", failed: true, posted: "Keep This Mac awake failed: api did not stop"))
    }

    func testStopEverythingIsDoneWhenTheEngineItStoppedHangsUp() async throws {
        let engine = try answered("mb-stop", "", hangUp: true)
        defer { engine.stop() }
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path).run(try fixtureAction("stop-all"))
        XCTAssertEqual(engine.received, [#"{"call":"engine.stop","id":"1","in":{"By":"the menu bar","Confirm":"yes"},"v":2}"#])
        XCTAssertNil(status)
    }

    func testTheTerminalActionsOpenStationInTheTerminalAndNothingElse() async throws {
        let calls = Recorded()
        let runner = ActionRunner(executable: "/u/bin/station", socket: "/nonexistent", launcher: RecordingLauncher(calls: calls))
        let opened = await runner.run(try fixtureAction("open-station"))
        let started = await runner.run(try XCTUnwrap(Tree.page("menubar.error").action("start")))
        XCTAssertNil(opened)
        XCTAssertNil(started)
        XCTAssertEqual(calls.all, [
            ["/usr/bin/open", "-a", "Terminal", "/u/bin/station"],
            ["/usr/bin/osascript", "-e", #"tell application "Terminal" to do script "'/u/bin/station' 'engine' 'start'""#, "-e", #"tell application "Terminal" to activate"#],
        ])
    }
}
