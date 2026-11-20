import XCTest

final class ViewCoverageTests: XCTestCase {
    static let views = ["atom/", "molecule/", "organism/", "template/", "motion/"]
    static let declaration = try! NSRegularExpression(pattern: #"^(?:(?:public|private|fileprivate|internal|final|@MainActor) )*(?:struct|enum|class|extension|func|protocol|actor) "#)
    static let view = try! NSRegularExpression(pattern: #"^(?:(?:public|private|fileprivate|internal) )?struct (\w+)(?:<.*>)?: (?:SwiftUI\.)?View\b"#)

    // value: the kit's views that record nothing in a view golden yet, the
    // recorder's pending output: a view that records leaves it, and the
    // list only shrinks.
    static let pending: [String] = []

    // invariant: every view a file of the kit declares at its top level
    // records itself, by a `.recorded(` in its declaration; the names of
    // those that do not, in their files' order.
    static func unrecorded(_ files: [(path: String, text: String)]) -> [String] {
        var out: [String] = []
        for (path, text) in files where views.contains(where: path.hasPrefix) {
            let lines = text.components(separatedBy: "\n")
            func matches(_ re: NSRegularExpression, _ line: String) -> NSTextCheckingResult? { re.firstMatch(in: line, range: NSRange(line.startIndex..., in: line)) }
            let starts = lines.indices.filter { matches(declaration, lines[$0]) != nil } + [lines.count]
            for (start, end) in zip(starts, starts.dropFirst()) {
                guard let found = matches(view, lines[start]), let name = Range(found.range(at: 1), in: lines[start]) else { continue }
                if !lines[start..<end].contains(where: { $0.contains(".recorded(") }) { out.append(String(lines[start][name])) }
            }
        }
        return out
    }

    func testEveryViewOfTheKitRecordsItselfButThePendingOnes() {
        let files = KitBoundaryTests.swiftFiles(Tree.sources)
        XCTAssertGreaterThan(files.count, 5)
        XCTAssertEqual(Self.unrecorded(files).sorted(), Self.pending)
    }

    func testTheCoverageScanNamesAViewThatRecordsNothing() {
        let found = Self.unrecorded([
            ("atom/Seen.swift", "public struct Seen: View {\n    var body: some View { Text(\"a\").recorded(.text(\"a\", .rowName)) }\n}\n\nstruct Unseen<Label: View>: View {\n    var body: some View { Text(\"b\") }\n}"),
            ("token/Swatch.swift", "struct Swatch: View {\n    var body: some View { Text(\"c\") }\n}"),
        ])
        XCTAssertEqual(found, ["Unseen"])
    }
}
