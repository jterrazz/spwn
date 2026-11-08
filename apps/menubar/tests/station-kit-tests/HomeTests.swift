import AppKit
import StationKit
import SwiftUI
import XCTest

@MainActor
final class HomeTests: XCTestCase {
    func texts(_ values: [Value]?) -> [String] { (values ?? []).compactMap(\.text) }

    // invariant: EN-10: Stop everything on This Mac asks y/n first, says
    // what stops and that the engine stays stopped, and its yes is the
    // engine.stop call, confirmed, over the socket.
    func testStopEverythingSheet() async throws {
        let page = try Tree.page("menubar.home.two-macs")
        let stop = try XCTUnwrap(page.action("stop-all"))
        let confirm = try XCTUnwrap(stop.confirm)
        XCTAssertEqual(confirm.question, "Stop everything on This Mac?")
        XCTAssertEqual(stop.label, "Stop everything on This Mac")
        XCTAssertEqual(texts(confirm.lines), [
            "linux-1 will stop; its disk is kept",
            "Awake ends; the engine stays off until you log in again or run",
            "station engine start",
        ])
        XCTAssertEqual(confirm.lines?.last?.kind, "command")
        XCTAssertEqual(stop.run?.call, ActionRunner.endsTheEngine)
        XCTAssertEqual(stop.run?.in, ["Confirm": "yes", "By": "the menu bar"])
        var state = Snapshot.state("menubar.home.two-macs", page)
        state.confirming = stop.id
        XCTAssertNil(try Golden.check("sheet.stop-everything", AnyView(Popover(state, scrolls: false))))
    }

    // invariant: AC-25: Home draws CPU, Memory and Network, each with its
    // sparkline and its value.
    func testHomeThreeMeasures() throws {
        let page = try Tree.page("menubar.home.two-macs")
        let strip = try XCTUnwrap(page.strip)
        XCTAssertGreaterThan(strip.spark?.count ?? 0, 1)
        XCTAssertGreaterThan(strip.memory_spark?.count ?? 0, 1)
        XCTAssertGreaterThan(strip.net_spark?.count ?? 0, 1)
        XCTAssertEqual(strip.net?.count, 1)
        XCTAssertNotNil(strip.cpu)
        XCTAssertNotNil(strip.memory)
    }

