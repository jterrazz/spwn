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
        let opened = try JSONSerialization.data(withJSONObject: ["v": 2, "id": "1", "out": ["stream": "page", "pages": [watch], "page": pageRevision]], options: [.sortedKeys])
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
            XCTAssertNotNil(Snapshot.render(AnyView(Popover(state, handlers: handlers, scrolls: false)), scheme: .dark), label)
            handlers.back()
            await until { engine.left }
            XCTAssertTrue(engine.left, "\(label): its watch stayed open after back")
            XCTAssertEqual(shown.all.last, "\(watch) nil", label)
            XCTAssertNil(nav.watched)
        }
    }

    // invariant: ADR-069: a page seen is drawn at once when it is opened
    // again, after back and after the panel closed, before its watch tells it.
    func testAPageSeenIsDrawnAtOnceWhenOpenedAgainAfterBackAndAClose() async throws {
        let engine = try FakeEngine("mb-seen")
        defer { engine.stop() }
        engine.serveOnce(try Streamed.answer(Navigation.ports, "menubar.ports.mini"), thenClose: false)
        let shown = Shown()
        let nav = Navigation(socket: engine.path, clock: TestClock()) { shown.add($0, $1) }
        let id = try XCTUnwrap(Tree.page("menubar.ports.mini").id)
        nav.open(Navigation.ports)
        await until { !shown.all.isEmpty }
        nav.back()
        nav.home()
        XCTAssertEqual(shown.all, ["\(Navigation.ports) \(id)", "\(Navigation.ports) nil"])
        nav.open(Navigation.ports)
        XCTAssertEqual(shown.all.last, "\(Navigation.ports) \(id)", "the page seen waited for its watch")
        XCTAssertEqual(nav.watched, Navigation.ports)
    }

    // invariant: the pages kept as last seen are the last `kept` ones; the
    // least recent is dropped first.
    func testThePagesKeptAreTheLastEightSeen() async throws {
        XCTAssertEqual(Navigation.kept, 8)
        let engine = try FakeEngine("mb-kept")
        defer { engine.stop() }
        let watches = (0 ... Navigation.kept).map { Navigation.page(Navigation.ports, of: "m\($0)") }
        for watch in watches { engine.serveOnce(try Streamed.answer(watch, "menubar.ports.mini"), thenClose: false) }
        let shown = Shown()
        let nav = Navigation(socket: engine.path, clock: TestClock()) { shown.add($0, $1) }
        for watch in watches {
            nav.open(watch)
            await until { shown.all.last?.hasPrefix(watch + " ") == true }
            nav.home()
        }
        let before = shown.all.count
        nav.open(watches[0])
        XCTAssertEqual(shown.all.count, before, "the least recent page was kept past \(Navigation.kept)")
        nav.home()
        nav.open(watches[1])
        XCTAssertEqual(shown.all.last, "\(watches[1]) \(try XCTUnwrap(Tree.page("menubar.ports.mini").id))")
    }
}

// value: a clock whose sleeps never end but Navigation's patience, which
// ends when the test says: a watch's reconnect never comes, and a page
// seen waits for its whole page until `release`.
final class HeldClock: StreamClock, @unchecked Sendable {
    private let lock = NSLock()
    private var waiters: [Int: (seconds: Double, go: CheckedContinuation<Void, Error>)] = [:]
    private var next = 0

    func sleep(_ seconds: Double) async throws {
        let id = lock.withLock { next += 1; return next }
        try await withTaskCancellationHandler {
            try await withCheckedThrowingContinuation { (c: CheckedContinuation<Void, Error>) in
                lock.withLock { waiters[id] = (seconds, c) }
                if Task.isCancelled { lock.withLock { waiters.removeValue(forKey: id) }?.go.resume(throwing: CancellationError()) }
            }
        } onCancel: {
            lock.withLock { waiters.removeValue(forKey: id) }?.go.resume(throwing: CancellationError())
        }
    }

    var patient: Int { lock.withLock { waiters.values.filter { $0.seconds == Navigation.patience }.count } }

    func release() {
        let ready = lock.withLock { () -> [CheckedContinuation<Void, Error>] in
            let ids = waiters.filter { $0.value.seconds == Navigation.patience }.map(\.key)
            return ids.compactMap { waiters.removeValue(forKey: $0)?.go }
        }
        for go in ready { go.resume() }
    }
}

final class Titles: @unchecked Sendable {
    private let lock = NSLock()
    private var list: [String] = []
    func add(_ page: Page?) { lock.withLock { list.append(page?.header?.title ?? "nil") } }
    var all: [String] { lock.withLock { list } }
}

