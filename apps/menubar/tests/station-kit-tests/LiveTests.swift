@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class LiveTests: XCTestCase {
    static let key = "cpu:jterrazz-macbook-pro"

    static func value(_ json: String) -> Value {
        try! JSONDecoder().decode(Value.self, from: Data(json.utf8))
    }

    static let pageCPU = value(#"{"kind":"percent","n":120,"live":"cpu:jterrazz-macbook-pro"}"#)
    static let tick = value(#"{"kind":"spark","items":[{"kind":"percent","n":450},{"kind":"count","n":15},{"kind":"count","n":45}]}"#)
    static let tickCPU = value(#"{"kind":"percent","n":450}"#)
    static let staleCPU = value(#"{"kind":"percent","n":450,"stale":true}"#)

    static func store(stale: Bool = false) -> LiveStore {
        let store = LiveStore()
        store.take(LiveSample(at: "2026-11-03T10:30:02Z", values: [key: tick], stale: stale ? [key] : []))
        return store
    }

    static func strip(_ cpu: Value, spark: [Int]? = nil) -> Strip {
        var strip = try! JSONDecoder().decode(Strip.self, from: Data("{}".utf8))
        strip.cpu = cpu
        strip.spark = spark
        return strip
    }

    func drawn<V: View>(_ view: V, _ store: LiveStore = LiveStore()) throws -> Bitmap {
        let framed = AnyView(view.frame(width: 320, alignment: .leading).environment(\.stationLive, store))
        return try XCTUnwrap(Snapshot.render(framed, scheme: .dark, desk: false)?.bitmap)
    }

    func same(_ a: Bitmap, _ b: Bitmap) -> Bool {
        a.width == b.width && a.height == b.height && a.differing(b) == 0
    }

    // invariant: SPEC §B3: the strip draws This Mac's CPU and its line
    // from the last tick, never the page's value, so it moves without the
    // page being rebuilt.
    func testTheStripDrawsTheTicksCPUNotThePages() throws {
        let live = try drawn(StripView(Self.strip(Self.pageCPU)), Self.store())
        XCTAssertTrue(same(live, try drawn(StripView(Self.strip(Self.tickCPU, spark: [15, 45])))), "the strip does not draw the tick's CPU and line")
        XCTAssertFalse(same(live, try drawn(StripView(Self.strip(Self.pageCPU)))), "the strip draws the page's CPU over the tick's")
    }

    // invariant: SPEC §B5.15: a key told late keeps its last value, drawn
    // dimmed as stale.
    func testTheStripDimsATickToldLate() throws {
        let late = try drawn(StripView(Self.strip(Self.pageCPU)), Self.store(stale: true))
        XCTAssertTrue(same(late, try drawn(StripView(Self.strip(Self.staleCPU, spark: [15, 45])))), "the strip does not draw the late tick's value dimmed")
        XCTAssertFalse(same(late, try drawn(StripView(Self.strip(Self.tickCPU, spark: [15, 45])))), "a late tick is drawn as a fresh one")
    }

    // invariant: SPEC §B3: Home's and a machine's Activity tiles draw the
    // tick's value and series, the same tick the strip draws.
    func testTheActivityTilesDrawTheTicksValueAndLine() throws {
        let live = try drawn(Metrics(measures: [("CPU", Self.pageCPU, nil)]), Self.store())
        XCTAssertTrue(same(live, try drawn(Metrics(measures: [("CPU", Self.tickCPU, [15, 45])]))), "the tile does not draw the tick's CPU and line")
        XCTAssertFalse(same(live, try drawn(Metrics(measures: [("CPU", Self.pageCPU, nil)]))), "the tile draws the page's CPU over the tick's")
    }

    // invariant: the menu bar's CPU line, when the owner shows it, is the
    // tick's series.
    func testTheMenuBarsCPULineIsTheTicks() throws {
        var page = try Tree.page("menubar.home.one-mac")
        page.strip = Self.strip(Self.pageCPU, spark: [1, 2])
        XCTAssertEqual(MenuBarIcon.of(connection: .live, page: page, sparkline: true, live: Self.store()).spark, [15, 45])
        XCTAssertEqual(MenuBarIcon.of(connection: .live, page: page, sparkline: true, live: LiveStore()).spark, [1, 2])
    }
}
