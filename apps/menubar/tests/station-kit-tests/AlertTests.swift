import SwiftUI
import XCTest
@testable import StationKit

@MainActor
final class AlertTests: XCTestCase {
    func confirm(_ story: String) throws -> Confirm {
        let s = Stories[story]
        let page = try XCTUnwrap(s.state.shown)
        return try XCTUnwrap(page.action(try XCTUnwrap(s.ui.alert))?.confirm)
    }

    // invariant: SPEC A5: the level is the engine's, drawn as it comes: a
    // typed alert names what to type, a yes or no one names nothing.
    func testEachAlertAsksAtTheLevelTheEngineGraded() throws {
        let typed = ["alert.stop-server", "alert.stop-server.typed", "alert.delete-server", "alert.delete-vm.permanent", "alert.restart-mac"]
        let asked = try Stories.load(root: Tree.stories).filter { $0.group == .alert }.map(\.id)
        XCTAssertEqual(asked.count, 16)
        for id in asked {
            XCTAssertEqual(try confirm(id).type != nil, typed.contains(id), id)
        }
    }

    // invariant: the red button is never the default: a gesture that loses
    // something defaults to Cancel, and a typed one stays off until its
    // name is typed whole.
    func testTheRedButtonIsNeverTheDefaultAndATypedOneWaitsForItsName() throws {
        XCTAssertTrue(AlertCard.yesIsDefault(try confirm("alert.restart-server")))
        for id in ["alert.stop-server", "alert.delete-vm.throwaway", "alert.stop-agent", "alert.forget-mac"] {
            XCTAssertFalse(AlertCard.yesIsDefault(try confirm(id)), id)
        }
        let stop = try confirm("alert.stop-server")
        XCTAssertFalse(AlertCard.allows(stop, typed: ""))
        XCTAssertFalse(AlertCard.allows(stop, typed: "infr"))
        XCTAssertTrue(AlertCard.allows(stop, typed: "infra"))
        let typed = Stories["alert.stop-server.typed"]
        XCTAssertTrue(AlertCard.allows(stop, typed: typed.state.answers["stop"]?[ActionRunner.typedKey] ?? ""))
    }

    // invariant: a question with no form is drawn on the panel's alert
    // layer, over the page, never in its flow.
    func testAQuestionIsDrawnOnTheAlertLayer() throws {
        let story = Stories["alert.stop-vm"]
        XCTAssertNotNil(AlertLayer.layers(story.state.shown, story.state, PopoverHandlers()).alert)
        var closed = story.state
        closed.confirming = nil
        XCTAssertNil(AlertLayer.layers(story.state.shown, closed, PopoverHandlers()).alert)
    }

    // invariant: the answer the call takes: the name typed when the call
    // checks it, the page's fixed yes when the alert asked typed of a call
    // that asks yes or no (a Mac hosting a server restarts on yes).
    func testTheCallTakesTheAnswerItsOwnLevelAsks() throws {
        let server = try XCTUnwrap(Stories["alert.stop-server"].state.shown?.action("stop"))
        XCTAssertEqual(ActionRunner.filled(server, answers: [ActionRunner.typedKey: "infra"])[ActionRunner.typedKey], "infra")
        let mac = try XCTUnwrap(Stories["alert.restart-mac"].state.shown?.action("restart-mac"))
        XCTAssertEqual(mac.confirm?.type, "mac-mini")
        XCTAssertNil(ActionRunner.filled(mac, answers: [ActionRunner.typedKey: "mac-mini"])[ActionRunner.typedKey])
        XCTAssertEqual(mac.run?.in?[ActionRunner.typedKey], "yes")
    }
}
