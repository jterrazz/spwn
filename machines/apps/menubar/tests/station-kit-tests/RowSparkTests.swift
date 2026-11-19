@testable import StationKit
import XCTest

// invariant: Home V3: a CPU spark ends a row only while its machine runs
// (a running or healthy VM, an online Mac), drawn as it fills from the
// right edge, and it follows the ticks of its live key.
@MainActor
final class RowSparkTests: XCTestCase {
    static let homes = ["home.needs", "home.down", "home.restarting", "home.exposed", "home.fine", "home.loading", "home.empty", "home.awake-off"]
    static let running: Set<String> = ["running", "healthy", "online"]

    func testASparkEndsOnlyARunningRow() throws {
        var sparks = 0, quiet = 0
        for id in Self.homes {
            let page = try XCTUnwrap(Stories[id].state.page, id)
            for kind in [HomePage.servers, HomePage.vms, HomePage.macs] {
                guard let s = page.section(kind) else { continue }
                for row in s.rows where row.id != HomePage.addMac {
                    let word = row.detail?.items?.first?.text ?? row.detail?.text ?? ""
                    let spark = RowView.spark(row, columns: s.columns ?? [])
                    XCTAssertEqual(spark != nil, Self.running.contains(word), "\(id) \(row.id ?? ""): \(word)")
                    if spark == nil { quiet += 1 } else { sparks += 1 }
                }
            }
        }
        XCTAssertGreaterThan(sparks, 0)
        XCTAssertGreaterThan(quiet, 0, "no stopped, asleep or down row was read")
    }

    func testASparkDrawsAsItFillsFromTheRightEdge() {
        let full = Token.Metric.Chart.rowSparkPoints
        XCTAssertEqual(RowSpark.slots(0), [])
        XCTAssertEqual(RowSpark.slots(1), [full - 1])
        XCTAssertEqual(RowSpark.slots(3), [full - 3, full - 2, full - 1])
        XCTAssertEqual(RowSpark.slots(full), Array(0..<full))
        XCTAssertEqual(RowSpark.kept(Array(0..<(full + 4))), Array(4..<(full + 4)))
    }

    func testASparkFollowsItsKeysTicksAndElseThePagesPoints() throws {
        let page = try XCTUnwrap(Stories["home.needs"].state.page)
        let vms = try XCTUnwrap(page.section(HomePage.vms))
        let row = try XCTUnwrap(vms.rows.first { RowView.spark($0, columns: vms.columns ?? []) != nil })
        let cell = try XCTUnwrap(RowView.spark(row, columns: vms.columns ?? []))
        let store = LiveStore()
        XCTAssertEqual(store.points(cell, else: cell.points), cell.points)
        var ticked = Value(kind: "spark")
        var now = Value(kind: "percent")
        now.n = 420
        ticked.items = [now] + [Int64(3), 9, 27].map { n in
            var v = Value(kind: "count")
            v.n = n
            return v
        }
        store.take(LiveSample(at: "", values: [cell.live ?? "": ticked]))
        XCTAssertEqual(store.points(cell, else: cell.points), [3, 9, 27])
        XCTAssertEqual(store.now(RowView.lead(cell))?.n, 420)
    }

    func testTheShortcutsVerbIsASwitchItsPickedAction() throws {
        let page = try XCTUnwrap(Stories["home.needs"].state.page)
        let awake = try XCTUnwrap(page.section(HomePage.shortcuts)?.rows.first { $0.id == "awake" })
        XCTAssertEqual(RowView.action(awake.verb, in: page.actions ?? [])?.picked, true)
        let off = try XCTUnwrap(Stories["home.awake-off"].state.page)
        let keep = try XCTUnwrap(off.section(HomePage.shortcuts)?.rows.first { $0.id == "awake" })
        XCTAssertNotEqual(RowView.action(keep.verb, in: off.actions ?? [])?.picked, true)
    }
}
