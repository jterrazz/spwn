import Foundation
import StationKit

enum Paths {
    static let stories = Stories.root
    static let goldens = stories.deletingLastPathComponent().appendingPathComponent("tests/station-kit-tests/testdata/stories")
    static let schemes = ["dark", "light"]

    static var design: URL? {
        ProcessInfo.processInfo.environment["STATION_DESIGN"].map { URL(fileURLWithPath: $0) }
    }

    static func capture(_ id: String, dark: Bool, design: URL? = Paths.design) -> URL? {
        guard let design else { return nil }
        let url = design.appendingPathComponent("v4/stories/\(id)\(dark ? "" : ".light").png")
        return FileManager.default.fileExists(atPath: url.path) ? url : nil
    }

    static func actual(_ id: String, scheme: String, goldens: URL = Paths.goldens) -> URL? {
        let url = goldens.appendingPathComponent("\(id).\(scheme).actual.png")
        return FileManager.default.fileExists(atPath: url.path) ? url : nil
    }

    static func drifted(_ id: String) -> Bool {
        schemes.contains { actual(id, scheme: $0) != nil }
    }

    static func order(root: URL = stories) -> [(id: String, section: String)] {
        guard let text = try? String(contentsOf: root.appendingPathComponent("STORIES.md"), encoding: .utf8) else { return [] }
        var section = ""
        return text.split(separator: "\n").compactMap { line in
            if line.hasPrefix("## ") { section = String(line.dropFirst(3)) }
            guard line.hasPrefix("### `"), let end = line.dropFirst(5).firstIndex(of: "`") else { return nil }
            return (String(line.dropFirst(5)[..<end]), section)
        }
    }
}
