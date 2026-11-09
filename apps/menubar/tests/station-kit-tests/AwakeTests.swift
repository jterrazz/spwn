import AppKit
import StationKit
import SwiftUI
import XCTest

@MainActor
final class AwakeTests: XCTestCase {
    func texts(_ values: [Value]?) -> [String] { (values ?? []).compactMap(\.text) }

    // invariant: AW-21: Keep awake for 4 hr and Lid closed on This Mac are
    // awake.keep calls; Change… on mac-mini, which never sleeps, asks y/n
    // naming infra, and its yes is settings.off never-sleeps run on mac-mini.
    func testAwakeTogglesAndChange() async throws {
        let page = try Tree.page("menubar.awake.two-macs")
        let mac = try XCTUnwrap(page.section("this-mac")?.rows.first?.id)
        let actions = page.actions(for: mac)
        XCTAssertEqual(actions.map(\.label), ["1 hr", "4 hr", "Until 18:00", "Lid closed", "On battery", "Let it sleep"])
        let four = try XCTUnwrap(actions.first { $0.label == "4 hr" })
        XCTAssertEqual(four.run?.call, "awake.keep")
        XCTAssertEqual(Client.input([:], fixed: four.run?.in) as NSDictionary,
                       ["For": 14_400_000_000_000, "Lid": true, "Battery": false, "By": "the menu bar"] as NSDictionary)
        let lid = try XCTUnwrap(actions.first { $0.id == "lid" })
        XCTAssertEqual(lid.run?.call, "awake.keep")
        XCTAssertEqual(lid.run?.in?["Lid"], "false")

        let change = try XCTUnwrap(page.actions(for: "mac-mini").first)
        XCTAssertEqual(change.confirm?.question, "Let mac-mini sleep?")
        XCTAssertEqual(texts(change.confirm?.lines), ["infra pauses while mac-mini sleeps."])
        XCTAssertEqual(change.run?.call, "settings.off")
        XCTAssertEqual(change.run?.target?.name, "mac-mini")
        XCTAssertEqual(Client.input([:], fixed: change.run?.in) as NSDictionary, ["Names": ["never-sleeps"], "Confirm": "yes"] as NSDictionary)

        let engine = try FakeEngine("mb-change")
        defer { engine.stop() }
        engine.serveOnce(Data(#"{"v":2,"id":"1","out":[]}"#.utf8) + [0x0A], thenClose: false)
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path).run(change)
        XCTAssertNil(status)
        XCTAssertEqual(engine.received, [#"{"call":"settings.off","id":"1","in":{"Confirm":"yes","Names":["never-sleeps"]},"target":{"name":"mac-mini"},"v":2}"#])
    }
}