@MainActor
final class KeptPageTests: XCTestCase {
    func until(_ condition: @escaping () -> Bool) async {
        for _ in 0..<500 where !condition() {
            try? await Task.sleep(nanoseconds: 10_000_000)
        }
    }

    static func page(_ title: String, reading: Bool = false, state: String? = nil) throws -> Page {
        let row = reading ? #","items":[{"row":{"id":"reading","label":"reading","mark":"pending"}}]"# : ""
        let state = state.map { #","state":""# + $0 + #"""# } ?? ""
        return try Page.decode(Data((#"{"schema":""# + pageSchema + #"","id":"menubar.machine","header":{"title":""# + title + #""},"sections":[{"id":"summary""# + row + "}]" + state + "}").utf8))
    }

    // invariant: the engine's rule of a full page: a header, no loading
    // state, no value or mark still pending.
    func testAPageIsWholeByTheEnginesRuleOfAFullPage() throws {
        XCTAssertTrue(try Self.page("mac-mini").whole)
        XCTAssertFalse(try Self.page("mac-mini", reading: true).whole, "a mark still pending")
        XCTAssertFalse(try Self.page("mac-mini", state: "loading").whole, "a page loading")
        var value = try Self.page("mac-mini")
        value.sections?[0].caption = [Value(kind: "pending")]
        XCTAssertFalse(value.whole, "a value still pending")
        var bare = try Self.page("mac-mini")
        bare.header = nil
        XCTAssertFalse(bare.whole, "a page without its header")
    }

    // invariant: ADR-069: a page opened again is drawn as last seen, then
    // as the first whole page its watch tells; the pages it tells while
    // still reading are never drawn between the two.
    func testAPageSeenGivesWayOnlyToAWholePageOfItsWatch() async throws {
        let titles = Titles(), clock = HeldClock()
        let nav = Navigation(socket: "/private/tmp/station-work/mb-none-\(getpid()).sock", clock: clock) { _, page in titles.add(page) }
        let watch = PageID.of("mac-mini")
        nav.open(watch)
        nav.told(try Self.page("seen"), on: watch)
        nav.home()
        XCTAssertEqual(titles.all, ["seen", "nil"])
        nav.open(watch)
        XCTAssertEqual(titles.all.last, "seen", "the page seen waited for its watch")
        nav.told(try Self.page("half", reading: true), on: watch)
        XCTAssertEqual(titles.all.last, "seen", "a page still reading replaced the page seen")
        nav.told(try Self.page("read"), on: watch)
        XCTAssertEqual(titles.all, ["seen", "nil", "seen", "read"])
        nav.told(try Self.page("next", reading: true), on: watch)
        XCTAssertEqual(titles.all.last, "next", "a page drawn from its watch held back the next one")
    }

    // invariant: a page seen whose watch tells no whole page gives way to
    // the newest page read once `patience` passed, and a patience that ran
    // out after its page was left draws nothing.
    func testAPageSeenGivesWayToThePageReadOncePatienceRanOut() async throws {
        let titles = Titles(), clock = HeldClock()
        let nav = Navigation(socket: "/private/tmp/station-work/mb-none-\(getpid()).sock", clock: clock) { _, page in titles.add(page) }
        let watch = PageID.of("mac-mini")
        nav.open(watch)
        nav.told(try Self.page("seen"), on: watch)
        nav.home()
        nav.open(watch)
        nav.told(try Self.page("half", reading: true), on: watch)
        await until { clock.patient == 1 }
        clock.release()
        await until { titles.all.last == "half" }
        XCTAssertEqual(titles.all, ["seen", "nil", "seen", "half"])
        nav.home()
        nav.open(watch)
        nav.told(try Self.page("later", reading: true), on: watch)
        await until { clock.patient == 1 }
        nav.home()
        clock.release()
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertEqual(titles.all, ["seen", "nil", "seen", "half", "nil", "half", "nil"])
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
        XCTAssertEqual(Navigation.title(PageID.of("mac-mini"), try? Tree.page("menubar.mac.mini")), "mac-mini")
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

    // invariant: Home's shortcut rows open their fleet page.
    func testTheShortcutsOpenTheirFleetPage() throws {
        let home = try Tree.page("menubar.full")
        XCTAssertEqual(home.section("shortcuts")?.rows.compactMap(\.nav), ["menubar.awake", "menubar.ports", "menubar.accounts"])
    }
}
