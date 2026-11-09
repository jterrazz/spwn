import StationKit
import SwiftUI
import XCTest

final class Said: @unchecked Sendable {
    private let lock = NSLock()
    private var calls: [String] = []
    func add(_ s: String) { lock.withLock { calls.append(s) } }
    var all: [String] { lock.withLock { calls } }
}

@MainActor
final class PlayFixesTests: XCTestCase {
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

    // invariant: memory is said as macOS says it, 16 GB for 16 GiB, the
    // same on Home as on Activity; a disk stays in powers of 1000.
    func ratio(_ n: Int64, _ of: Int64) -> Value {
        var v = Value(kind: "ratio")
        v.n = n
        v.of = of
        return v
    }

    func testMemoryIsSaidInGiBAsMacOSSaysIt() {
        XCTAssertEqual(Format.show(ratio(13_314_398_618, 17_179_869_184)), "12.4/16.0 GB")
        XCTAssertEqual(Format.show(ratio(16_106_127_360, 25_769_803_776)), "15.0/24.0 GB")
        XCTAssertEqual(Format.show(ratio(606_000_000_000, 994_000_000_000)), "606.0/994.0 GB")
    }

    // invariant: a duration is spaced as the vision's: `3 h 57 min`.
    func testDurationsAreSpacedAsTheVisions() {
        XCTAssertEqual(Format.duration(3 * 3600 + 57 * 60), "3 h 57 min")
        XCTAssertEqual(Format.duration(12 * 86400 + 3600), "12 d 1 h")
        XCTAssertEqual(Format.duration(40), "40 s")
    }

    // invariant: M29: a count of any size draws all its digits, never cut
    // to `…`: the item snaps to the width they take.
    func testACountOfAnySizeDrawsAllItsDigits() throws {
        let width = { (n: Int) in MenuBarIcon(.plain, count: n).image(scheme: .dark)?.width ?? 0 }
        XCTAssertGreaterThan(width(1), MenuBarIcon(.plain).image(scheme: .dark)?.width ?? 0)
        XCTAssertGreaterThan(width(10), width(1))
        XCTAssertGreaterThan(width(120), width(10))
    }

    // invariant: the engine stopped keeps a badge: the icon never says
    // all good while nothing answers.
    func testTheEngineStoppedIsNotTheSymbolAlone() {
        let icon = MenuBarIcon.of(connection: .stopped, page: nil)
        XCTAssertEqual(icon.mark, .stopped)
        XCTAssertNotEqual(icon.symbol, Token.Icon.orbit)
    }

    // invariant: Home's tiles open their fleet page.
    func testTheTilesOpenTheirFleetPage() throws {
        let home = try Tree.page("menubar.full")
        XCTAssertEqual(home.section("tiles")?.items?.compactMap(\.tile?.nav), ["menubar.ports", "menubar.accounts", "menubar.awake"])
    }
}

@MainActor
final class AwakeRaceTests: XCTestCase {
    // invariant: Lid closed then On battery before the page is read again
    // ends with both on: the second call carries the first's switch as
    // the owner set it, never the stale page's.
    func testASecondSwitchKeepsTheFirstsNewState() throws {
        let page = try Tree.page("menubar.awake.two-macs")
        let lid = try XCTUnwrap(page.action("lid"))
        let battery = try XCTUnwrap(page.action("battery"))
        let lidOn = lid.run?.in?["Lid"] == "false"
        let first = AwakePage.flags(page, held: [:], flipping: lid)
        XCTAssertEqual(first["Lid"], String(!lidOn))
        let second = AwakePage.flags(page, held: first, flipping: battery)
        XCTAssertEqual(second, ["Lid": String(!lidOn), "Battery": first["Battery"] == "true" ? "false" : "true"])
        XCTAssertEqual(ActionRunner.filled(battery, answers: second)["Lid"], String(!lidOn), "the stale page's Lid is not what is sent")
    }

    // invariant: a held switch lets go once the page says the same.
    func testAHeldSwitchLetsGoOnceThePageSaysIt() throws {
        let page = try Tree.page("menubar.awake.two-macs")
        let lid = try XCTUnwrap(page.action("lid"))
        let paged = lid.run?.in?["Lid"] == "false"
        XCTAssertEqual(AwakePage.settled(["Lid": String(paged)], page), [:])
        XCTAssertEqual(AwakePage.settled(["Lid": String(!paged)], page), ["Lid": String(!paged)])
    }
}
