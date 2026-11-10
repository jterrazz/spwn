import AppKit
@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class CallProgressTests: XCTestCase {
    func decoded<T: Decodable>(_ json: String) throws -> T { try JSONDecoder().decode(T.self, from: Data(json.utf8)) }

    // invariant: each update line's Update all spins its own line alone and
    // says it is done once it ran.
    func testUpdateAllSaysDone() {
        var a = Action()
        a.id = "update-all:2-mise"; a.label = "Update all"; a.on = "row"
        XCTAssertTrue(ActionRunner.saysDone(a))
        XCTAssertEqual(ActionRunner.done(a, "apps.update").line, "Update all: done")
    }

    // invariant: Stop everything that the engine never ends is said, not a
    // spinner for ever.
    func testStopEverythingThatHangsSaysSo() async throws {
        let engine = try FakeEngine("mb-stop-hangs")
        defer { engine.stop() }
        engine.serveOnce(Data(), thenClose: false)
        let stop: Action = try decoded(#"{"id":"stop-all","label":"Stop everything on This Mac","on":"page","run":{"call":"engine.stop"}}"#)
        var runner = ActionRunner(executable: "/u/bin/station", socket: engine.path)
        runner.stopWithin = 0.3
        let status = await runner.run(stop)
        XCTAssertEqual(status?.failed, true)
        XCTAssertTrue(status?.line.hasPrefix("The engine did not stop") == true, status?.line ?? "")
    }

    // invariant: Stop everything waits out what the engine's stop does
    // first: each VM's graceful stop (60 s) and its hard stop (10 s).
    func testStopEverythingWaitsOutTheVMsStopping() {
        let runner = ActionRunner(executable: "/u/bin/station", socket: "/nonexistent")
        XCTAssertGreaterThanOrEqual(runner.stopWithin, 70)
    }

    // invariant: a call that takes a while says what it does from the
    // click: New VM says it creates, Stop everything that it stops.
    func testALongCallSaysWhatItDoesAtOnce() throws {
        let page = try Tree.page("menubar.home.two-macs")
        let new = try XCTUnwrap(page.action("new-vm"))
        XCTAssertEqual(HomePage.doing(new), "Creating the VM\(Token.Glyph.ellipsis)")
        let stop = try XCTUnwrap(page.action("stop-all"))
        XCTAssertEqual(HomePage.doing(stop), "Stopping everything\(Token.Glyph.ellipsis)")
    }

    // invariant: a workspace being stopped says Stopping in its head.
    func testStopSaysStopping() {
        XCTAssertEqual(MachineContent.doing("stop"), "Stopping")
        XCTAssertNil(MachineContent.doing("share-port"))
        XCTAssertNil(MachineContent.doing(nil))
        XCTAssertEqual(MachineContent.subtitle(["Workspace", "on This Mac"]), "Workspace · on\u{00A0}This\u{00A0}Mac")
    }

    // invariant: a port being shared shows its row at once, pending, with
    // the port typed, until the page read again lists it.
    func testAPortBeingSharedShowsAtOnce() throws {
        let page = try Tree.page("menubar.vm.linux-1")
        var state = PanelState(page: nil, connection: .live)
        XCTAssertNil(MachineContent.sharing(page, state))
        state.running = "share-port"
        state.sent = ["Port": "3000"]
        XCTAssertEqual(MachineContent.sharing(page, state), "Port 3000")
        state.running = nil
        state.settling = "share-port"
        XCTAssertEqual(MachineContent.sharing(page, state), "Port 3000")
        state.sent = ["Port": "5173"]
        XCTAssertNil(MachineContent.sharing(page, state), "a port the page lists already is its own row")
    }

    // invariant: while a server restarts, its health checks wait for the
    // boot that follows: none is drawn green from the last boot, and the
    // address it checks stays named.
    func testHealthWaitsWhileTheServerRestarts() throws {
        let page = try Page.decode(Data(#"{"schema":"station.page/v1","id":"menubar.machine","sections":[{"id":"health","title":"Health","items":[{"row":{"id":"running","mark":"ok","label":"Running","cells":[{"kind":"none"}]}},{"row":{"id":"address","mark":"ok","label":"Has an address","cells":[{"kind":"text","text":"192.168.64.5"}]}},{"row":{"id":"check","mark":"ok","label":"Check passes","cells":[{"kind":"list","items":[{"kind":"text","text":"https://infra:6443/readyz"},{"kind":"text","text":"200"}]}]}}]}]}"#.utf8))
        let health = try XCTUnwrap(page.section("health"))
        XCTAssertEqual(MachineContent.checks(health, restarting: false), health.rows)
        let waiting = MachineContent.checks(health, restarting: true)
        XCTAssertEqual(waiting.map(\.mark), [.pending, .pending, .pending])
        XCTAssertEqual(waiting.map { $0.cells?.first.map { Format.show($0) } ?? "" }, ["", "", "https://infra:6443/readyz"])
        XCTAssertEqual(MachineContent.doing("restart"), "Restarting")
    }
}
