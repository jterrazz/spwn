@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class Round3Tests: XCTestCase {
    func decoded<T: Decodable>(_ json: String) throws -> T { try JSONDecoder().decode(T.self, from: Data(json.utf8)) }

    func awake() throws -> (Page, [Action]) {
        let page = try Tree.page("menubar.awake.two-macs")
        let mac = try XCTUnwrap(page.section("this-mac")?.rows.first?.id)
        return (page, page.actions(for: mac))
    }

    // invariant: the term the owner picked is drawn chosen before the page
    // read again says it, and a switch flipped meanwhile sends its For,
    // never the stale page's: 1 h then Lid closed keeps 1 h.
    func testAwakeTermPickedIsHeldUntilThePageSaysIt() throws {
        let (page, keeps) = try awake()
        let one = try XCTUnwrap(keeps.first { $0.id == "keep:1h" })
        let four = try XCTUnwrap(keeps.first { $0.id == "keep:4h" })
        let lid = try XCTUnwrap(keeps.first { $0.id == "lid" })
        let held = AwakePage.holding(one, answers: [:])
        XCTAssertEqual(held, [AwakePage.keepKey: "keep:1h", AwakePage.forKey: "3600000000000"])
        XCTAssertTrue(AwakePage.picked(one, among: keeps, held: held))
        XCTAssertFalse(AwakePage.picked(four, among: keeps, held: held))
        XCTAssertNotEqual(lid.run?.in?["For"], "3600000000000")
        let sent = AwakePage.flags(page, held: held, flipping: lid)
        XCTAssertEqual(sent["For"], "3600000000000")
        XCTAssertEqual(AwakePage.settled(held, page), held, "a page that still picks another term keeps the pick held")

        var read = page
        read.actions = read.actions?.map { a in
            var a = a
            if a.id?.hasPrefix("keep") == true { a.picked = a.id == "keep:1h" }
            return a
        }
        XCTAssertEqual(AwakePage.settled(held, read), [:])
        let until = try XCTUnwrap(keeps.first { $0.form != nil })
        XCTAssertEqual(AwakePage.holding(until, answers: ["For": "21600000000000"]), [AwakePage.keepKey: until.id ?? "", AwakePage.forKey: "21600000000000"])
        XCTAssertEqual(AwakePage.holding(lid, answers: [:]), [:])
    }

    // invariant: the until form's options sit four to a row, and its
    // confirm names the hour picked; New VM's confirm is Create.
    func testFormConfirmSaysWhatItDoes() throws {
        let (_, keeps) = try awake()
        let until = try XCTUnwrap(keeps.first { $0.form != nil })
        let form = try XCTUnwrap(until.form)
        let field = try XCTUnwrap(form.fields?.first)
        let last = try XCTUnwrap(field.options?.last)
        XCTAssertEqual(FormSheet.verb(until.label ?? "", form, [field.key ?? "": last.value ?? ""]), "Until \(last.label ?? "")")
        XCTAssertGreaterThan(field.options?.count ?? 0, Choice.perRow)
        let home = try Tree.page("menubar.home.two-macs")
        let new = try XCTUnwrap(home.action("new-vm"))
        XCTAssertEqual(FormSheet.verb(new.label ?? "", try XCTUnwrap(new.form), [:]), "Create")
    }

    // invariant: a port is a number from 1 to 65535: `abc` keeps Share
    // off.
    func testSharePortTakesOnlyAPort() throws {
        let form: StationKit.Form = try decoded(#"{"title":"Share a port","fields":[{"key":"Port","label":"Port","kind":"text","required":true}]}"#)
        XCTAssertFalse(FormSheet.complete(form, ["Port": "abc"]))
        XCTAssertFalse(FormSheet.complete(form, ["Port": "70000"]))
        XCTAssertFalse(FormSheet.complete(form, [:]))
        XCTAssertTrue(FormSheet.complete(form, ["Port": "3000"]))
    }

    // invariant: a port field holding what is not a port says why Share
    // stays off; an empty or valid one says nothing.
    func testSharePortSaysWhatAPortIs() throws {
        let form: StationKit.Form = try decoded(#"{"title":"Share a port","fields":[{"key":"Port","label":"Port","kind":"text","required":true}]}"#)
        let field = try XCTUnwrap(form.fields?.first)
        XCTAssertEqual(FormSheet.refused(field, ["Port": "abc"]), "A port is a number from 1 to 65535.")
        XCTAssertEqual(FormSheet.refused(field, ["Port": "70000"]), "A port is a number from 1 to 65535.")
        XCTAssertNil(FormSheet.refused(field, [:]))
        XCTAssertNil(FormSheet.refused(field, ["Port": "8080"]))
    }

    // invariant: Stop everything that the engine never ends is said, not a
    // spinner for ever.
    func testStopEverythingThatHangsSaysSo() async throws {
        let engine = try FakeEngine("mb-stop-hangs")
        defer { engine.stop() }
        engine.serveOnce(Data(), thenClose: false)
        let stop: Action = try decoded(#"{"id":"stop-all","label":"Stop everything on This Mac","on":"page","run":{"call":"engine.stop"}}"#)
        var runner = ActionRunner(executable: "/u/bin/station", socket: engine.path)
        runner.stopWithin = 0.3
        let status = await runner.run(stop)
        XCTAssertEqual(status?.failed, true)
        XCTAssertTrue(status?.line.hasPrefix("The engine did not stop") == true, status?.line ?? "")
    }

    // invariant: a workspace being stopped says Stopping in its head.
    func testStopSaysStopping() {
        XCTAssertEqual(MachineContent.doing("stop"), "Stopping")
        XCTAssertNil(MachineContent.doing("share-port"))
        XCTAssertNil(MachineContent.doing(nil))
        XCTAssertEqual(MachineContent.subtitle(["Workspace", "on This Mac"]), "Workspace · on\u{00A0}This\u{00A0}Mac")
    }

    // invariant: Home's arrows light a machine line and move through
    // them, never past either end.
    func testHomeArrowsMoveThroughTheMachines() {
        XCTAssertEqual(HomeKeys.moved(nil, by: 1, count: 4), 0)
        XCTAssertEqual(HomeKeys.moved(nil, by: -1, count: 4), 3)
        XCTAssertEqual(HomeKeys.moved(3, by: 1, count: 4), 3)
        XCTAssertEqual(HomeKeys.moved(0, by: -1, count: 4), 0)
        XCTAssertNil(HomeKeys.moved(nil, by: 1, count: 0))
    }

    // invariant: Needs you's folded line is short words: a launchd label
    // by its last part, nothing past a comma.
    func testNeedsYouFoldedLineIsShort() throws {
        let rows: [Row] = try decoded(#"[{"label":"npm sign-in expired"},{"label":"com.jterrazz.brain.x-snapshot is failing, last exit 1"}]"#)
        XCTAssertEqual(HomePage.folded(rows), "npm sign-in expired · x-snapshot is failing")
    }

    // invariant: the engine-stopped page names the Mac as the engine does.
    func testHostNameDropsItsDomain() {
        XCTAssertEqual(Format.bareName("jterrazz-macbook-pro.local"), "jterrazz-macbook-pro")
        XCTAssertEqual(Format.bareName("mini"), "mini")
    }

    // invariant: the panel is made key again after it shows, not only at
    // once: macOS shows its window after the content appears.
    func testPanelIsMadeKeyAfterItShows() {
        XCTAssertGreaterThan(PanelWindow.tries.count, 1)
        XCTAssertGreaterThanOrEqual(PanelWindow.tries.max() ?? 0, 0.5)
    }
}
