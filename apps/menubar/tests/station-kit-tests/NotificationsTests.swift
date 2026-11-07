import Foundation
import StationKit
import XCTest

final class NotificationsTests: XCTestCase {
    func until(_ condition: @escaping () -> Bool) async {
        for _ in 0..<500 where !condition() {
            try? await Task.sleep(nanoseconds: 10_000_000)
        }
    }

    // invariant: MB-10: each event of SPEC §10.4 the engine's events watch
    // streams is posted once, in its words; the page watch's notices are
    // never posted.
    func testNotificationsFromEvents() async throws {
        let engine = try FakeEngine("mb-events")
        defer { engine.stop() }
        engine.serveOnce(try Data(contentsOf: Tree.events), thenClose: true)
        let recorder = Recorder()
        let sub = Subscription(path: engine.path, watch: Notifications.watch, clock: TestClock()) { recorder.add($0) }
        let task = Task { await sub.run() }
        let lines = try String(contentsOf: Tree.events, encoding: .utf8).split(separator: "\n").filter { $0.contains(#""event":"notice""#) }.count
        await until { recorder.events.filter { if case .notice = $0 { true } else { false } }.count >= lines }
        task.cancel()
        await task.value
        XCTAssertEqual(engine.received, [#"{"id":"1","v":2,"watch":"engine.events"}"#])
        XCTAssertEqual(recorder.events.compactMap { Notifications.posted($0, from: Notifications.watch) }, [
            "infra stopped and is restarting (↺ 3)",
            "infra is in a loop: 5 restarts in 10 min · waiting 60 s",
            "infra powered off from inside · it stays stopped",
            "infra is back · all 4 health steps pass",
            "Awake stopped on This Mac: battery 20 %",
            "mac-mini: port 5000 is reachable outside Tailscale",
            "This Mac: 3 % of its disk is free",
        ])
    }

    // invariant: MB-10: the eighth event of SPEC §10.4: a call started from
    // the menu bar that fails posts what it was doing and the engine's
    // message, the hint staying on the status line.
    func testAFailedMenuBarCallIsPosted() async throws {
        let engine = try FakeEngine("mb-call-failed")
        defer { engine.stop() }
        let frames = [
            #"{"v":2,"id":"1","progress":{"task":"Installing jq"}}"#,
            #"{"v":2,"id":"1","error":{"type":"refused","code":"apps.needs","message":"brew is not installed","hint":"station apps install brew"}}"#,
        ]
        engine.serveOnce(Data(frames.joined(separator: "\n").utf8) + [0x0A], thenClose: false)
        let install = try JSONDecoder().decode(Action.self, from: Data(#"{"id":"install","label":"Install","run":{"call":"apps.install","in":{"Names":"jq"}}}"#.utf8))
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path).run(install)
        XCTAssertEqual(status?.posted, "Installing jq failed: brew is not installed")
        XCTAssertEqual(status?.line, "brew is not installed  → station apps install brew")
        XCTAssertEqual(status?.failed, true)
    }

    // invariant: MB-16: Update all on Home is the apps.update call; when
    // one app's update fails and the others are done, one notification
    // says which and why, and the status line says it too.
    func testNotificationOnFailedCall() async throws {
        let page = try Tree.page("menubar.home.two-macs")
        let update = try XCTUnwrap(page.actions?.first { $0.id?.hasPrefix(ActionRunner.updateAll) == true })
        XCTAssertEqual(update.label, "Update all")
        XCTAssertEqual(update.run?.call, "apps.update")
        let engine = try FakeEngine("mb-update")
        defer { engine.stop() }
        let frames = [
            #"{"v":2,"id":"1","progress":{"task":"Updating the apps with an update"}}"#,
            #"{"v":2,"id":"1","out":[{"Name":"ghostty","Verb":"update","Result":"ok"},{"Name":"gh","Verb":"update","Result":"broken","Detail":"brew said no"},{"Name":"node","Verb":"update","Result":"ok"}]}"#,
        ]
        engine.serveOnce(Data(frames.joined(separator: "\n").utf8) + [0x0A], thenClose: false)
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path).run(update)
        XCTAssertEqual(engine.received, [#"{"call":"apps.update","id":"1","v":2}"#])
        XCTAssertEqual(status?.posted, "Updating gh failed: brew said no")
        XCTAssertEqual(status?.failed, true)
    }

    func testAPageWatchNoticeIsNeverPosted() throws {
        let page = StreamEvent.notice(kind: "awake_ended", icon: "moon", text: "Awake stopped on This Mac: battery 20 %")
        XCTAssertNil(Notifications.posted(page, from: Subscription.pageWatch))
        XCTAssertNil(Notifications.posted(.notice(kind: "", icon: "", text: "free text"), from: Notifications.watch))
        XCTAssertNil(Notifications.posted(.tick, from: Notifications.watch))
    }
}
