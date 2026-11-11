import Foundation
import StationKit
import XCTest

final class FindTests: XCTestCase {
    static let recent = ["linux-1", "mac-mini", "infra"]

    func index(_ world: String = "machines.find") throws -> FindIndex {
        let url = Tree.stories.appendingPathComponent("pages/\(world).find.json")
        return FindIndex(try Page.decode(Data(contentsOf: url)))
    }

    func said(_ groups: [FindGroup]) -> [String] {
        groups.flatMap { g in ["\(g.title):"] + g.entries.map(\.text) + (g.more > 0 ? ["Show \(g.more) more"] : []) }
    }

    // invariant: SPEC A6 [find.empty]: with nothing typed, Find lists the
    // top needs by their verbs, the machines opened last, This Mac's awake
    // action and New VM on the two roomiest Macs.
    func testNothingTypedListsTheNeedsTheRecentMachinesAndTheActionsAtHand() throws {
        XCTAssertEqual(said(try index().results(query: "", recent: Self.recent)), [
            "Needs you:", "Review review-bot · PR #42 is ready", "Extend Claude on linux-1",
            "Recent:", "linux-1", "mac-mini", "infra",
            "Actions:", "Keep This Mac awake 4 hr", "New VM on This Mac", "New VM on mac-mini",
        ])
    }

    // invariant: SPEC A6 [find.down]: a down server leads the needs as
    // "Restart infra", and the needs stop at three.
    func testADownServerLeadsTheNeedsAndTheyStopAtThree() throws {
        let groups = try index("machines.find.down").results(query: "", recent: Self.recent)
        XCTAssertEqual(groups.first?.entries.map(\.title), ["Restart infra", "Review review-bot · PR #42 is ready", "Extend Claude on linux-1"])
    }

    // invariant: SPEC B5.21: Recent is the app's own list, the machines it
    // names as the index has them, none when it is empty.
    func testRecentListsOnlyTheMachinesTheAppOpenedLast() throws {
        let groups = try index().results(query: "", recent: ["infra", "gone-vm"])
        XCTAssertEqual(groups.first { $0.id == FindIndex.recent }?.entries.map(\.title), ["infra"])
        XCTAssertNil(try index().results(query: "", recent: []).first { $0.id == FindIndex.recent })
    }

    // invariant: SPEC A6: a result found by its title's start outranks one
    // found by a word's start, then by what its title holds, then by a tag.
    func testAQueryRanksPrefixThenWordThenSubstringThenTag() throws {
        let e = { (title: String, tags: [String]) in FindEntry(row: Row(), group: FindIndex.actions, title: title, address: nil, scope: nil, tags: tags, prefix: nil) }
        XCTAssertEqual(FindIndex.score(e("Awake until linux-1 ends", []), query: "awake"), 0)
        XCTAssertEqual(FindIndex.score(e("Keep This Mac awake 4 hr", []), query: "awake"), 1)
        XCTAssertEqual(FindIndex.score(e("mac-mini", []), query: "ini"), 2)
        XCTAssertEqual(FindIndex.score(e("New VM on This Mac", ["create"]), query: "cre"), 3)
        XCTAssertNil(FindIndex.score(e("New VM on This Mac", ["create"]), query: "zzzq"))
        XCTAssertEqual(said(try index().results(query: "awake", recent: [])), [
            "Actions:", "Awake until linux-1 ends", "Keep This Mac awake 4 hr", "Sections:", "Awake · All Macs",
        ])
    }

    // invariant: SPEC A6 [find.port, find.machine, find.action]: a port is
    // found by its number, a Mac by a part of its name with its actions and
    // sections, New VM by its verb.
    func testAQueryFindsAPortAMachineAndAnAction() throws {
        let find = try index()
        XCTAssertEqual(said(find.results(query: "5173", recent: [])), ["Ports:", "Vite · linux-1:5173"])
        XCTAssertEqual(said(find.results(query: "mini", recent: [])), [
            "Machines:", "mac-mini", "Actions:", "New VM on mac-mini", "Sections:", "Ports of mac-mini", "Activity of mac-mini",
        ])
        XCTAssertEqual(said(find.results(query: "new", recent: [])), ["Actions:", "New VM on This Mac", "New VM on mac-mini"])
        XCTAssertEqual(find.results(query: "zzzq", recent: []), [])
    }

    // invariant: SPEC A6 [find.keyboard, find.more-open] and A9 1-16: three
    // per group then "Show N more", which opens its group in place while
    // the others keep theirs; two letters that begin a VM's name list its
    // verbs first among the actions.
    func testThreePerGroupThenShowMoreWhichOpensItsGroupInPlace() throws {
        let find = try index()
        XCTAssertEqual(said(find.results(query: "linux", recent: [])), [
            "Machines:", "linux-1", "linux-3",
            "Ports:", "Vite · linux-1:5173", "Port 3000 · linux-1:3000", "Port 8080 · linux-3:8080", "Show 3 more",
            "Actions:", "Open linux-1 in Terminal", "Share a port of linux-1", "Lend an account to linux-1", "Show 3 more",
        ])
        let open = find.results(query: "linux", recent: [], open: [FindIndex.ports])
        XCTAssertEqual(open.first { $0.id == FindIndex.ports }?.entries.count, 6)
        XCTAssertEqual(open.first { $0.id == FindIndex.ports }?.more, 0)
        XCTAssertEqual(open.first { $0.id == FindIndex.actions }?.more, 3)
        XCTAssertEqual(FindIndex.lines(open).count, 12)
    }

    // invariant: one letter finds by what it begins or holds, never a VM's
    // verbs [find.type-to-find].
    func testOneLetterAddsNoVerbs() throws {
        let groups = try index().results(query: "l", recent: [])
        XCTAssertFalse(groups.flatMap(\.entries).contains { $0.group == FindIndex.verbs })
        XCTAssertEqual(groups.first { $0.id == FindIndex.actions }?.entries.map(\.title), ["Awake until linux-1 ends", "Clean up ephemeral workspaces on This Mac"])
    }

    // invariant: the keys never leave the lines drawn; typing starts over
    // on the first line with no group opened.
    func testTheLightStaysOnTheLinesAndTypingStartsOver() {
        let q = FindQuery("linux", selected: 2, open: [FindIndex.ports])
        XCTAssertEqual(q.moved(5, lines: 4).selected, 3)
        XCTAssertEqual(q.moved(-9, lines: 4).selected, 0)
        XCTAssertEqual(q.typed("linux-"), FindQuery("linux-"))
        XCTAssertEqual(q.opening(FindIndex.ports).open, [FindIndex.ports])
    }

    // invariant: SPEC A9 2-15: the part matched is cut out of the title as
    // typed, case aside, so it is drawn in t1 semibold.
    func testThePartMatchedIsCutOutCaseAside() {
        let cut = FindResult.split("Ports of mac-mini", "MINI")
        XCTAssertEqual(cut?.0, "Ports of mac-")
        XCTAssertEqual(cut?.1, "mini")
        XCTAssertEqual(cut?.2, "")
        XCTAssertNil(FindResult.split("infra", "zz"))
    }
}
