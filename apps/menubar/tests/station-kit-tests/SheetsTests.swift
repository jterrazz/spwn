import Foundation
import StationKit
import XCTest

// invariant: SPEC A5's other sheets, in what the owner sees: Grant a
// password names the password and grants it from This Mac to the
// workspace it is asked for; Clean up's button says what it does with what is checked, and
// sends only what is left checked; a hint's command is the page's.
final class SheetsTests: XCTestCase {
    func sheet(_ story: String, _ id: String) throws -> (Page, Action, Answers) {
        let page = Stories[story].pages[0]
        let action = try XCTUnwrap(page.action(id))
        return (page, action, Answers(try XCTUnwrap(action.form)))
    }

    func testAGrantSaysThePasswordItGrantsAndTheyStayUntilRemoved() throws {
        let (_, _, answers) = try sheet("sheet.grant", "grant")
        XCTAssertEqual(answers.primary.map { answers.fill($0.label ?? "") }, "Grant GitHub token")
        var other = answers
        other.given["Entry"] = "pw-claude"
        XCTAssertEqual(other.primary.map { other.fill($0.label ?? "") }, "Grant Claude token")
        XCTAssertEqual(answers.sent["Entry"], "pw-github")
        XCTAssertNil(answers.sent["To"])
    }

    func testPickingAnotherWorkspaceGrantsToIt() throws {
        let (page, action, answers) = try sheet("sheet.grant", "grant")
        let to = try XCTUnwrap(answers.field("To"))
        let agent = try XCTUnwrap(to.options?.first { $0.label == "agent-7f" }?.value)
        let other = try XCTUnwrap(page.action(agent))
        XCTAssertEqual(other.run?.call, "grants.add")
        XCTAssertEqual(other.run?.in?["To"], "agent-7f")
        XCTAssertEqual(action.run?.in?["To"], "linux-1")
    }

    func testCleanUpSaysWhatItDeletesAndForgetsAsTheOwnerChecks() throws {
        let (_, action, answers) = try sheet("sheet.cleanup.mac", "cleanup:macbook-pro")
        XCTAssertEqual(answers.primary?.label, "Delete 1 ephemeral workspace")
        XCTAssertEqual(answers.primary?.danger, true)
        let agents = try XCTUnwrap(answers.field("Agent"))
        XCTAssertEqual(answers.picked(agents), ["agent-7f"], "the ephemeral workspace checked carries its finished agent")
        XCTAssertTrue(answers.follows(agents, try XCTUnwrap(agents.options?.first)))
        var kept = answers
        kept.given["VMs"] = ""
        XCTAssertEqual(kept.picked(agents), [], "unchecked, the ephemeral workspace leaves its agent to the owner")
        XCTAssertNil(kept.primary, "nothing checked, the button says nothing it would do")
        XCTAssertEqual(action.run?.call, "machines.cleanup")
    }

    func calls(_ action: Action, _ given: [String: String]) throws -> [String] {
        let run = try XCTUnwrap(action.run)
        return ActionRunner.steps([run] + (run.and ?? []), of: action, answers: given).map { step, input in
            ([step.call ?? ""] + input.sorted { $0.key < $1.key }.map { "\($0.key)=\($0.value)" }).joined(separator: " ")
        }
    }

    func testCleanUpDeletesTheEphemeralCheckedAndItsAgentGoesWithIt() throws {
        let (_, action, _) = try sheet("sheet.cleanup.mac", "cleanup:macbook-pro")
        XCTAssertEqual(try calls(action, [:]), [#"machines.cleanup Confirm=yes VMs=["agent-7f"]"#])
        XCTAssertEqual(try calls(action, ["Agent": "agent-7f"]), [#"machines.cleanup Confirm=yes VMs=["agent-7f"]"#], "an agent its ephemeral workspace takes is not forgotten apart")
    }

    func testCleanUpKeepsTheEphemeralUncheckedAndForgetsTheAgentChecked() throws {
        let (_, action, form) = try sheet("sheet.cleanup.mac", "cleanup:macbook-pro")
        var kept = form
        kept.given = ["VMs": "", "Agent": "agent-7f"]
        XCTAssertEqual(kept.primary?.label, "Forget 1 agent")
        XCTAssertEqual(try calls(action, kept.given), ["agents.forget Agent=agent-7f"])
        XCTAssertEqual(try calls(action, ["VMs": ""]), [], "nothing checked, nothing is asked")
    }

    func testCleanUpOfEveryMacSendsOnlyWhatIsLeftChecked() throws {
        let fixed = ["VMs": #"["agent-7f","old"]"#, "Confirm": "yes"]
        XCTAssertEqual(ActionRunner.narrowed(fixed, by: ["VMs": #"["old"]"#], checks: ["VMs"]), ["VMs": #"["old"]"#, "Confirm": "yes"])
        XCTAssertNil(ActionRunner.narrowed(fixed, by: ["VMs": #"["linux-1"]"#], checks: ["VMs"]), "a Mac with nothing checked is not asked")
        XCTAssertNil(ActionRunner.narrowed(fixed, by: [:], checks: ["VMs"]), "an ephemeral workspace unchecked is not deleted")
        XCTAssertEqual(ActionRunner.narrowed(["Agent": "docs"], by: ["Agent": #"["docs"]"#], checks: ["Agent"]), ["Agent": "docs"])
        XCTAssertNil(ActionRunner.narrowed(["Agent": "docs"], by: ["Agent": #"["agent-7f"]"#], checks: ["Agent"]), "an agent unchecked is not forgotten")
        XCTAssertNil(ActionRunner.narrowed(["Agent": "docs"], by: [:], checks: ["Agent"]), "no agent checked, none is forgotten")
    }

    func testAHintsCommandIsThePagesOwn() throws {
        let (_, action, answers) = try sheet("sheet.hint.add-mac", "add-mac")
        let commands = (action.form?.fields ?? []).compactMap(\.hint?.command)
        XCTAssertEqual(commands, ["curl -fsSL https://raw.githubusercontent.com/jterrazz/studio-station/main/scripts/install.sh | sh", "station machines add <mac>"])
        XCTAssertEqual(answers.primary?.label, "Done")
        let (_, declare, _) = try sheet("sheet.hint.declare-public", "declare-public:macbook-pro:5000")
        XCTAssertEqual(declare.form?.fields?.first?.hint?.command, "station ports declare 5000 --via <route>")
    }

    func testSharingAPortTypesNothingAndSaysWhenNothingListens() throws {
        let (page, action, answers) = try sheet("sheet.share-port", "share-port")
        let rows = try XCTUnwrap(answers.field("Port"))
        XCTAssertEqual(rows.options?.map(\.label), ["Astro", "Node inspector"])
        XCTAssertEqual(rows.options?.compactMap { page.action($0.value ?? "")?.run?.call }, ["ports.share", "ports.share"])
        XCTAssertNil(action.run)
        let (_, empty, _) = try sheet("sheet.share-port.empty", "share-port")
        XCTAssertEqual(empty.form?.fields?.last?.options ?? [], [])
        XCTAssertEqual(empty.form?.fields?.last?.note, "Nothing listens in review-bot right now.")
    }
}
