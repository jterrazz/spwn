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
        let view = MacPage(page: page, state: story.state, handlers: handlers(calls), scrolls: false)
        let picker = view.header.segmented(page.segments ?? [])
        XCTAssertEqual(picker.map(\.label), ["Overview", "Activity", "Ports", "Accounts"])
        for option in picker { option.tap() }
        XCTAssertEqual(calls.made, [])
    }

    // invariant: SPEC A4.1: Activity opens on the window last picked for
    // that machine, else on the one its page opens on; picking one redraws
    // the page held, remembers it for that machine, and asks nothing.
    func testActivityOpensOnTheWindowLastPickedForThatMachine() throws {
        let story = Stories["mac.mini.activity"]
        let page = try XCTUnwrap(story.state.pushed)
        var state = PanelState(page: nil, connection: .live, pushed: page)
        XCTAssertEqual(ActivityWindow.remembered(state, page), "1h")
        state.windows = ["macbook-pro": "6m"]
        XCTAssertEqual(ActivityWindow.remembered(state, page), "1h", "another Mac's window carried over")
        state.windows["mac-mini"] = "24h"
        XCTAssertEqual(ActivityWindow.remembered(state, page), "24h")
        XCTAssertEqual(Stories["mac.mini.activity.6m"].state.windows, ["mac-mini": "6m"])

        let calls = Calls()
        var h = handlers(calls)
        h.remember = { machine, window in calls.made.append("remember \(window) on \(machine)") }
        MacPage(page: page, state: story.state, handlers: h, scrolls: false).pickWindow("6m")
        XCTAssertEqual(calls.made, ["remember 6m on mac-mini"])

        let sections = MacPage.sections(page, segment: ActivityWindow.segment)
        var drawn: [Data] = []
        for window in ["6m", "1h", "24h"] {
            XCTAssertEqual(MachineActivity.shown(sections, window: window)?.id, "window-\(window)")
            let view = MachineActivity(page: page, sections: sections, window: window, pick: { _ in }, scrub: nil, onScrub: { _ in }, run: { _, _ in }, open: { _ in })
            drawn.append(try XCTUnwrap(Snapshot.render(AnyView(view.frame(width: Token.Metric.Panel.width)), scheme: .dark)?.png))
        }
        XCTAssertEqual(Set(drawn).count, 3, "two windows draw the same charts")
    }

    // invariant: SPEC A4: a running workspace's page, as the live app
    // holds it with no segment picked, is drawn on the frame its segments
    // are picked in; its loans, each with its time left, are one pick away
    // under Accounts and never drawn on its Overview.
    func testARunningWorkspacesLoansAreOnePickAwayUnderAccounts() throws {
        let page = try Tree.page("menubar.vm.linux-1")
        let state = Snapshot.state("menubar.vm.linux-1", page)
        XCTAssertNil(state.segment)
        let calls = Calls()
        XCTAssertNotNil(MachinePage.view(state, handlers: handlers(calls), scrolls: false))
        XCTAssertEqual(page.segments?.map(\.label), ["Overview", "Activity", "Ports", "Accounts"])
        let loans = MacPage.sections(page, segment: "accounts").flatMap(\.rows).filter { $0.left != nil }
        XCTAssertEqual(loans.map(\.label), ["GitHub", "Claude"])
        let overview = MacPage.sections(page, segment: "overview").compactMap(\.id)
        XCTAssertEqual(overview, ["agents", "gauges", "ports", "footnote"])
        var picked = state
        picked.segment = "accounts"
        let drawn = try [state, picked].map { s in
            try XCTUnwrap(Snapshot.render(AnyView(WorkspacePage(page: page, state: s, handlers: handlers(calls), scrolls: false)), scheme: .dark)?.png)
        }
        XCTAssertNotEqual(drawn[0], drawn[1])
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

    // invariant: SPEC A4: mac-mini never sleeps, so its upkeep unfolds to
    // its readiness after Settings, Background, Apps and Disk.
    func testAMacThatNeverSleepsUnfoldsItsUpkeepToItsReadiness() throws {
        let page = try XCTUnwrap(Stories["mac.mini.overview"].state.pushed)
        let upkeep = try XCTUnwrap(page.section("care")?.items?.compactMap(\.group).first { $0.head?.id == "upkeep" })
        XCTAssertEqual(upkeep.head?.cells?.first?.text, "all fine")
        XCTAssertEqual((upkeep.items ?? []).compactMap(\.row?.label), ["Settings", "Background", "Apps", "Disk", "Ready to host servers"])
        XCTAssertEqual(upkeep.items?.last?.row?.detail?.text, "6 of 6")
    }
}
