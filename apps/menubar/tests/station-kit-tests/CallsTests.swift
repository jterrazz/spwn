import Foundation
import StationKit
import XCTest

final class CallsTests: XCTestCase {
    func names() throws -> Set<String> {
        let obj = try XCTUnwrap(JSONSerialization.jsonObject(with: Data(contentsOf: Tree.calls)) as? [String: Any])
        let calls = try XCTUnwrap(obj["calls"] as? [[String: Any]])
        return Set(calls.compactMap { $0["name"] as? String })
    }

    // invariant: MB-08: every action the menu bar can show names a call of
    // the generated calls.json, opens a terminal, or copies or opens a link
    // the app itself handles.
    func testActionsAreCalls() throws {
        let known = try names()
        var called = 0
        for (name, data) in try Tree.fixtures() {
            for action in try Page.decode(data).actions ?? [] {
                guard let run = action.run else { continue }
                if let call = run.call {
                    XCTAssertTrue(known.contains(call), "\(name): \(action.id ?? "") names \(call), no call of calls.json")
                    called += 1
                } else if run.copy == nil && run.open == nil {
                    XCTAssertEqual(run.terminal, true, "\(name): \(action.id ?? "") neither calls nor opens a terminal")
                }
            }
        }
        XCTAssertGreaterThan(called, 0)
        XCTAssertTrue(known.contains(ActionRunner.endsTheEngine))
    }

    static func execs(_ files: [(path: String, text: String)]) -> [String] {
        files.filter { $0.path != "engine/Terminal.swift" && ($0.text.contains("Process()") || $0.text.contains("NSTask")) }.map(\.path)
    }

    func swiftFiles(_ root: URL) -> [(path: String, text: String)] {
        FileManager.default.enumerator(at: root, includingPropertiesForKeys: nil)!.compactMap { $0 as? URL }
            .filter { $0.pathExtension == "swift" }
            .map { (String($0.path.dropFirst(root.path.count + 1)), try! String(contentsOf: $0, encoding: .utf8)) }
    }

    // invariant: rule menubar-no-exec (MB-09): a program is started in the
    // file that opens the terminal and nowhere else, in the kit or the app.
    func testNoCLIExec() {
        XCTAssertEqual(Self.execs(swiftFiles(Tree.sources) + swiftFiles(Tree.app)), [])
        XCTAssertEqual(Self.execs([("engine/Actions.swift", "let p = Process()"), ("engine/Terminal.swift", "let p = Process()")]), ["engine/Actions.swift"])
    }
}
