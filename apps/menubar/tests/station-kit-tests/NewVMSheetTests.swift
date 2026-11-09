import Foundation
import StationKit
import XCTest

// invariant: SPEC A5 New VM, in what the owner sees: the sheet says the
// VM it makes in the engine's words, a name in use creates nothing, and
// Create sends the call its form asks.
final class NewVMSheetTests: XCTestCase {
    func home() -> Page { Stories["sheet.new-vm"].pages[0] }

    func sheet(_ id: String = "new-vm") throws -> (Action, Answers) {
        let action = try XCTUnwrap(home().action(id))
        return (action, Answers(try XCTUnwrap(action.form)))
    }

    func testCreatingSaysTheVMAndTheMacItIsMadeOn() throws {
        let (action, answers) = try sheet()
        guard case let .runs(sent, hud) = SheetOutcome.of(action, answers) else { return XCTFail("a free name creates") }
        XCTAssertEqual(hud, "Creating linux-2 on This Mac…")
        XCTAssertEqual(sent["Request.Name"], "linux-2")
        XCTAssertEqual(sent["Share"], #"["github","claude"]"#)
    }

    func testANameAlreadyInUseCreatesNothingAndTheHUDSaysSo() throws {
        let (action, answers) = try sheet("new-vm:mac-mini")
        var taken = answers
        taken.given["Request.Name"] = "linux-1"
        XCTAssertEqual(SheetOutcome.of(action, taken), .refused(hud: "linux-1 already exists"))
    }

    func testTheFoldedOptionsSayWhatTheVMSharesOrThatItSharesNone() throws {
        let (_, answers) = try sheet()
        let options = try XCTUnwrap(answers.field("Options"))
        let share = try XCTUnwrap(answers.field("Sharing"))
        XCTAssertEqual(answers.fill(options.summary), "Throwaway · Linux · 8 GB")
        XCTAssertEqual(answers.fill(options.note ?? ""), "linux-2 · shares GitHub and Claude for 4 hr")
        XCTAssertEqual(answers.fill(share.summary), "GitHub, Claude for 4 hr")
        var none = answers
        none.given["Share"] = ""
        XCTAssertEqual(none.fill(options.note ?? ""), "linux-2 · shares no account")
        XCTAssertEqual(none.fill(share.summary), "No account")
        XCTAssertFalse(none.shown(try XCTUnwrap(none.field("ShareFor"))), "Share for hides while nothing is shared")
        XCTAssertNil(none.sent["ShareFor"])
    }

    // invariant: SPEC A9 1-8: the hatch starts where the Mac's free memory
    // truly falls on the log scale, and a stop past it is refused.
    func testTheMemoryPastWhatTheMacHasFreeIsHatchedAndRefused() throws {
        let (_, answers) = try sheet()
        let memory = try XCTUnwrap(answers.field("Request.Size.MemoryBytes"))
        XCTAssertEqual(memory.limit_note, "Over 28 GB would squeeze This Mac")
        XCTAssertEqual(MemorySlider.share(8, stops: [4, 8, 16, 32]), 1.0 / 3, accuracy: 1e-9)
        XCTAssertEqual(MemorySlider.share(Double(memory.limit ?? 0), stops: memory.stops ?? []), log2(28.0 / 4) / 3, accuracy: 1e-9)
        XCTAssertEqual(memory.options?.filter { $0.disabled != nil }.compactMap(\.label), ["32 GB"])
    }

    func testOnAndKindOpenTheFormTheEngineBuiltForThatMacAndKind() throws {
        let page = home()
        for action in (page.actions ?? []) where action.sheet == "new-vm" {
            for field in Answers(try XCTUnwrap(action.form)).fields where field.swaps == true {
                for option in field.options ?? [] where option.disabled == nil {
                    XCTAssertEqual(page.action(option.value ?? "")?.sheet, "new-vm", "\(action.id ?? ""): \(field.key ?? "") offers \(option.value ?? "")")
                }
            }
        }
        let (_, server) = try sheet("new-server:macbook-pro")
        XCTAssertEqual(server.fill(server.field("Options")?.summary), "Server · Linux · 8 GB")
        XCTAssertEqual(server.form.warn, "This Mac may sleep: a server pauses with it.")
    }

    func testCreateSendsItsCallWithTheAnswersShownAndNoSwap() async throws {
        let (action, _) = try sheet()
        let engine = try FakeEngine("mb-sheet-create")
        engine.serveOnce(Data(#"{"v":2,"id":"1","out":{"VM":{"Name":"linux-2"}}}"#.utf8) + [0x0A], thenClose: false)
        defer { engine.stop() }
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path).run(action, answers: ["Request.Lifetime": "permanent", "Share": "claude", "Options": Answers.open])
        XCTAssertNil(status)
        XCTAssertEqual(engine.received, [#"{"call":"machines.create-workspace","id":"1","in":{"Request":{"Image":"linux","Lifetime":"permanent","Name":"linux-2","Network":"nat","Size":{"MemoryBytes":8589934592}},"Share":["claude"],"ShareFor":14400000000000},"v":2}"#])
    }

    func testCreateAndOpenInTerminalOpensItOnceMade() async throws {
        let open = try XCTUnwrap(home().action("open-new:new-vm"))
        let engine = try FakeEngine("mb-sheet-open")
        engine.serveOnce(Data(#"{"v":2,"id":"1","out":{"VM":{"Name":"linux-2"}}}"#.utf8) + [0x0A], thenClose: false)
        defer { engine.stop() }
        let calls = Recorded()
        let status = await ActionRunner(executable: "/u/bin/station", socket: engine.path, launcher: RecordingLauncher(calls: calls)).run(open)
        XCTAssertNil(status)
        XCTAssertEqual(engine.received.count, 1)
        XCTAssertTrue(calls.all.first?.joined(separator: " ").contains("'--on' 'linux-2'") == true, "\(calls.all)")
    }
}
