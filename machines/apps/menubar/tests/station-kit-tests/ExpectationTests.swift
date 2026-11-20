import Foundation
import XCTest

final class ExpectationTests: XCTestCase {
    func testTheListIsOneRootRelativePathPerLine() throws {
        let file = FileManager.default.temporaryDirectory.appendingPathComponent("expect-\(UUID().uuidString).txt")
        defer { try? FileManager.default.removeItem(at: file) }
        try "# a note\n  a/b.png  \n\nc/d.png\n".write(to: file, atomically: true, encoding: .utf8)
        XCTAssertEqual(Expectation.paths(file.path), ["a/b.png", "c/d.png"])
        XCTAssertEqual(Expectation.paths(nil), [])
        XCTAssertEqual(Expectation.paths(file.path + ".missing"), [])
    }

    func testOnlyAGoldenOfTheListIsListed() {
        let listed = Tree.storyGoldens.appendingPathComponent("home.fine.dark.png")
        let set: Set<String> = ["apps/menubar/tests/station-kit-tests/__expected__/stories/home.fine.dark.png"]
        XCTAssertTrue(Expectation.listed(listed, in: set))
        XCTAssertFalse(Expectation.listed(Tree.storyGoldens.appendingPathComponent("home.fine.light.png"), in: set))
        XCTAssertFalse(Expectation.listed(listed, in: []))
    }
}
