import Foundation
import XCTest

// invariant: ADR-036: the app holds exactly one glass, the panel's
// NSGlassEffectView, and the content is that glass's contentView, never a
// sibling laid over it; no other glass, container or backdrop of any era
// is drawn in the app, so nothing is glass on glass.
final class AppGlassTests: XCTestCase {
    static let others = ["glassEffect", "GlassEffectContainer", "NSGlassEffectContainerView", "Material", ".containerBackground", "NSVisualEffectView"]
    static let made = try! NSRegularExpression(pattern: #"let (\w+) = NSGlassEffectView\("#)

    static func scan(_ files: [(path: String, text: String)]) -> [String] {
        var out: [String] = []
        let glasses = files.flatMap { file in file.text.components(separatedBy: "NSGlassEffectView(").dropFirst().map { _ in file.path } }
        if glasses.count != 1 { out.append("\(glasses.count) glass views, not one") }
        for (path, text) in files {
            for word in others where text.contains(word) { out.append("\(path): \(word), a second glass") }
            for match in made.matches(in: text, range: NSRange(text.startIndex..., in: text)) {
                let name = String(text[Range(match.range(at: 1), in: text)!])
                if !text.contains("\(name).contentView = ") { out.append("\(path): \(name) holds no contentView") }
            }
        }
        return out
    }

    func testTheAppHoldsOneGlassWithItsContentInside() throws {
        let files = KitBoundaryTests.swiftFiles(Tree.app)
        XCTAssertGreaterThan(files.count, 2)
        XCTAssertEqual(Self.scan(files), [])
    }

    func testTheGlassScanCatchesWhatItForbids() {
        XCTAssertEqual(Self.scan([("A.swift", "let a = NSGlassEffectView()\nwindow.contentView = a\nx.addSubview(host)")]), ["A.swift: a holds no contentView"])
        XCTAssertEqual(Self.scan([
            ("A.swift", "let a = NSGlassEffectView()\na.contentView = host"),
            ("B.swift", "let b = NSGlassEffectView()\nb.contentView = c\nv.glassEffect(.regular).background(.ultraThinMaterial)\nNSVisualEffectView()"),
        ]), ["2 glass views, not one", "B.swift: glassEffect, a second glass", "B.swift: Material, a second glass", "B.swift: NSVisualEffectView, a second glass"])
        XCTAssertEqual(Self.scan([("A.swift", "Popover(state)")]), ["0 glass views, not one"])
    }
}
