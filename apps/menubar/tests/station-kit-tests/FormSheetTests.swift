import AppKit
@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class FormSheetTests: XCTestCase {
    func decoded<T: Decodable>(_ json: String) throws -> T { try JSONDecoder().decode(T.self, from: Data(json.utf8)) }

    // invariant: a form's first text field takes the keys as it opens.
    func testAFormsFirstTextFieldTakesTheKeys() {
        var pick = Field(), name = Field()
        pick.key = "Image"; pick.kind = "select"
        name.key = "Name"; name.kind = "text"
        var f = Form()
        f.fields = [pick, name]
        XCTAssertEqual(FormSheet.first(f), "Name")
    }

    func awake() throws -> (Page, [Action]) {
        let page = try Tree.page("menubar.awake.two-macs")
        let mac = try XCTUnwrap(page.section("this-mac")?.rows.first?.id)
        return (page, page.actions(for: mac))
    }

    // invariant: the until form's options sit four to a row, and its
    // confirm names the hour picked; New VM's confirm is Create.
    func testFormConfirmSaysWhatItDoes() throws {
        let (_, keeps) = try awake()
        let until = try XCTUnwrap(keeps.first { $0.form != nil })
        let form = try XCTUnwrap(until.form)
        let field = try XCTUnwrap(form.fields?.first)
        let last = try XCTUnwrap(field.options?.last)
        XCTAssertEqual(FormSheet.verb(until.label ?? "", form, [field.key ?? "": last.value ?? ""]), "Until \(last.label ?? "")")
        XCTAssertGreaterThan(field.options?.count ?? 0, Choice.perRow)
        let home = try Tree.page("menubar.home.two-macs")
        let new = try XCTUnwrap(home.action("new-vm"))
        XCTAssertEqual(FormSheet.verb(new.label ?? "", try XCTUnwrap(new.form), [:]), "Create")
    }

    // invariant: a port is a number from 1 to 65535: `abc` keeps Share
    // off.
    func testSharePortTakesOnlyAPort() throws {
        let form: StationKit.Form = try decoded(#"{"title":"Share a port","fields":[{"key":"Port","label":"Port","kind":"text","required":true}]}"#)
        XCTAssertFalse(FormSheet.complete(form, ["Port": "abc"]))
        XCTAssertFalse(FormSheet.complete(form, ["Port": "70000"]))
        XCTAssertFalse(FormSheet.complete(form, [:]))
        XCTAssertTrue(FormSheet.complete(form, ["Port": "3000"]))
    }

    // invariant: a port field holding what is not a port says why Share
    // stays off; an empty or valid one says nothing.
    func testSharePortSaysWhatAPortIs() throws {
        let form: StationKit.Form = try decoded(#"{"title":"Share a port","fields":[{"key":"Port","label":"Port","kind":"text","required":true}]}"#)
        let field = try XCTUnwrap(form.fields?.first)
        XCTAssertEqual(FormSheet.refused(field, ["Port": "abc"]), "A port is a number from 1 to 65535.")
        XCTAssertEqual(FormSheet.refused(field, ["Port": "70000"]), "A port is a number from 1 to 65535.")
        XCTAssertNil(FormSheet.refused(field, [:]))
        XCTAssertNil(FormSheet.refused(field, ["Port": "8080"]))
    }

    // invariant: a confirm's command is set off as code, never as prose.
    func testAConfirmSetsItsCommandOffAsCode() throws {
        let line: Value = try decoded(#"{"kind":"command","text":"station engine start"}"#)
        XCTAssertEqual(ConfirmSheet.line(line).code, true)
        XCTAssertEqual(ConfirmSheet.line(line).words, "station engine start")
        let prose: Value = try decoded(#"{"kind":"text","text":"linux-1 will stop; its disk is kept"}"#)
        XCTAssertEqual(ConfirmSheet.line(prose).code, false)
    }

    // invariant: a button whose sheet asks for the name reads as its label
    // and an ellipsis, live, never greyed with the instruction inside it.
    func testATypedStopIsAPlainButton() {
        var stop = Action()
        stop.id = "stop"; stop.label = "Stop"
        var c = Confirm()
        c.type = "infra"
        stop.confirm = c
        XCTAssertEqual(MachineContent.button(stop), "Stop\(Token.Glyph.ellipsis)")
        XCTAssertEqual(MachineContent.role(stop), .plain)
    }
}
