import AppKit
@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class ActivityTests: XCTestCase {
    func until(_ condition: @escaping () -> Bool) async {
        for _ in 0..<500 where !condition() {
            try? await Task.sleep(nanoseconds: 10_000_000)
        }
    }

    // invariant: AC-25: a machine's Activity page draws every measure as a
    // graph over the window picked, 1 h unless picked, the picker offering
    // 6 min, 1 h and 24 h; picking 6 min makes the engine stream This Mac's
    // activity over 6 min in its place, and runs nothing.
    func testActivityPagePicker() async throws {
        let page = try Tree.page("menubar.activity.mac")
        XCTAssertEqual(ActivityPage.picked(page), "1 h")
        let graphs = try XCTUnwrap(page.section("graphs"))
        XCTAssertEqual(graphs.rows.filter { ($0.cells?.last?.points.count ?? 0) > 1 }.map(\.label),
                       ["CPU", "Memory", "Swap", "Disk read", "Disk write", "Net in", "Net out"])
        XCTAssertEqual(page.section("processes")?.rows.map(\.label), ["Virtualization", "Zed", "WindowServer"])
        XCTAssertEqual(page.actions ?? [], [])

        let engine = try FakeEngine("mb-pick")
        defer { engine.stop() }
        engine.serveOnce(try Streamed.answer("menubar.activity", "menubar.activity.mac"), thenClose: false)
        engine.serveOnce(try Streamed.answer("menubar.activity:6m", "menubar.activity.mac"), thenClose: false)
        let shown = Shown()
        let nav = Navigation(socket: engine.path, clock: TestClock()) { shown.add($0, $1) }
        let handlers = nav.wire(PopoverHandlers())
        handlers.push(Navigation.activity)
        await until { shown.all.count == 1 }
        let picker = ActivityPage.picker(page, handlers: handlers)
        XCTAssertEqual(picker.map(\.words), ["6 min", "1 h", "24 h"])
        XCTAssertEqual(picker.map(\.picked), [false, true, false])
        picker[0].tap()
        await until { shown.all.count == 2 }
        XCTAssertEqual(engine.received, [Streamed.watch("menubar.activity"), Streamed.watch("menubar.activity:6m")])
        XCTAssertEqual(shown.all, ["menubar.activity menubar.activity", "menubar.activity:6m menubar.activity"])
        XCTAssertEqual(nav.watched, "menubar.activity:6m")
        handlers.back()
    }
}

@MainActor
final class ActivityChartTests: XCTestCase {
    func decoded<T: Decodable>(_ json: String) throws -> T { try JSONDecoder().decode(T.self, from: Data(json.utf8)) }

    // invariant: SPEC §10.2 Activity draws every measure with a series as
    // its graph, the vision's line over an area; one not sampled yet keeps
    // its graph's room, waiting, never a table line alone.
    func testActivityDrawsEveryMeasureAsAGraph() throws {
        let page = try Tree.page("menubar.activity.mac")
        let rows = try XCTUnwrap(page.section("graphs")?.rows)
        XCTAssertGreaterThan(rows.count, 4)
        for r in rows where r.id != "disk-free" {
            guard case .graph(let points) = ActivityPage.chart(r) else { return XCTFail("\(r.id ?? "") is not a graph") }
            XCTAssertGreaterThan(points.count, 1)
        }
        let free = try XCTUnwrap(rows.first { $0.id == "disk-free" })
        guard case .meter = ActivityPage.chart(free) else { return XCTFail("disk free is not a meter") }
        let unsampled: Row = try decoded(#"{"id":"cpu","label":"CPU","cells":[{"kind":"text","text":"34 % of 10 CPU"}]}"#)
        guard case .waiting = ActivityPage.chart(unsampled) else { return XCTFail("an unsampled measure has no room") }
    }

    // invariant: Top processes says its memory column in one unit.
    func testTopProcessesSayMemoryInOneUnit() throws {
        let page = try Page.decode(Data(#"{"schema":"station.page/v1","id":"menubar.activity","sections":[{"id":"processes","title":"Top processes","columns":[{"key":"name","kind":"text"},{"key":"what","kind":"text"},{"key":"cpu","kind":"percent"},{"key":"memory","kind":"size"}],"items":[{"row":{"id":"1","label":"tart","cells":[{"kind":"none"},{"kind":"percent","n":180},{"kind":"size","n":4402341478}]}},{"row":{"id":"2","label":"WindowServer","cells":[{"kind":"none"},{"kind":"percent","n":40},{"kind":"size","n":644245094}]}},{"row":{"id":"3","label":"idle","cells":[{"kind":"none"},{"kind":"percent","n":10},{"kind":"none"}]}}]}]}"#.utf8))
        let top = try XCTUnwrap(page.section("processes"))
        XCTAssertEqual(ActivityPage.memory(top), ["1": "4.4 GB", "2": "0.6 GB"])
    }
}
