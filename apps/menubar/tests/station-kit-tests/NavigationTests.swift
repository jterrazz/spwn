import AppKit
@testable import StationKit
import SwiftUI
import XCTest

final class Shown: @unchecked Sendable {
    private let lock = NSLock()
    private var list: [(String, String?)] = []
    func add(_ watch: String, _ page: Page?) { lock.withLock { list.append((watch, page?.id)) } }
    var all: [String] { lock.withLock { list.map { "\($0.0) \($0.1 ?? "nil")" } } }
}

enum Streamed {
    // value: what the engine answers a page watch: its opening, then the
    // page of a fixture, on one line each.
    static func answer(_ watch: String, _ fixture: String) throws -> Data {
        let page = try JSONSerialization.jsonObject(with: Data(contentsOf: Tree.pages.appendingPathComponent(fixture + ".json")))
        let opened = try JSONSerialization.data(withJSONObject: ["v": 2, "id": "1", "out": ["stream": "page", "pages": [watch]]], options: [.sortedKeys])
        let event = try JSONSerialization.data(withJSONObject: ["v": 2, "id": "1", "out": ["event": "page", "page": page]], options: [.sortedKeys])
        return opened + [0x0A] + event + [0x0A]
    }

    static func watch(_ page: String) -> String { #"{"id":"1","in":{"pages":[""# + page + #""]},"v":2,"watch":"page"}"# }
}

@MainActor
final class NavigationTests: XCTestCase {
    func until(_ condition: @escaping () -> Bool) async {
        for _ in 0..<500 where !condition() {
            try? await Task.sleep(nanoseconds: 10_000_000)
        }
    }

    // invariant: SPEC §10.3, AW-21, PO-34: Home opens Activity, Awake and
    // Ports, each the engine's watch of that page, drawn in place of Home
    // while it is open; back ends the watch and Home is drawn again.
    func testHomeOpensTheMenuBarsOwnPagesFromTheEngine() async throws {
        XCTAssertEqual(Popover.pages.map(\.label), ["Activity", "Awake", "Ports"])
        for (label, fixture) in [("Activity", "menubar.activity.mac"), ("Awake", "menubar.awake.two-macs"), ("Ports", "menubar.ports.mini")] {
            let watch = try XCTUnwrap(Popover.pages.first { $0.label == label }?.watch)
            let engine = try FakeEngine("mb-nav")
            defer { engine.stop() }
            engine.serveOnce(try Streamed.answer(watch, fixture), thenClose: false)
            let shown = Shown()
            let nav = Navigation(socket: engine.path, clock: TestClock()) { shown.add($0, $1) }
            let handlers = nav.wire(PopoverHandlers())
            handlers.push(watch)
            await until { !shown.all.isEmpty }
            XCTAssertEqual(engine.received, [Streamed.watch(watch)], label)
            let page = try Tree.page(fixture)
            XCTAssertEqual(shown.all, ["\(watch) \(page.id ?? "")"], label)
            let home = try Tree.page("menubar.home.two-macs")
            let state = PanelState(page: home, connection: .live, pushed: page)
            XCTAssertEqual(state.shown?.id, page.id)
            XCTAssertNil(try Golden.check(fixture, AnyView(Popover(state, handlers: handlers, scrolls: false)), scheme: .dark), label)
            handlers.back()
            await until { engine.left }
            XCTAssertTrue(engine.left, "\(label): its watch stayed open after back")
            XCTAssertEqual(shown.all.last, "\(watch) nil", label)
            XCTAssertNil(nav.watched)
        }
    }
}

final class Said: @unchecked Sendable {
    private let lock = NSLock()
    private var calls: [String] = []
    func add(_ s: String) { lock.withLock { calls.append(s) } }
    var all: [String] { lock.withLock { calls } }
}

@MainActor
final class NavigationPathTests: XCTestCase {
    // invariant: back goes one level down, to the page it names, never
    // straight to Home: Ports opened from infra goes back to infra.
    func testBackGoesDownOneLevelAndNamesIt() {
        let shown = Said()
        let nav = Navigation(socket: "/nonexistent") { watch, page in shown.add("\(watch)=\(page == nil ? "nil" : "page")") }
        nav.open(PageID.of("infra"))
        nav.open(Navigation.ports)
        XCTAssertEqual(nav.path, [PageID.of("infra"), Navigation.ports])
        XCTAssertEqual(nav.backWords, "infra")
        nav.back()
        XCTAssertEqual(nav.path, [PageID.of("infra")])
        XCTAssertEqual(nav.watched, PageID.of("infra"))
        XCTAssertNil(nav.backWords)
        nav.back()
        XCTAssertEqual(nav.path, [])
        XCTAssertEqual(shown.all.last, "\(PageID.of("infra"))=nil")
    }

