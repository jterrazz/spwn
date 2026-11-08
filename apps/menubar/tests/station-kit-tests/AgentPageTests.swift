import StationKit
import XCTest

@MainActor
final class AgentPageTests: XCTestCase {
    // invariant: SPEC A4: the tint is on the next step: Open PR while the
    // PR is unopened, Merge once it was opened from the page.
    func testTheTintMovesToMergeOnceThePRIsOpened() {
        let waiting = Stories["agent.waiting"].state, opened = Stories["agent.waiting.pr-opened"].state
        XCTAssertEqual(WaitCard.tinted(opened: waiting.prOpened.contains("review-bot")), "open-pr")
        XCTAssertEqual(WaitCard.tinted(opened: opened.prOpened.contains("review-bot")), "merge")
        XCTAssertEqual(opened.shown?.action("merge")?.run?.call, "agents.merge")
    }

    // invariant: an agent's page and a workspace's are pushed pages, each
    // with its way back, drawn over Home.
    func testAnAgentsAndAWorkspacesPagesArePushed() {
        for id in ["agent.working", "workspace.agents"] {
            let state = Stories[id].state
            XCTAssertNil(state.page, id)
            XCTAssertEqual(state.pushed?.header?.back, "All machines", id)
        }
    }
}
