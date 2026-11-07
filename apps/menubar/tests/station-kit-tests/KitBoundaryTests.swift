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
    static let sizeKey = try! NSRegularExpression(pattern: #"\b(spacing|width|height|minWidth|minHeight|maxWidth|maxHeight|idealWidth|idealHeight|minLength|cornerRadius|lineWidth|radius|size|top|leading|bottom|trailing|x|y|duration|dash|bounce|response): "#)
    static let sizeCall = try! NSRegularExpression(pattern: #"\.(padding|offset|opacity|tracking|kerning|blur|scaleEffect|ratio|brightness|saturation|cornerRadius|lineSpacing|minimumScaleFactor|rotationEffect|shadow|timingCurve|easeOut|easeIn|easeInOut|linear|spring)\("#)
    static let sizeDeclared = try! NSRegularExpression(pattern: #"\b(?:let|var) \w+: \[?CGFloat\]?"#)
    static let number = try! NSRegularExpression(pattern: #"(?<![\w.\\])(\d+\.\d+|\.\d+|\d+)(?![\w.])"#)
    static let doubled = try! NSRegularExpression(pattern: #"[*/] ?2(?![\w.])|(?<![\w.])2 ?\*"#)
    static let style = try! NSRegularExpression(pattern: #"Color\.(?!clear\b)[a-z]|color: \.|\(\.(primary|secondary|tertiary|quaternary|separator|fill|tint|black|white)\b"#)
    static let motion = try! NSRegularExpression(pattern: #"withAnimation \{|withAnimation\(\.|\.animation\(\.|Animation\."#)

    // invariant: a view draws with the tokens only: a size, a share, a
    // style or an animation of its own is a literal Token does not name;
    // zero, a token doubled or halved, an opacity of one and a chart's
    // plotted value are not, nor a glyph's drawn points, which are its
    // geometry on its own grid.
    static func literals(_ line: String) -> [String] {
        let trimmed = line.trimmingCharacters(in: .whitespaces)
        guard !trimmed.hasPrefix("//") else { return [] }
        let text = doubled.stringByReplacingMatches(in: line, range: NSRange(line.startIndex..., in: line), withTemplate: "")
        let chars = Array(text)
        func argument(from start: Int, comma: Bool) -> String {
            var depth = 0, i = start
            while i < chars.count {
                let c = chars[i]
                if "([{".contains(c) { depth += 1 }
                if ")]}".contains(c) { if depth == 0 { break }; depth -= 1 }
                if comma && c == "," && depth == 0 { break }
                i += 1
            }
            return String(chars[start..<i])
        }
        func opened(_ re: NSRegularExpression, comma: Bool) -> [(String, String)] {
            re.matches(in: text, range: NSRange(text.startIndex..., in: text)).map {
                let end = text.distance(from: text.startIndex, to: Range($0.range, in: text)!.upperBound)
                return (String(text[Range($0.range(at: 1), in: text)!]), argument(from: end, comma: comma))
            }
        }
        func numbers(_ arg: String) -> [Double] {
            number.matches(in: arg, range: NSRange(arg.startIndex..., in: arg)).compactMap { Double(String(arg[Range($0.range, in: arg)!])) }
        }
        let neutral: Set<String> = ["opacity", "ratio", "scaleEffect", "minimumScaleFactor"]
        var out: [String] = []
        for (key, arg) in opened(sizeKey, comma: true) where !arg.hasPrefix(".value(") && numbers(arg).contains(where: { $0 != 0 }) { out.append("a size literal (\(key))") }
        for (call, arg) in opened(sizeCall, comma: false) {
            let kept = numbers(arg).filter { $0 != 0 && !($0 == 1 && neutral.contains(call)) }
            if !kept.isEmpty { out.append("a size literal (\(call))") }
        }
        let range = NSRange(text.startIndex..., in: text)
        if let declared = sizeDeclared.firstMatch(in: text, range: range) {
            let rest = String(text[Range(declared.range, in: text)!.upperBound...])
            if numbers(rest).contains(where: { $0 != 0 && $0 != 1 }) { out.append("a size defined outside Token") }
        }
        if numbers(text).contains(where: { $0 > 0 && $0 < 1 }) { out.append("a share literal") }
        if style.firstMatch(in: text, range: range) != nil { out.append("a style outside Token") }
        if motion.firstMatch(in: text, range: range) != nil { out.append("an animation outside Token") }
        return out
    }

    // invariant: the Swift side of the kit's boundary: colours, symbol names
    // and glyphs live in Token only, no AppKit-backed control the offscreen
    // renderer cannot draw, a Mac's symbol named only by Token and keyed by
    // its lid (DECISIONS E1/E2: a capability, never the model), and Page and
    // Engine stay Foundation.
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
            for model in models where text.components(separatedBy: "\n").contains(where: { $0.contains(model) && !(token && $0.contains("lid")) }) {
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

    static let drawnByTheKit = ["Popover(", "PageView(", "MenuBarIconView(", "MenuPage.", "HomeContent(", "MachineContent(", "StoryView("]
    static let engine = ["Client.", "UnixSocket.", "EnginePath.", "ActionRunner(", "Navigation(", "Terminal."]

    static func scanCatalog(_ files: [(path: String, text: String)]) -> [Finding] {
        var out: [Finding] = []
        for (path, text) in files {
            if text.contains("Color(") || text.range(of: #"Color\.(red|green|blue|yellow|orange|black|white|gray)"#, options: .regularExpression) != nil {
                out.append(Finding(file: path, rule: "a colour outside Token"))
            }
            for word in drawnByTheKit where text.contains(word) {
                out.append(Finding(file: path, rule: "\(word.dropLast()) drawn by the Catalog, not through StoryStage"))
            }
            for word in engine where text.contains(word) {
                out.append(Finding(file: path, rule: "\(word.dropLast()): the Catalog reaches no engine"))
            }
        }
        return out
    }

    static func swiftFiles(_ root: URL) -> [(path: String, text: String)] {
        FileManager.default.enumerator(at: root, includingPropertiesForKeys: nil)!.compactMap { $0 as? URL }
            .filter { $0.pathExtension == "swift" }
            .map { (String($0.path.dropFirst(root.path.count + 1)), try! String(contentsOf: $0, encoding: .utf8)) }
    }

    func testTheCatalogDrawsOnlyThroughTheKitAndIsNeverBundled() throws {
        let files = Self.swiftFiles(Tree.root.appendingPathComponent("apps/menubar/sources/station-catalog"))
        XCTAssertGreaterThan(files.count, 2)
        XCTAssertEqual(Self.scanCatalog(files), [])
        let package = try String(contentsOf: Tree.root.appendingPathComponent("apps/menubar/Package.swift"), encoding: .utf8)
        let app = try XCTUnwrap(package.split(separator: "\n").first { $0.contains("name: \"StationMenuBar\"") })
        XCTAssertFalse(app.contains("StationCatalog"), "Station.app links the Catalog")
        let bundle = try String(contentsOf: Tree.root.appendingPathComponent("apps/menubar/scripts/build.sh"), encoding: .utf8)
        XCTAssertFalse(bundle.contains("StationCatalog"), "the bundle carries the Catalog")
    }

    func testTheCatalogScanCatchesWhatItForbids() {
        let caught = Self.scanCatalog([
            ("Canvas.swift", "Popover(state).background(Color.blue)"),
            ("Live.swift", "await Client.call(path: socket, name: name)"),
        ]).map(\.rule)
        XCTAssertEqual(caught, ["a colour outside Token", "Popover drawn by the Catalog, not through StoryStage", "Client: the Catalog reaches no engine"])
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
            ("token/AtomTokens.swift", "static func mac(lid: Bool) -> String { lid ? \"laptopcomputer\" : \"macmini\" }"),
            ("template/Panel.swift", "x.background(palette.surface).glassEffect(.regular)"),
            ("molecule/Field.swift", "x.background(.quinary).background(.ultraThinMaterial)"),
            ("template/Many.swift", "a.background(Token.Fill.tile(scheme))\nStationButton(role: .primary)\nStationButton(role: x ? .primary : .plain)"),
            ("molecule/Surface.swift", "Toggle().toggleStyle(.switch); x.background(.quinary)"),
            ("atom/Ring.swift", "VStack(spacing: 6) { x.frame(width: Token.Tile.dot + 2).opacity(on ? 1 : 0.4).foregroundStyle(.secondary) }\n// spacing: 4\n.padding(Token.Gap.small * 2).animation(.snappy, value: on)"),
            ("engine/Clock.swift", "x.frame(width: 4)"),
            ("template/Awake.swift", "Spacer(minLength: 6).cornerRadius(7).lineSpacing(3).opacity(max(0, 1) > 0 ? 1 : 0.3)\nx.padding(Token.Gap.small * 3).frame(width: max(w, 4))"),
            ("molecule/Layout.swift", "static let charWidth: CGFloat = 6.6\nstatic var room: CGFloat { max(h ?? 800, Token.Panel.least) }\nguard !waiting else { return 0.25 }"),
            ("atom/Glyph.swift", "path.move(to: p(3, 4.5)); x.padding(Token.Gap.small * 2).padding(-Token.Stroke.open / 2)\nLineMark(x: .value(\"t\", i), y: .value(\"v\", min(p, 100))); zip(a.map(\\.0), b.map(\\.1))\nlet slack: CGFloat = 1"),
        ]).map(\.rule)
        XCTAssertEqual(caught, [
            "a colour outside Token", "ProgressView is not drawn offscreen",
            "a symbol name outside Token", "a glyph outside Token",
            "import SwiftUI: Page and Engine import Foundation only",
            "macmini draws a Mac's model, which no capability reports",
            "glassEffect in the content", "a painted background",
            "Material in the content", ".quinary outside the kit's surface",
            "Token.Fill.tile outside the kit's surface", "2 primary actions on one page",
            "a size literal (spacing)", "a size literal (width)", "a size literal (opacity)", "a share literal", "a style outside Token", "an animation outside Token",
            "a size literal (minLength)", "a size literal (cornerRadius)", "a size literal (lineSpacing)", "a size literal (opacity)", "a share literal", "a size literal (width)", "a size literal (padding)",
            "a size defined outside Token", "a share literal",
        ])
    }
}
