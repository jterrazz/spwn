import SwiftUI
import XCTest
@testable import StationKit

@MainActor
final class MacPageTests: XCTestCase {
    final class Calls: @unchecked Sendable {
        var made: [String] = []
    }

    func handlers(_ calls: Calls) -> PopoverHandlers {
        var h = PopoverHandlers(run: { a, _ in calls.made.append("run \(a.id ?? "")") }, confirm: { a in calls.made.append("confirm \(a?.id ?? "")") })
        h.push = { calls.made.append("push \($0)") }
        h.openMachine = { calls.made.append("open \($0)") }
        h.window = { calls.made.append("window \($0)") }
        h.back = { calls.made.append("back") }
        return h
    }

    // invariant: SPEC A4, B3: picking a segment on a Mac's page draws its
    // sections from the page the app already holds, and asks the engine
    // nothing: every segment's sections came in that one page.
    func testPickingASegmentDrawsItFromThePageHeldAndAsksNothing() throws {
        let story = Stories["mac.mini.overview"]
        let page = try XCTUnwrap(story.state.pushed)
        let segments = (page.segments ?? []).compactMap(\.id)
        XCTAssertEqual(segments, ["overview", "activity", "ports", "accounts"])
        let calls = Calls()
        var drawn: [Data] = []
        for segment in segments {
            let sections = MacPage.sections(page, segment: segment)
            XCTAssertFalse(sections.isEmpty, "\(segment) has nothing to draw in the page held")
            XCTAssertTrue(sections.allSatisfy { $0.segment == segment }, "\(segment) draws another segment's section")
            var state = story.state
            state.segment = segment
            let view = MacPage(page: page, state: state, handlers: handlers(calls), scrolls: false)
            drawn.append(try XCTUnwrap(Snapshot.render(AnyView(view), scheme: .dark)?.png))
        }
        XCTAssertEqual(Set(drawn).count, segments.count, "two segments draw the same page")
        XCTAssertEqual(calls.made, [])
    }

    // invariant: SPEC A4: Needs you shows its first rows until unfolded,
    // then every row; the fold's words are the page's.
    func testNeedsFoldAfterThePagesTopAndUnfoldInPlace() throws {
        let page = try XCTUnwrap(Stories["mac.this.needs-unfolded"].state.pushed)
        let needs = try XCTUnwrap(page.section("needs-you"))
        XCTAssertEqual(NeedsCard.shown(needs, open: false).count, 3)
        XCTAssertEqual(NeedsCard.shown(needs, open: true).count, 4)
        XCTAssertEqual(needs.rest?.row?.label, "1 more")
        XCTAssertEqual(needs.rest?.row?.detail?.text, "Show less")
    }

    // invariant: SPEC A4, B5.15: a Mac not heard from says when it was
    // seen after its facts, dimmed with them.
    func testAStaleMacSaysWhenItWasSeenAfterItsFacts() throws {
        let page = try XCTUnwrap(Stories["mac.mini.stale"].state.pushed)
        let facts = MachineHeader.facts(page).items ?? []
        XCTAssertEqual(facts.suffix(2).map(\.text), ["seen 3 min ago", "checking"])
        XCTAssertEqual(facts.suffix(2).map(\.signal), ["stale", "stale"])
        XCTAssertNil(MachineHeader.facts(try XCTUnwrap(Stories["mac.mini.overview"].state.pushed)).items?.first { $0.signal == "stale" })
    }
}
