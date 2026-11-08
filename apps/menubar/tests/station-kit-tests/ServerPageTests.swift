import SwiftUI
import XCTest
@testable import StationKit

@MainActor
final class ServerPageTests: XCTestCase {
    // invariant: SPEC A4: a healthy server's Health opens on its head and
    // folds back; a restart's is open by itself and stays open.
    func testHealthUnfoldsOnItsHeadAndARestartsStaysOpen() throws {
        let healthy = try XCTUnwrap(Stories["server.healthy"].state.pushed?.section("health")?.items?.first?.group)
        XCTAssertFalse(HealthCard.shown(healthy, open: false))
        XCTAssertTrue(HealthCard.shown(healthy, open: true))
        let restarting = try XCTUnwrap(Stories["server.restarting"].state.pushed?.section("health")?.items?.first?.group)
        XCTAssertTrue(HealthCard.shown(restarting, open: false))
        XCTAssertEqual(restarting.items?.compactMap(\.row).map(CheckRow.state), [.passed, .running, .next, .next])
    }

    // invariant: SPEC M17: while a restart runs every gauge waits as a
    // skeleton, drawn still with its band at rest, so the golden holds.
    func testTheSkeletonIsDrawnStill() throws {
        let gauges = try XCTUnwrap(Stories["server.restarting"].state.pushed?.section("gauges"))
        XCTAssertTrue(ServerPage.waiting(gauges))
        XCTAssertFalse(ServerPage.waiting(try XCTUnwrap(Stories["server.healthy"].state.pushed?.section("gauges"))))
        let still = AnyView(SkeletonGauges(gauges).environment(\.stationStill, true))
        let first = try XCTUnwrap(Snapshot.render(still, scheme: .dark)?.png)
        let again = try XCTUnwrap(Snapshot.render(still, scheme: .dark)?.png)
        XCTAssertEqual(first, again)
        XCTAssertEqual(Skeleton.band(SkeletonGauges.stillProgress, reduce: false), 0)
    }

    // invariant: SPEC A4, A9 1-10: Open in Terminal waits while the server
    // cannot answer, its reason in place of the ssh.
    func testOpenInTerminalWaitsWhileTheServerCannotAnswer() throws {
        for id in ["server.down", "server.restarting", "server.stopped"] {
            let page = try XCTUnwrap(Stories[id].state.pushed)
            XCTAssertNil(page.footer?.copy, id)
            XCTAssertNotNil(page.footer?.note, id)
            XCTAssertNotNil(page.action("open-station")?.disabled, id)
        }
        let healthy = try XCTUnwrap(Stories["server.healthy"].state.pushed)
        XCTAssertEqual(healthy.footer?.copy?.text, "ssh admin@infra")
        XCTAssertNil(healthy.action("open-station")?.disabled)
    }
}
