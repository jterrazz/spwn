import AppKit
import Foundation
import StationKit
import XCTest

// invariant: ADR-053: the app's one background is the panel's
// NSGlassEffectView with no tint, whose contentView is a ground painted in
// Token.Glass.ground, the system's window colour at 0.6, which carries no
// hue in either appearance; no tint, second glass, container or material of
// any other kind is drawn in the app, so nothing is glass on glass and the
// panel casts no colour the desk did not give it.
final class AppGlassTests: XCTestCase {
    static let others = ["NSVisualEffectView", "glassEffect", "GlassEffectContainer", "Material", ".containerBackground", "tintColor"]
    static let made = try! NSRegularExpression(pattern: #"let (\w+) = NSGlassEffectView\("#)

    static func scan(_ files: [(path: String, text: String)]) -> [String] {
        var out: [String] = []
        let glasses = files.flatMap { file in file.text.components(separatedBy: "NSGlassEffectView(").dropFirst().map { _ in file.path } }
        if glasses.count != 1 { out.append("\(glasses.count) glasses, not one") }
        if !files.contains(where: { $0.text.contains("Token.Glass.ground.setFill()") }) { out.append("no ground painted in Token.Glass.ground") }
        for (path, text) in files {
            for word in others where text.contains(word) { out.append("\(path): \(word), not the one untinted glass") }
            for match in made.matches(in: text, range: NSRange(text.startIndex..., in: text)) {
                let name = String(text[Range(match.range(at: 1), in: text)!])
                for setting in [".style = .regular", ".contentView = "] where !text.contains(name + setting) {
                    out.append("\(path): \(name) lacks \(name)\(setting)")
                }
            }
        }
        return out
    }

    func testTheAppHoldsOneUntintedGlassOverItsGround() throws {
        let files = KitBoundaryTests.swiftFiles(Tree.app)
        XCTAssertGreaterThan(files.count, 2)
        XCTAssertEqual(Self.scan(files), [])
    }

    func testTheGlassScanCatchesWhatItForbids() {
        let glass = "let a = NSGlassEffectView()\na.style = .regular\na.contentView = ground\nToken.Glass.ground.setFill()"
        XCTAssertEqual(Self.scan([("A.swift", glass)]), [])
        XCTAssertEqual(Self.scan([("A.swift", glass + "\na.tintColor = tint")]), ["A.swift: tintColor, not the one untinted glass"])
        XCTAssertEqual(Self.scan([("A.swift", "let a = NSGlassEffectView()\na.style = .clear\nNSColor.white.setFill()")]), [
            "no ground painted in Token.Glass.ground", "A.swift: a lacks a.style = .regular", "A.swift: a lacks a.contentView = ",
        ])
        XCTAssertEqual(Self.scan([
            ("A.swift", glass),
            ("B.swift", "let b = NSVisualEffectView()\nv.glassEffect(.regular).background(.ultraThinMaterial)\nNSGlassEffectView()"),
        ]), ["2 glasses, not one", "B.swift: NSVisualEffectView, not the one untinted glass", "B.swift: glassEffect, not the one untinted glass", "B.swift: Material, not the one untinted glass"])
        XCTAssertEqual(Self.scan([("A.swift", "Popover(state)")]), ["0 glasses, not one", "no ground painted in Token.Glass.ground"])
    }

    // value: the ground's colour resolved in each appearance, in sRGB bytes.
    static func ground(_ name: NSAppearance.Name) -> [Int] {
        var bytes: [Int] = []
        NSAppearance(named: name)!.performAsCurrentDrawingAppearance {
            let c = Token.Glass.ground.usingColorSpace(.sRGB)!
            bytes = [c.redComponent, c.greenComponent, c.blueComponent].map { Int(($0 * 255).rounded()) }
            bytes.append(Int((c.alphaComponent * 100).rounded()))
        }
        return bytes
    }

    func testTheGroundIsAHueFreeGreyAtSixTenthsInEachAppearance() {
        let light = Self.ground(.aqua), dark = Self.ground(.darkAqua)
        for (scheme, rgba) in [("light", light), ("dark", dark)] {
            XCTAssertEqual(Set(rgba.prefix(3)).count, 1, "\(scheme) ground \(rgba) has a hue")
            XCTAssertEqual(rgba[3], 60, "\(scheme) ground's alpha")
        }
        XCTAssertGreaterThan(light[0], dark[0], "the ground follows the appearance")
    }
}
