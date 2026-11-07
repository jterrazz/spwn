import XCTest

final class KitBoundaryTests: XCTestCase {
    struct Finding: Equatable, CustomStringConvertible {
        let file: String
        let rule: String
        var description: String { "\(file): \(rule)" }
    }

    static let forbidden = ["ProgressView", "Gauge(", "List(", "List {", "WindowGroup", "NSWindow(", "Settings {"]
    static let models = ["\"laptopcomputer", "\"macmini", "\"macbook", "\"macpro", "\"imac", "\"macstudio"]
    // value: L50d, Apple's rules: the MenuBarExtra's window is the only
    // glass, so no view of the kit is glass or a material; L50e: the
    // vision's tiles are translucent fills drawn by molecule/Surface only.
    static let glass = ["glassEffect", "GlassEffectContainer", "Material", ".containerBackground"]
    static let surface = "molecule/Surface.swift"
    static let surfaceOnly = [".quinary", "Token.Fill.tile", ".glassProminent", ".buttonStyle(.bordered", ".toggleStyle(.switch)"]
    static let glyphs: [Character] = ["→", "·", "…", "⏻", "⚡", "●", "◐", "○", "✗", "◌"]

    static let views = ["atom/", "molecule/", "organism/", "template/", "catalog/"]
    static let sizeKey = try! NSRegularExpression(pattern: #"\b(spacing|width|height|minWidth|minHeight|maxWidth|maxHeight|idealWidth|idealHeight|cornerRadius|lineWidth|radius|size|top|leading|bottom|trailing|x|y|duration|dash|bounce|response): ([^,)]*)"#)
    static let sizeCall = try! NSRegularExpression(pattern: #"\.(padding|offset|opacity|tracking|kerning|blur|scaleEffect|ratio|brightness|saturation|timingCurve|easeOut|easeIn|easeInOut|linear|spring)\(([^)]*)\)"#)
    static let number = try! NSRegularExpression(pattern: #"(?<![\w.])(\d+\.\d+|\.\d+|\d+)(?![\w.])"#)
    static let scaled = try! NSRegularExpression(pattern: #"[*/] ?\d+(\.\d+)?|(?<![\w.])\d+(\.\d+)? ?\*"#)
    static let style = try! NSRegularExpression(pattern: #"Color\.(?!clear\b)[a-z]|color: \.|\(\.(primary|secondary|tertiary|quaternary|separator|fill|tint|black|white)\b"#)
    static let motion = try! NSRegularExpression(pattern: #"withAnimation \{|withAnimation\(\.|\.animation\(\.|Animation\."#)

    // invariant: a view draws with the tokens only: a size, a share, a
    // style or an animation of its own is a literal Token does not name;
    // zero, a token scaled, and an opacity of one are not.
    static func literals(_ line: String) -> [String] {
        let trimmed = line.trimmingCharacters(in: .whitespaces)
        guard !trimmed.hasPrefix("//") else { return [] }
        let text = scaled.stringByReplacingMatches(in: line, range: NSRange(line.startIndex..., in: line), withTemplate: "")
        func groups(_ re: NSRegularExpression) -> [(String, String)] {
            re.matches(in: text, range: NSRange(text.startIndex..., in: text)).map {
                (String(text[Range($0.range(at: 1), in: text)!]), String(text[Range($0.range(at: 2), in: text)!]))
            }
        }
        func numbers(_ arg: String) -> [Double] {
            number.matches(in: arg, range: NSRange(arg.startIndex..., in: arg)).compactMap { Double(String(arg[Range($0.range, in: arg)!])) }
        }
        var out: [String] = []
        for (key, arg) in groups(sizeKey) where numbers(arg).contains(where: { $0 != 0 }) { out.append("a size literal (\(key))") }
        for (call, arg) in groups(sizeCall) {
            let kept = numbers(arg).filter { $0 != 0 && !(($0 == 1) && (call == "opacity" || call == "ratio")) }
            if !kept.isEmpty { out.append("a size literal (\(call))") }
        }
        let range = NSRange(text.startIndex..., in: text)
        if style.firstMatch(in: text, range: range) != nil { out.append("a style outside Token") }
        if motion.firstMatch(in: text, range: range) != nil { out.append("an animation outside Token") }
        return out
    }

    // invariant: the Swift side of the kit's boundary: colours, symbol names
    // and glyphs live in Token only, no AppKit-backed control the offscreen
    // renderer cannot draw, every Mac drawn with one symbol (DECISIONS E1/E2:
    // the page reports no model), and Page and Engine stay Foundation.
    static func scan(_ files: [(path: String, text: String)]) -> [Finding] {
        var out: [Finding] = []
        for (path, text) in files {
            let token = path.hasPrefix("token/")
            if !token {
                if text.contains("Color(") || text.range(of: #"Color\.(red|green|blue|yellow|orange|black|white|gray)"#, options: .regularExpression) != nil {
                    out.append(Finding(file: path, rule: "a colour outside Token"))
                }
                if text.contains("systemName: \"") {
                    out.append(Finding(file: path, rule: "a symbol name outside Token"))
                }
                if text.contains(where: { glyphs.contains($0) }) {
                    out.append(Finding(file: path, rule: "a glyph outside Token"))
                }
            }
            if views.contains(where: path.hasPrefix) {
                for line in text.components(separatedBy: "\n") {
                    for rule in literals(line) where !out.contains(Finding(file: path, rule: rule)) { out.append(Finding(file: path, rule: rule)) }
                }
            }
            for model in models where text.contains(model) {
                out.append(Finding(file: path, rule: "\(model.dropFirst()) draws a Mac's model, which no capability reports"))
            }
            for word in forbidden where text.contains(word) {
                out.append(Finding(file: path, rule: "\(word) is not drawn offscreen"))
            }
            for word in glass where text.contains(word) {
                out.append(Finding(file: path, rule: "\(word) in the content"))
            }
            if path != surface {
                for word in surfaceOnly where text.contains(word) {
                    out.append(Finding(file: path, rule: "\(word) outside the kit's surface"))
                }
            }
            if text.range(of: #"\.background\(palette\.(?!qrLight)"#, options: .regularExpression) != nil {
                out.append(Finding(file: path, rule: "a painted background"))
            }
            // invariant: a page holds one primary action at most.
            if path.hasPrefix("template/") {
                let primaries = text.components(separatedBy: "\n").filter { $0.range(of: #"role: [^,)]*\.primary"#, options: .regularExpression) != nil }.count
                if primaries > 1 { out.append(Finding(file: path, rule: "\(primaries) primary actions on one page")) }
            }
            if path.hasPrefix("page/") || path.hasPrefix("engine/") {
                for line in text.split(separator: "\n") where line.hasPrefix("import ") && line != "import Foundation" {
                    out.append(Finding(file: path, rule: "\(line): Page and Engine import Foundation only"))
                }
            }
        }
        return out
    }

    func testTheKitKeepsItsBoundary() throws {
        let root = Tree.sources
        let files = FileManager.default.enumerator(at: root, includingPropertiesForKeys: nil)!.compactMap { $0 as? URL }
            .filter { $0.pathExtension == "swift" }
            .map { (String($0.path.dropFirst(root.path.count + 1)), try! String(contentsOf: $0, encoding: .utf8)) }
        XCTAssertGreaterThan(files.count, 5)
        XCTAssertEqual(Self.scan(files), [])
    }

    func testTheScanCatchesWhatItForbids() {
        let caught = Self.scan([
            ("atom/Dot.swift", "import SwiftUI\nstruct D: View { var body: some View { ProgressView().foregroundStyle(Color(red: 1, green: 0, blue: 0)) } }"),
            ("molecule/Row.swift", "Image(systemName: \"circle\"); Text(\"→\")"),
            ("page/Page.swift", "import SwiftUI"),
            ("token/Token.swift", "static let mini = \"macmini\""),
            ("template/Panel.swift", "x.background(palette.surface).glassEffect(.regular)"),
            ("molecule/Field.swift", "x.background(.quinary).background(.ultraThinMaterial)"),
            ("template/Many.swift", "a.background(Token.Fill.tile(scheme))\nStationButton(role: .primary)\nStationButton(role: x ? .primary : .plain)"),
            ("molecule/Surface.swift", "Toggle().toggleStyle(.switch); x.background(.quinary)"),
            ("atom/Ring.swift", "VStack(spacing: 6) { x.frame(width: Token.Tile.dot + 2).opacity(on ? 1 : 0.4).foregroundStyle(.secondary) }\n// spacing: 4\n.padding(Token.Gap.small * 2).animation(.snappy, value: on)"),
            ("engine/Clock.swift", "x.frame(width: 4)"),
        ]).map(\.rule)
        XCTAssertEqual(caught, [
            "a colour outside Token", "ProgressView is not drawn offscreen",
            "a symbol name outside Token", "a glyph outside Token",
            "import SwiftUI: Page and Engine import Foundation only",
            "macmini draws a Mac's model, which no capability reports",
            "glassEffect in the content", "a painted background",
            "Material in the content", ".quinary outside the kit's surface",
            "Token.Fill.tile outside the kit's surface", "2 primary actions on one page",
            "a size literal (spacing)", "a size literal (width)", "a size literal (opacity)", "a style outside Token", "an animation outside Token",
        ])
    }
}
