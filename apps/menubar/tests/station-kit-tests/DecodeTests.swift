import StationKit
import XCTest

final class DecodeTests: XCTestCase {
    func testEveryFixtureDecodesAndSurvivesARoundTrip() throws {
        let fixtures = try Tree.fixtures()
        let onDisk = try FileManager.default.contentsOfDirectory(at: Tree.pages, includingPropertiesForKeys: nil)
            .filter { $0.pathExtension == "json" }
        XCTAssertFalse(fixtures.isEmpty)
        XCTAssertEqual(fixtures.count, onDisk.count, "a fixture of the pages folder was skipped")
        for (name, data) in fixtures {
            let page = try Page.decode(data)
            XCTAssertEqual(try Page.decode(page.encoded()), page, name)
        }
    }

    func testAPageThatSetsEveryFieldOfTheLanguageDecodesWhole() throws {
        let data = try Data(contentsOf: Tree.root.appendingPathComponent("apps/engine/page/testdata/every-field.json"))
        let page = try Page.decode(data)
        let sent = try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? NSDictionary)
        let kept = try XCTUnwrap(JSONSerialization.jsonObject(with: page.encoded()) as? NSDictionary)
        XCTAssertEqual(kept, sent, "the kit drops or changes a field the engine sends")
        XCTAssertEqual(try Page.decode(page.encoded()), page)
    }

    func testADocumentOfAnotherSchemaIsRefused() {
        XCTAssertThrowsError(try Page.decode(Data(#"{"schema":"station.status/v1"}"#.utf8)))
    }

    func testUnknownKeysAreIgnored() throws {
        let page = try Page.decode(Data(#"{"schema":"station.page/v1","id":"x","future":{"a":1}}"#.utf8))
        XCTAssertEqual(page.id, "x")
    }
}
