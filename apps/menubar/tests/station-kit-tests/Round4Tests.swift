@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class Round4Tests: XCTestCase {
    func decoded<T: Decodable>(_ json: String) throws -> T { try JSONDecoder().decode(T.self, from: Data(json.utf8)) }

    // invariant: Needs you's folded line is the engine's short words
    // (vision panel a), never its rows' long labels.
    func testNeedsYouFoldsIntoTheEnginesShortWords() throws {
        let page = try Tree.page("menubar.home.two-macs")
        let needs = try XCTUnwrap(page.section(TodoBadge.section))
        XCTAssertEqual(HomePage.folded(needs), ["2 updates", "npm sign-in", "~/.tart", "firewall off", "1 failing task"].joined(separator: Token.Glyph.sep))
        let bare: StationKit.Section = try decoded(#"{"id":"needs-you","items":[{"row":{"label":"npm sign-in expired"}}]}"#)
        XCTAssertEqual(HomePage.folded(bare), "npm sign-in expired")
    }

    // invariant: one duration format and one percent format across the
    // panel: `3 h 58 min`, `12 d`, `18 %`, whatever words the engine sent.
    func testOneDurationAndOnePercentFormat() throws {
        func shown(_ v: Value) -> String { Format.show(v).replacingOccurrences(of: Format.glued, with: " ") }
        XCTAssertEqual(shown(Value(kind: "text", text: "Awake until 14:11 · 3h 59m left")), "Awake until 14:11 · 3 h 59 min left")
        XCTAssertEqual(shown(Value(kind: "text", text: "up 3d 4h")), "up 3 d 4 h")
        XCTAssertEqual(shown(Value(kind: "text", text: "1h left")), "1 h left")
        XCTAssertEqual(shown(Value(kind: "text", text: "rates over 10s")), "rates over 10 s")
        XCTAssertEqual(shown(Value(kind: "text", text: "battery at 18%")), "battery at 18 %")
        XCTAssertEqual(shown(Value(kind: "text", text: "linux-1 · tail1234 · 2026.11.01 · 5173")), "linux-1 · tail1234 · 2026.11.01 · 5173")
        XCTAssertEqual(shown(try decoded(#"{"kind":"percent","n":180}"#) as Value), "18 %")
        XCTAssertEqual(shown(try decoded(#"{"kind":"percent","n":75}"#) as Value), "7.5 %")
        XCTAssertEqual(Format.show(try decoded(#"{"kind":"duration","n":14280}"#) as Value), "3 h 58 min")
        XCTAssertEqual(Format.show(Value(kind: "text", text: "3h 59m left")), ["3", "h", "59", "min", "left"].joined(separator: Format.glued), "a duration is kept whole on its line")
    }

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

    // invariant: a confirm's command is set off as code, never as prose.
    func testAConfirmSetsItsCommandOffAsCode() throws {
        let line: Value = try decoded(#"{"kind":"command","text":"station engine start"}"#)
        XCTAssertEqual(ConfirmSheet.line(line).code, true)
        XCTAssertEqual(ConfirmSheet.line(line).words, "station engine start")
        let prose: Value = try decoded(#"{"kind":"text","text":"linux-1 will stop; its disk is kept"}"#)
        XCTAssertEqual(ConfirmSheet.line(prose).code, false)
    }

    // invariant: a call that takes a while says what it does from the
    // click: New VM says it creates, Stop everything that it stops.
    func testALongCallSaysWhatItDoesAtOnce() throws {
        let page = try Tree.page("menubar.home.two-macs")
        let new = try XCTUnwrap(page.action("new-vm"))
        XCTAssertEqual(HomePage.doing(new), "Creating the VM\(Token.Glyph.ellipsis)")
        let stop = try XCTUnwrap(page.action("stop-all"))
        XCTAssertEqual(HomePage.doing(stop), "Stopping everything\(Token.Glyph.ellipsis)")
    }

    // invariant: a port being shared shows its row at once, pending, with
    // the port typed, until the page read again lists it.
    func testAPortBeingSharedShowsAtOnce() throws {
        let page = try Tree.page("menubar.vm.linux-1")
        var state = PanelState(page: nil, connection: .live)
        XCTAssertNil(MachineContent.sharing(page, state))
        state.running = "share-port"
        state.sent = ["Port": "3000"]
        XCTAssertEqual(MachineContent.sharing(page, state), "Port 3000")
        state.running = nil
        state.settling = "share-port"
        XCTAssertEqual(MachineContent.sharing(page, state), "Port 3000")
        state.sent = ["Port": "5173"]
        XCTAssertNil(MachineContent.sharing(page, state), "a port the page lists already is its own row")
    }

    // invariant: a stopped machine's measures are none: its tiles say no
    // figure while it is stopped or stopping.
    func testAStoppingMachineDrawsNoFigures() throws {
        let page = try Tree.page("menubar.vm.linux-1")
        let activity = try XCTUnwrap(page.section("activity"))
        XCTAssertNotNil(MachineContent.measures(activity, stopping: false).first?.value)
        XCTAssertTrue(MachineContent.measures(activity, stopping: true).allSatisfy { $0.value == nil && $0.spark == nil })
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
}
