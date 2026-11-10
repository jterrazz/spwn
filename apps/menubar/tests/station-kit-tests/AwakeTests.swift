import AppKit
@testable import StationKit
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

@MainActor
final class AwakeTermTests: XCTestCase {
    // invariant: the picker draws chosen the term the page says is in
    // force, never Until while a 1 hr term holds, and none while the
    // keep-awake follows a run.
    func testKeepAwakeForDrawsThePickedTerm() {
        var one = Action(), four = Action(), until = Action()
        one.id = "keep:1h"; one.label = "1 hr"; one.picked = true
        four.id = "keep:4h"; four.label = "4 hr"
        until.id = "keep-until"; until.label = "Until 18:00"; until.form = Form()
        let keeps = [one, four, until]
        XCTAssertEqual(keeps.map { AwakePage.picked($0, among: keeps) }, [true, false, false])
        one.picked = nil
        let following = [one, four, until]
        XCTAssertEqual(following.map { AwakePage.picked($0, among: following) }, [false, false, false])
    }

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

    // invariant: a switch flipped after a term was picked sends that
    // term, by its length (1 h) or by its end (Until 17:00), never the
    // page's stale one; Until is held by the end it sent.
    func testASwitchSendsTheTermLastPicked() throws {
        let page = try Tree.page("menubar.awake.two-macs")
        let mac = try XCTUnwrap(page.section("this-mac")?.rows.first?.id)
        let actions = page.actions(for: mac)
        let lid = try XCTUnwrap(actions.first { $0.id == "lid" })
        let one = try XCTUnwrap(actions.first { $0.id == "keep:1h" })
        let until = try XCTUnwrap(actions.first { $0.id == "keep-until" })
        XCTAssertEqual(until.form?.fields?.first?.key, AwakePage.untilKey)
        let byLength = AwakePage.sending(lid, page, held: AwakePage.holding(one, answers: [:]), flipping: true)
        XCTAssertEqual(byLength.run?.in?["For"], "3600000000000")
        XCTAssertNil(byLength.run?.in?["Until"])
        let end = #""2026-11-04T17:00:00Z""#
        let held = AwakePage.holding(until, answers: [AwakePage.untilKey: end])
        XCTAssertEqual(held, [AwakePage.keepKey: "keep-until", AwakePage.untilKey: end])
        let byEnd = AwakePage.sending(lid, page, held: held, flipping: true)
        XCTAssertEqual(byEnd.run?.in?["Until"], end)
        XCTAssertNil(byEnd.run?.in?["For"])
        XCTAssertEqual(byEnd.run?.in?["Lid"], lid.run?.in?["Lid"], "the switch tapped sends its other side")
    }

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