    // invariant: Activity's window replaces Activity on top, is kept, and
    // the next visit opens on it.
    func testActivityWindowIsKeptForTheNextVisit() {
        let kept = Said()
        let nav = Navigation(socket: "/nonexistent") { _, _ in }
        nav.remember = { kept.add($0) }
        nav.open(Navigation.activity)
        nav.window("24h")
        nav.window("6m")
        XCTAssertEqual(nav.path, [Navigation.page(Navigation.activity, of: "6m")])
        XCTAssertEqual(kept.all, ["24h", "6m"])
        nav.home()
        XCTAssertEqual(nav.path, [])
        nav.open(Navigation.activity)
        XCTAssertEqual(nav.watched, Navigation.page(Navigation.activity, of: "6m"))
        let fresh = Navigation(socket: "/nonexistent", window: "24h") { _, _ in }
        fresh.open(Navigation.activity)
        XCTAssertEqual(fresh.watched, Navigation.page(Navigation.activity, of: "24h"))
    }

    func testTheWayBackIsWordedByThePageUnderTheTop() {
        XCTAssertEqual(Navigation.title(Navigation.ports, nil), "Ports")
        XCTAssertEqual(Navigation.title(Navigation.page(Navigation.ports, of: "infra"), nil), "Ports")
        XCTAssertEqual(Navigation.title(PageID.of("jterrazz-mac-mini"), try? Tree.page("menubar.mac.mini")), "mac-mini")
    }

    // invariant: a page is swapped once the click that asked for it ended,
    // never under it.
    func testAPageSwapWaitsForTheClickToEnd() {
        var later: [() -> Void] = []
        var opened: [String] = []
        var h = PopoverHandlers()
        h.push = { opened.append($0) }
        h.back = { opened.append("back") }
        let d = h.deferred { later.append($0) }
        d.push(Navigation.awake)
        d.back()
        XCTAssertEqual(opened, [])
        later.forEach { $0() }
        XCTAssertEqual(opened, [Navigation.awake, "back"])
    }

    // invariant: a status line is about the page it was said on: another
    // page shown clears it, the same page read again keeps it.
    func testAStatusLineStaysOnThePageItWasSaidOn() {
        var state = PanelState(page: nil, connection: .live)
        state.say(Status("no machine named infra", failed: true), on: PageID.of("infra"))
        state.showing(PageID.of("infra"))
        XCTAssertNotNil(state.status)
        state.showing(Navigation.page(Navigation.ports, of: "infra"))
        XCTAssertNil(state.status)
        state.say(Status("Update all: done", failed: false), on: nil)
        state.showing(nil)
        XCTAssertNotNil(state.status)
        state.showing(PageID.find)
        XCTAssertNil(state.status)
    }

    // invariant: Home's tiles open their fleet page.
    func testTheTilesOpenTheirFleetPage() throws {
        let home = try Tree.page("menubar.full")
        XCTAssertEqual(home.section("tiles")?.items?.compactMap(\.tile?.nav), ["menubar.ports", "menubar.accounts", "menubar.awake"])
    }
}
