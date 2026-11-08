import StationKit
import SwiftUI
import XCTest

@MainActor
final class HomeCardsTests: XCTestCase {
    // invariant: SPEC A4 Home 3: the agents' fold hides the live agents past
    // the first two until it opens; the arrows move only through what shows.
    func testTheAgentsFoldHidesItsRowsUntilItOpens() throws {
        let page = try XCTUnwrap(Stories["home.agents-folded"].pages.first)
        let folded = HomeCards.navigable(page, unfolded: []).compactMap(\.id)
        XCTAssertEqual(folded, ["review-bot", "linux-1", "infra", "macbook-pro", "mac-mini", "macbook-air"])
        let open = HomeCards.navigable(page, unfolded: [HomeCards.agents]).compactMap(\.id)
        XCTAssertEqual(open, ["review-bot", "linux-1", "linux-3", "linux-5", "linux-6", "infra", "macbook-pro", "mac-mini", "macbook-air"])
    }

    // invariant: SPEC A3: Home unfolded past the panel's room stays within
    // its cap of 700 and scrolls its body; the stories draw it whole.
    func testHomeUnfoldedStaysWithinThePanelsCap() throws {
        for id in ["home.agents-unfolded", "home.agent-creating", "home.many-needs"] {
            let story = Stories[id]
            let view = Popover(story.state, scrolls: true).stationTheme(.dark).environment(\.stationStill, true)
            let renderer = ImageRenderer(content: view.fixedSize())
            renderer.scale = 1
            let height = try XCTUnwrap(renderer.cgImage).height
            XCTAssertLessThanOrEqual(CGFloat(height), Token.Metric.Panel.cap, id)
            XCTAssertGreaterThan(CGFloat(height), Token.Metric.Panel.cap - Token.Metric.Panel.footer.bottom, id)
        }
    }
}