    // invariant: MB-11: the engine stopped, Home says it, with Start the
    // engine, which opens Terminal on station engine start and nothing else.
    func testEngineStoppedHome() async throws {
        let page = try Tree.page("menubar.home.engine-stopped")
        let row = try XCTUnwrap(page.section("needs-you")?.items?.first?.row)
        XCTAssertEqual(row.label, "engine")
        XCTAssertEqual(texts(row.cells), ["stopped"])
        XCTAssertEqual(page.pageActions.map(\.label), ["Start the engine", "Open in Terminal"])
        let calls = Recorded()
        let runner = ActionRunner(executable: "/u/bin/station", socket: "/nonexistent", launcher: RecordingLauncher(calls: calls))
        let status = await runner.run(try XCTUnwrap(page.action("start")))
        XCTAssertNil(status)
        XCTAssertEqual(calls.all, [
            ["/usr/bin/osascript", "-e", #"tell application "Terminal" to do script "'/u/bin/station' 'engine' 'start'""#, "-e", #"tell application "Terminal" to activate"#],
        ])
        XCTAssertEqual(Snapshot.state("menubar.home.engine-stopped", page).connection, .stopped)
    }

    func drawn(_ name: String, _ state: PanelState) throws {
        XCTAssertNil(try Golden.check(name, AnyView(Popover(state, scrolls: false))))
    }

    // invariant: MB-05, SPEC A4: Home is its hero first, then its tiles,
    // its servers and its Macs; Open in Terminal and Stop everything on
    // This Mac are the station menu's; New VM is the header's.
    func testHomeLayout() throws {
        let page = try Tree.page("menubar.home.two-macs")
        XCTAssertEqual(page.sections?.compactMap(\.id), ["hero", "tiles", "servers", "macs", "needs-you", "machines"])
        XCTAssertEqual(page.header?.buttons, ["new-vm"])
        let strip = try XCTUnwrap(page.strip)
        XCTAssertNotNil(strip.cpu)
        XCTAssertNotNil(strip.memory)
        XCTAssertNotNil(strip.net?.first)
        XCTAssertEqual([strip.spark?.count, strip.memory_spark?.count, strip.net_spark?.count], [16, 16, 16])
        XCTAssertEqual(page.section("needs-you")?.caption?.first?.text, "5 things need you")
        XCTAssertEqual(page.pageActions.map(\.label), ["New VM", "Open in Terminal", "Stop everything on This Mac"])
        XCTAssertEqual(page.pageActions.map(\.key), ["cmd+n", "cmd+t", nil])
        XCTAssertEqual(page.section("tiles")?.items?.compactMap(\.tile).map(TileStrip.said), ["Ports, 2 shared", "Accounts, 1 signed out", "Awake, 6 hr left"])
        try drawn("menubar.home.two-macs", Snapshot.state("menubar.home.two-macs", page))
    }

    // invariant: MA-14: under each Mac its Servers, then its Workspaces,
    // each group named; infra carries its Server kind and a healthy mark.
    func testHomeMachinesGrouped() throws {
        let tree = try XCTUnwrap(Tree.page("menubar.home.two-macs").section("machines"))
        let macs = (tree.items ?? []).compactMap(\.group)
        XCTAssertEqual(macs.compactMap(\.head?.label), ["This Mac", "mac-mini"])
        func groups(_ g: StationKit.Group) -> [String] {
            var out: [String] = []
            for item in g.items ?? [] {
                guard let child = item.group else { continue }
                let names: [String] = (child.items ?? []).compactMap { $0.row?.label }
                out.append((child.head?.label ?? "") + ": " + names.joined(separator: ", "))
            }
            return out
        }
        XCTAssertEqual(groups(macs[0]), ["Workspaces: linux-1, api"])
        XCTAssertEqual(groups(macs[1]), ["Servers: infra"])
        let infra = try XCTUnwrap(macs[1].items?.first?.group?.items?.first?.row)
        XCTAssertEqual(infra.cells?.first?.text, "Server")
        XCTAssertEqual(infra.mark, .ok)
        XCTAssertEqual(texts(infra.cells?.last?.items).prefix(2), ["Always on", "healthy"])
        XCTAssertEqual(texts(macs[0].items?.first?.group?.items?.last?.row?.cells?.last?.items), ["Permanent", "Stopped"])
    }

    // invariant: AP-23: Needs you's `2 app updates` carries Update all,
    // which runs apps.update on This Mac, for every update there, then reads
    // Home again so the line leaves.
    func testNeedsYouUpdateAll() async throws {
        let page = try Tree.page("menubar.home.two-macs")
        let row = try XCTUnwrap(page.section("needs-you")?.items?.first?.row)
        XCTAssertEqual(row.label, "2 app updates")
        let update = try XCTUnwrap(page.rowActions(row.id ?? "").first)
        XCTAssertEqual(update.label, "Update all")
        XCTAssertNil(update.confirm)
        let engine = try FakeEngine("mb-update")
        engine.serveOnce(Data(#"{"v":2,"id":"1","out":[]}"#.utf8) + [0x0A], thenClose: false)
        engine.serveOnce(Data(#"{"v":2,"id":"1","out":{}}"#.utf8) + [0x0A], thenClose: false)
        defer { engine.stop() }
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path, reread: [PageID.shown(nil)]).run(update)
        XCTAssertEqual(status?.line, "Update all: done")
        XCTAssertEqual(engine.received, [
            #"{"call":"apps.update","id":"1","v":2}"#,
            #"{"call":"engine.refresh","id":"1","in":{"pages":["menubar"]},"v":2}"#,
        ])
    }

    // invariant: CR-22: New VM… ⌘N opens the create form as a sheet, in the
    // CLI's words, and its Create sends machines.create-workspace.
    func testNewVMSheetCreates() async throws {
        let page = try Tree.page("menubar.home.two-macs")
        let create = try XCTUnwrap(page.action("new-vm"))
        XCTAssertEqual(create.form?.fields?.compactMap(\.label), ["Will it last?", "Memory"])
        var state = Snapshot.state("menubar.home.two-macs", page)
        state.confirming = create.id
        try drawn("sheet.new-vm", state)
        let engine = try FakeEngine("mb-new-vm")
        engine.serveOnce(Data(#"{"v":2,"id":"1","out":{"VM":{"Name":"linux-2"}}}"#.utf8) + [0x0A], thenClose: false)
        defer { engine.stop() }
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path).run(create, answers: ["Request.Lifetime": "throwaway"])
        XCTAssertNil(status)
        XCTAssertEqual(engine.received, [#"{"call":"machines.create-workspace","id":"1","in":{"Request":{"Image":"linux","Lifetime":"throwaway","Network":"nat","Size":{"MemoryBytes":4294967296}}},"v":2}"#])
    }

    // invariant: MB-12: every snapshot, of a page, a sheet or the icon, is
    // drawn offscreen: rendering them all opens no window.
    func testSnapshotsOffscreen() throws {
        let before = NSApplication.shared.windows.count
        var drawn = 0
        for (name, data) in try Tree.fixtures() {
            let page = try Page.decode(data)
            if Snapshot.render(Snapshot.view(name, page), scheme: .dark) != nil { drawn += 1 }
            if MenuBarIcon.of(connection: .live, page: page).image(scheme: .dark) != nil { drawn += 1 }
        }
        XCTAssertEqual(drawn, try Tree.fixtures().count * 2)
        XCTAssertEqual(NSApplication.shared.windows.count, before)
        XCTAssertEqual(before, 0)
    }
}
