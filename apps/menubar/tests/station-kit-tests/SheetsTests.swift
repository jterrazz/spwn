import Foundation
import StationKit
import XCTest

// invariant: SPEC A5's other sheets, in what the owner sees: Lend an
// account names the account and the time and lends from the workspace's
// own Mac; Clean up's button says what it does with what is checked, and
// sends only what is left checked; a hint's command is the page's.
final class SheetsTests: XCTestCase {
    func sheet(_ story: String, _ id: String) throws -> (Page, Action, Answers) {
        let page = Stories[story].pages[0]
        let action = try XCTUnwrap(page.action(id))
        return (page, action, Answers(try XCTUnwrap(action.form)))
    }

    func testLendingSaysTheAccountAndTheTimeAndWhenItEnds() throws {
        let (_, _, answers) = try sheet("sheet.lend-account", "share-account")
        XCTAssertEqual(answers.primary.map { answers.fill($0.label ?? "") }, "Share GitHub for 4 hr")
        var longer = answers
        longer.given["Account"] = "claude"
        longer.given["For"] = "604800000000000"
        XCTAssertEqual(longer.primary.map { longer.fill($0.label ?? "") }, "Share Claude for 7 days")
        let ends = try XCTUnwrap(answers.field("For")?.options).compactMap(\.note)
        XCTAssertEqual(ends, ["Ends at 14:30, then taken back", "Ends at 22:30, then taken back", "Ends Saturday 14 Nov"])
        XCTAssertEqual(answers.sent["Account"], "github")
        XCTAssertNil(answers.sent["To"])
    }

    func testPickingAnotherWorkspaceLendsFromItsMac() throws {
        let (page, action, answers) = try sheet("sheet.lend-account", "share-account")
        let to = try XCTUnwrap(answers.field("To"))
        let linux3 = try XCTUnwrap(to.options?.first { $0.label == "linux-3" }?.value)
        let other = try XCTUnwrap(page.action(linux3))
        XCTAssertEqual(other.form?.subtitle, "From mac-mini. It ends by itself.")
        XCTAssertEqual(other.run?.target?.name, "mac-mini")
        XCTAssertEqual(other.run?.in?["With"], "linux-3")
        XCTAssertNil(action.run?.target)
    }

    func testCleanUpSaysWhatItDeletesAndForgetsAsTheOwnerChecks() throws {
        let (_, action, answers) = try sheet("sheet.cleanup.mac", "cleanup:macbook-pro")
        XCTAssertEqual(answers.primary?.label, "Delete 1 throwaway")
        XCTAssertEqual(answers.primary?.danger, true)
        let agents = try XCTUnwrap(answers.field("Agent"))
        XCTAssertEqual(answers.picked(agents), ["agent-7f"], "the throwaway checked carries its finished agent")
        XCTAssertTrue(answers.follows(agents, try XCTUnwrap(agents.options?.first)))
        var kept = answers
        kept.given["VMs"] = ""
        XCTAssertEqual(kept.picked(agents), [], "unchecked, the throwaway leaves its agent to the owner")
        XCTAssertNil(kept.primary, "nothing checked, the button says nothing it would do")
        XCTAssertEqual(action.run?.call, "machines.cleanup")
    }

    func testCleanUpSendsOnlyWhatIsLeftChecked() throws {
        let fixed = ["VMs": #"["agent-7f","old"]"#, "Confirm": "yes"]
        XCTAssertEqual(ActionRunner.narrowed(fixed, by: ["VMs": #"["old"]"#]), ["VMs": #"["old"]"#, "Confirm": "yes"])
        XCTAssertNil(ActionRunner.narrowed(fixed, by: ["VMs": #"["linux-1"]"#]), "a Mac with nothing checked is not asked")
        XCTAssertEqual(ActionRunner.narrowed(["Agent": "docs"], by: ["Agent": #"["docs"]"#]), ["Agent": "docs"])
        XCTAssertNil(ActionRunner.narrowed(["Agent": "docs"], by: ["Agent": #"["agent-7f"]"#]), "an agent unchecked is not forgotten")
        XCTAssertEqual(ActionRunner.narrowed(["Agent": "docs"], by: [:]), ["Agent": "docs"])
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
