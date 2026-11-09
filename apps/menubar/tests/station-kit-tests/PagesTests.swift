import AppKit
import StationKit
import SwiftUI
import XCTest

// value: one call the menu bar made: the frame the engine received.
struct Sent {
    let frame: [String: Any]
    var call: String? { frame["call"] as? String }
    var input: [String: AnyHashable] { (frame["in"] as? [String: AnyHashable]) ?? [:] }
    var target: [String: String]? { frame["target"] as? [String: String] }
}

@MainActor
final class PagesTests: XCTestCase {
    func texts(_ values: [Value]?) -> [String] { (values ?? []).compactMap(\.text) }

    // value: runs the action as the popover does once its sheet is answered,
    // against an engine that answers ok, and returns the frame it got and the
    // programs started.
    func sent(_ action: Action, answers: [String: String] = [:], label: String) async throws -> (Sent, [[String]]) {
        let (frames, calls) = try await sentAll(action, answers: answers, label: label)
        return (try XCTUnwrap(frames.first), calls)
    }

    // value: every frame the engine got while the action ran on a page
    // watched, which reads that page again once the call returns.
    func sentAll(_ action: Action, answers: [String: String] = [:], label: String, reread: [String] = []) async throws -> ([Sent], [[String]]) {
        let engine = try FakeEngine(label)
        for _ in 0...reread.count { engine.serveOnce(Data(#"{"v":2,"id":"1","out":{}}"#.utf8) + [0x0A], thenClose: false) }
        defer { engine.stop() }
        let calls = Recorded()
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path, launcher: RecordingLauncher(calls: calls), reread: reread).run(action, answers: answers)
        XCTAssertNil(status)
        let frames = try engine.received.map { try XCTUnwrap(JSONSerialization.jsonObject(with: Data($0.utf8)) as? [String: Any]) }
        return (frames.map(Sent.init(frame:)), calls.all)
    }

    func drawn(_ name: String, _ state: PanelState) throws {
        XCTAssertNil(try Golden.check(name, AnyView(Popover(state, scrolls: false))))
    }

    func pushed(_ fixture: String, confirming: String? = nil, typed: String? = nil, segment: String? = nil) throws -> PanelState {
        let page = try Tree.page(fixture)
        var state = Snapshot.state(fixture, page)
        state.confirming = confirming
        state.segment = segment
        if let confirming, let typed { state.answers[confirming] = [ActionRunner.typedKey: typed] }
        return state
    }

    // invariant: MB-07: a workspace's, a server's and a Mac's page each draw
    // their sections and show only actions that name a call, open a
    // terminal, copy or open a link, or show a link's QR, or that only a
    // menu reaches with no call (its header, Open of its row's page), and
    // each draws offscreen.
    func testPagesLayout() throws {
        let linux = try Tree.page("menubar.vm.linux-1")
        XCTAssertEqual(linux.sections?.compactMap(\.id), ["summary", "agents", "gauges", "ports", "footnote", "private", "shared", "window-6m", "window-1h", "window-24h", "memory", "processes", "measured"])
        XCTAssertEqual(linux.section("summary")?.caption.map { $0.compactMap(\.text) }, ["throwaway", "on This Mac", "no agent running"])
        XCTAssertEqual(linux.pageActions.map(\.label), ["Stop", "Delete", "New agent", "Share a port", "Open in Terminal"])
        let infra = try Tree.page("menubar.vm.infra")
        XCTAssertEqual(infra.sections?.compactMap(\.id), ["summary", "health", "gauges", "ports", "tooling", "window-6m", "window-1h", "window-24h", "memory", "processes", "measured"])
        XCTAssertEqual(infra.section("health")?.items?.first?.group?.items?.compactMap(\.row?.label), ["Running", "Has an address", "SSH answers", "Check passes"])
        XCTAssertEqual(infra.pageActions.map(\.label), ["Restart", "Stop", "Delete", "Copy ssh admin@infra", "Open in Terminal"])
        let mini = try Tree.page("menubar.mac.mini")
        XCTAssertEqual(mini.sections?.compactMap(\.id), ["summary", "gauges", "vms", "care", "window-6m", "window-1h", "window-24h", "memory", "processes", "public", "private", "https", "accounts:shared"])
        for page in [linux, infra, mini] {
            for action in page.actions ?? [] {
                XCTAssertTrue(action.run?.call != nil || action.run?.terminal == true || action.submenu != nil || action.run?.copy != nil || action.run?.open != nil || (action.run == nil && action.icon == Token.Symbol.qr) || (action.run == nil && action.on == "menu"), "\(page.context?.machine ?? ""): \(action.id ?? "") names no call")
            }
        }
        for name in ["menubar.vm.linux-1", "menubar.vm.infra", "menubar.mac.mini"] {
            try drawn(name, try pushed(name))
        }
    }

    // invariant: LC-24: Stop on linux-1 asks yes or no and sends yes.
    func testWorkspaceStopSheetYesNo() async throws {
        let stop = try XCTUnwrap(Tree.page("menubar.vm.linux-1").action("stop"))
        let confirm = try XCTUnwrap(stop.confirm)
        XCTAssertNil(confirm.type)
        XCTAssertEqual(confirm.question, "Stop linux-1 on This Mac?")
        let (frame, started) = try await sent(stop, label: "mb-stop-ws")
        XCTAssertEqual(frame.call, "machines.stop")
        XCTAssertEqual(frame.input, ["VM": "linux-1", "Confirm": "yes"])
        XCTAssertNil(frame.target)
        XCTAssertEqual(started, [])
    }

    // invariant: SPEC A5: Delete… on linux-1, a throwaway, asks yes or no;
    // its yes sends the name the delete call checks.
    func testThrowawayDeleteAsksYesOrNoAndSendsItsName() async throws {
        let delete = try XCTUnwrap(Tree.page("menubar.vm.linux-1").action("delete"))
        let confirm = try XCTUnwrap(delete.confirm)
        XCTAssertNil(confirm.type)
        XCTAssertEqual(texts(confirm.lines), ["This can't be undone. It goes with its disk (20 GB), 2 shared ports and 2 accounts lent to it."])
        let (frame, _) = try await sent(delete, label: "mb-delete")
        XCTAssertEqual(frame.call, "machines.delete")
        XCTAssertEqual(frame.input, ["VM": "linux-1", "Confirm": "linux-1"])
    }

    // invariant: LC-24: Stop… on infra is a typed sheet, sent to the Mac
    // that holds infra.
    func testServerStopSheetTyped() async throws {
        let stop = try XCTUnwrap(Tree.page("menubar.vm.infra").action("stop"))
        XCTAssertEqual(stop.confirm?.type, "infra")
        XCTAssertEqual(texts(stop.confirm?.lines).first, "infra stays stopped until you start it.")
        let (frame, _) = try await sent(stop, answers: [ActionRunner.typedKey: "infra"], label: "mb-stop-srv")
        XCTAssertEqual(frame.call, "machines.stop")
        XCTAssertEqual(frame.input, ["VM": "infra", "Confirm": "infra"])
        XCTAssertEqual(frame.target, ["name": "jterrazz-mac-mini"])
    }

    // invariant: SPEC A5: Restart on a healthy infra asks yes or no (an
    // outage), and its yes runs on infra's Mac, the call asking nothing.
    func testHealthyServerRestartAsksYesOrNo() async throws {
        let restart = try XCTUnwrap(Tree.page("menubar.vm.infra").action("restart"))
        XCTAssertEqual(restart.confirm?.question, "Restart infra on mac-mini?")
        XCTAssertNil(restart.confirm?.type)
        XCTAssertNil(restart.form)
        let (frame, started) = try await sent(restart, label: "mb-restart")
        XCTAssertEqual(frame.call, "machines.restart")
        XCTAssertEqual(frame.input, ["Name": "infra"])
        XCTAssertEqual(frame.target, ["name": "jterrazz-mac-mini"])
        XCTAssertEqual(started, [])
    }

    // invariant: PO-45: linux-1's page, opened from Home, is the engine's
    // watch of menubar.machine:linux-1; Share a port… there asks the port as
    // text and sends ports.share for linux-1, and a shared port is a link.
    func testWorkspaceSharePort() async throws {
        let engine = try FakeEngine("mb-share-open")
        defer { engine.stop() }
        let watch = PageID.of("linux-1")
        engine.serveOnce(try Streamed.answer(watch, "menubar.vm.linux-1"), thenClose: false)
        let shown = Shown()
        let nav = Navigation(socket: engine.path, clock: TestClock()) { shown.add($0, $1) }
        nav.open(watch)
        for _ in 0..<500 where shown.all.isEmpty { try await Task.sleep(nanoseconds: 10_000_000) }
        XCTAssertEqual(engine.received, [Streamed.watch("menubar.machine:linux-1")])
        XCTAssertEqual(shown.all, ["menubar.machine:linux-1 menubar.machine"])
        nav.back()

        let page = try Tree.page("menubar.vm.linux-1")
        XCTAssertNil(page.action("declare-public"))
        let share = try XCTUnwrap(page.action("share-port"))
        XCTAssertEqual(share.form?.fields?.map(\.kind), ["text"])
        let (frames, _) = try await sentAll(share, answers: ["Port": "3000"], label: "mb-share", reread: [watch])
        XCTAssertEqual(frames.map(\.call), ["ports.share", "engine.refresh"])
        XCTAssertEqual(frames.first?.input, ["VM": "linux-1", "Port": "3000"])
        XCTAssertTrue(frames.first?.frame["in"].flatMap { ($0 as? [String: Any])?["Port"] } is String)
        XCTAssertEqual(PortRow.link(try XCTUnwrap(page.section("private")?.rows.first), in: page.actions ?? [])?.run?.open, "https://jterrazz-macbook-pro.tail77a797.ts.net:40001")
        try drawn("sheet.share-port", try pushed("menubar.vm.linux-1", confirming: share.id))
    }

    // invariant: AU-15: Take back on linux-1's page asks yes or no, then
    // takes the account back from linux-1.
    func testAccountTakeBack() async throws {
        let page = try Tree.page("menubar.vm.linux-1")
        let shared = try XCTUnwrap(page.section("shared"))
        XCTAssertEqual(shared.title, "Lent to linux-1")
        XCTAssertEqual(shared.items?.compactMap(\.row?.label), ["GitHub", "Claude", "Lend an account"])
        let github = try XCTUnwrap(shared.rows.first { $0.label == "GitHub" }?.id)
        let take = try XCTUnwrap(page.rowActions(github).first { $0.label == "Take back" })
        XCTAssertEqual(take.label, "Take back")
        XCTAssertEqual(take.confirm?.question, "Take GitHub back from linux-1?")
        XCTAssertNil(take.confirm?.type)
        let (frames, _) = try await sentAll(take, label: "mb-take-back", reread: [PageID.of("linux-1")])
        XCTAssertEqual(frames.map(\.call), ["accounts.take-back", "engine.refresh"])
        XCTAssertEqual(frames.first?.input, ["Account": "github", "From": "linux-1", "Confirm": "yes"])
        XCTAssertEqual(frames.last?.input, ["pages": ["menubar.machine:linux-1"]])
    }

    // invariant: MB-15: Open in Terminal on linux-1's page (⌘↩, SPEC A4)
    // runs station on linux-1 in a new terminal, and no call.
    func testOpenInTerminalTargetsMachine() async throws {
        let open = try XCTUnwrap(Tree.page("menubar.vm.linux-1").action("open-station"))
        XCTAssertEqual(open.key, "cmd+enter")
        let calls = Recorded()
        let status = await ActionRunner(executable: "/u/bin/station", socket: "/nonexistent", launcher: RecordingLauncher(calls: calls)).run(open)
        XCTAssertNil(status)
        XCTAssertEqual(calls.all, [
            ["/usr/bin/osascript", "-e", #"tell application "Terminal" to do script "'/u/bin/station' '--on' 'linux-1'""#, "-e", #"tell application "Terminal" to activate"#],
        ])
    }

    // invariant: MC-13: Unlock… on the mini waiting at pre-boot opens
    // Terminal on station machines unlock mac-mini, and nothing else runs.
    func testMacUnlockOpensTerminal() async throws {
        let page = try Tree.page("menubar.mac.mini-offline")
        XCTAssertEqual(page.header?.stale?.items?.last?.text, "checking")
        let unlock = try XCTUnwrap(page.action("unlock"))
        XCTAssertNil(unlock.run?.call)
        try drawn("menubar.mac.mini-offline", try pushed("menubar.mac.mini-offline"))
        let calls = Recorded()
        let status = await ActionRunner(executable: "/u/bin/station", socket: "/nonexistent", launcher: RecordingLauncher(calls: calls)).run(unlock)
        XCTAssertNil(status)
        XCTAssertEqual(calls.all, [
            ["/usr/bin/osascript", "-e", #"tell application "Terminal" to do script "'/u/bin/station' 'machines' 'unlock' 'mac-mini'""#, "-e", #"tell application "Terminal" to activate"#],
        ])
        XCTAssertNil(try Tree.page("menubar.mac.mini").action("unlock"))
    }

    // invariant: MB-14: mac-mini's page, opened from This Mac, is watched as
    // that machine's page and draws what the mini's engine streams.
    func testRemotePageRenders() async throws {
        let page = try Tree.page("menubar.mac.mini")
        XCTAssertEqual(page.context?.machine, "jterrazz-mac-mini")
        XCTAssertEqual(page.section("vms")?.items?.first?.row?.label, "infra")
        XCTAssertEqual(page.section("summary")?.title, "mac-mini")
        let engine = try FakeEngine("mb-remote")
        defer { engine.stop() }
        let frame = try JSONSerialization.data(withJSONObject: ["v": 2, "id": "1", "out": ["event": "page", "page": JSONSerialization.jsonObject(with: page.encoded())]])
        engine.serveOnce(frame + [0x0A], thenClose: true)
        let got = Box()
        let sub = Subscription(path: engine.path, pages: [PageID.of("jterrazz-mac-mini")], clock: TestClock()) { event in
            if case .page(let p) = event { got.set(p) }
        }
        let task = Task.detached { await sub.run() }
        for _ in 0..<500 where got.page == nil { try? await Task.sleep(nanoseconds: 10_000_000) }
        task.cancel()
        XCTAssertEqual(engine.received, [#"{"id":"1","in":{"pages":["menubar.machine:jterrazz-mac-mini"]},"v":2,"watch":"page"}"#])
        XCTAssertEqual(got.page?.section("vms"), page.section("vms"))
        try drawn("menubar.mac.mini", Snapshot.state("menubar.mac.mini", try XCTUnwrap(got.page)))
    }
}

final class Box: @unchecked Sendable {
    private let lock = NSLock()
    private var value: Page?
    var page: Page? { lock.withLock { value } }
    func set(_ p: Page) { lock.withLock { value = p } }
}
