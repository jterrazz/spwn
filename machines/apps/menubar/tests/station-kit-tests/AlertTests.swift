import SwiftUI
import XCTest
@testable import StationKit

@MainActor
final class AlertTests: XCTestCase {
    func confirm(_ story: String) throws -> Confirm {
        let s = Stories[story]
        let page = try XCTUnwrap(s.state.shown)
        return try XCTUnwrap(page.action(try XCTUnwrap(s.ui.alert))?.confirm, story)
    }

    // value: the lines of a story's drawing that name what to type.
    func names(_ story: Story) -> [String] {
        ViewGolden.tree(ViewGolden.nodes(ViewGolden.record(AnyView(StoryView(story).stationAccessibility()), scheme: .dark, desk: true)))
            .filter { $0.contains("[font=Role.alertName]") }
    }

    // invariant: SPEC A5, ADR-097: the level is the engine's, drawn as it
    // comes: each alert opens a gesture its page asks, a typed alert draws
    // the name its page gives, a yes or no one draws none.
    func testEachAlertAsksAtTheLevelTheEngineGraded() throws {
        let asked = try Stories.load(root: Tree.stories).filter { $0.group == .alert }
        XCTAssertEqual(asked.count, 12)
        for story in asked {
            let confirm = try confirm(story.id)
            XCTAssertEqual(names(story).map { $0.contains("text \"\(confirm.type ?? "")\"") }, confirm.type == nil ? [] : [true], story.id)
        }
    }

    // invariant: the red button is never the default: a gesture that loses
    // something defaults to Cancel, and a typed one stays off until its
    // name is typed whole.
    func testTheRedButtonIsNeverTheDefaultAndATypedOneWaitsForItsName() throws {
        XCTAssertTrue(AlertCard.yesIsDefault(try confirm("alert.restart-server")))
        for id in ["alert.stop-server", "alert.restart-mac", "alert.delete-vm.ephemeral", "alert.forget-mac"] {
            XCTAssertFalse(AlertCard.yesIsDefault(try confirm(id)), id)
        }
        let delete = try confirm("alert.delete-server")
        XCTAssertFalse(AlertCard.allows(delete, typed: ""))
        XCTAssertFalse(AlertCard.allows(delete, typed: "infr"))
        XCTAssertTrue(AlertCard.allows(delete, typed: "infra"))
        let typed = Stories["alert.delete-server.typed"]
        XCTAssertTrue(AlertCard.allows(delete, typed: typed.state.answers["delete"]?[ActionRunner.typedKey] ?? ""))
    }

    // invariant: a question with no form is drawn on the panel's alert
    // layer, over the page, never in its flow.
    func testAQuestionIsDrawnOnTheAlertLayer() throws {
        let story = Stories["alert.restart-server"]
        XCTAssertNotNil(AlertLayer.layers(story.state.shown, story.state, PopoverHandlers()).alert)
        var closed = story.state
        closed.confirming = nil
        XCTAssertNil(AlertLayer.layers(story.state.shown, closed, PopoverHandlers()).alert)
    }

    // invariant: ADR-097: the answer the call takes is the owner's: the
    // name he typed when the alert asks it, which neither the page nor the
    // app fills, and the page's yes when it asks yes or no.
    func testTheCallTakesTheAnswerItsOwnLevelAsks() throws {
        let delete = try XCTUnwrap(Stories["alert.delete-server"].state.shown?.action("delete"))
        XCTAssertEqual(delete.confirm?.type, "infra")
        XCTAssertNil(delete.run?.in?[ActionRunner.typedKey])
        XCTAssertNil(ActionRunner.filled(delete, answers: [:])[ActionRunner.typedKey])
        XCTAssertEqual(ActionRunner.filled(delete, answers: [ActionRunner.typedKey: "infra"])[ActionRunner.typedKey], "infra")
        for (story, id) in [("alert.stop-server", "stop"), ("alert.restart-server", "restart"), ("alert.restart-mac", "restart-mac")] {
            let action = try XCTUnwrap(Stories[story].state.shown?.action(id), story)
            XCTAssertNil(action.confirm?.type, story)
            XCTAssertNil(ActionRunner.filled(action, answers: [ActionRunner.typedKey: "infra"])[ActionRunner.typedKey], story)
            XCTAssertEqual(action.run?.in?[ActionRunner.typedKey], "yes", story)
        }
    }
}
