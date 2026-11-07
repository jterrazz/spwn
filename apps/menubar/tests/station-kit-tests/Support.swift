import AppKit
import Foundation
import StationKit
import SwiftUI

enum Tree {
    static let root = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
    static let pages = root.appendingPathComponent("src/ui/testdata/pages")
    static let stream = root.appendingPathComponent("src/ui/testdata/stream/menubar.jsonl")
    static let events = root.appendingPathComponent("src/ui/testdata/stream/events.jsonl")
    static let goldens = URL(fileURLWithPath: #filePath).deletingLastPathComponent().appendingPathComponent("testdata")
    static let sources = root.appendingPathComponent("apps/menubar/sources/station-kit")
    static let app = root.appendingPathComponent("apps/menubar/sources/station-menu-bar")
    static let calls = root.appendingPathComponent("src/remote/api/calls.json")
    static let stories = root.appendingPathComponent("apps/menubar/stories")
    static let storyGoldens = goldens.appendingPathComponent("stories")

    static func fixtures() throws -> [(name: String, data: Data)] {
        try FileManager.default.contentsOfDirectory(at: pages, includingPropertiesForKeys: nil)
            .filter { $0.pathExtension == "json" }
            .sorted { $0.lastPathComponent < $1.lastPathComponent }
            .map { ($0.deletingPathExtension().lastPathComponent, try Data(contentsOf: $0)) }
    }

    static func page(_ name: String) throws -> Page {
        try Page.decode(Data(contentsOf: pages.appendingPathComponent(name + ".json")))
    }
}

struct Bitmap {
    let width: Int
    let height: Int
    let pixels: [UInt8]

    init?(_ image: CGImage) {
        let width = image.width, height = image.height
        self.width = width
        self.height = height
        var buf = [UInt8](repeating: 0, count: width * height * 4)
        let ok = buf.withUnsafeMutableBytes { raw -> Bool in
            guard let ctx = CGContext(data: raw.baseAddress, width: width, height: height, bitsPerComponent: 8, bytesPerRow: width * 4,
                                      space: CGColorSpace(name: CGColorSpace.sRGB)!, bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue) else { return false }
            ctx.draw(image, in: CGRect(x: 0, y: 0, width: width, height: height))
            return true
        }
        guard ok else { return nil }
        pixels = buf
    }

    init?(png: Data) {
        guard let rep = NSBitmapImageRep(data: png), let image = rep.cgImage else { return nil }
        self.init(image)
    }

    func differing(_ other: Bitmap, tolerance: Int = 16) -> Int {
        var n = 0
        for i in stride(from: 0, to: pixels.count, by: 4) {
            for c in 0..<4 where abs(Int(pixels[i + c]) - Int(other.pixels[i + c])) > tolerance {
                n += 1
                break
            }
        }
        return n
    }

    var placeholder: Int {
        var n = 0
        for i in stride(from: 0, to: pixels.count, by: 4)
        where abs(Int(pixels[i]) - 255) <= 4 && abs(Int(pixels[i + 1]) - 204) <= 4 && Int(pixels[i + 2]) <= 4 {
            n += 1
        }
        return n
    }
}

enum Snapshot {
    static let now = ISO8601DateFormatter().date(from: "2026-11-03T10:30:00Z")!

    // value: the popover as a fixture shows it: Home or a page of the menu
    // bar's own, or the switcher or a machine's page pushed over Home, the
    // switcher with infra opened last.
    static func state(_ name: String, _ page: Page) -> PanelState {
        let connection: Connection = name.hasSuffix(".error") || name.hasSuffix(".engine-stopped") ? .stopped : .live
        guard page.id == PageID.switcher || page.id == PageID.machine else { return PanelState(page: page, connection: connection) }
        return PanelState(page: nil, connection: connection, pushed: page, recent: page.id == PageID.switcher ? ["infra"] : [])
    }

    static func popover(_ page: Page) -> Bool { page.id == PageID.home || page.id == PageID.switcher || page.id == PageID.machine }

    @MainActor
    static func view(_ name: String, _ page: Page) -> AnyView {
        if popover(page) { return AnyView(Popover(state(name, page), scrolls: false)) }
        if let own = MenuPage.view(state(name, page)) { return own }
        return AnyView(PageView(page))
    }

    // invariant: a page is drawn over a fixed colourful desk, the vision's,
    // so its glass reads in the PNG; the menu bar icon is drawn bare.
    // In dark the desk is saturated as the vision's glass saturates what
    // it blurs (backdrop-filter saturate 185 %), which the stand-in cannot.
    @MainActor
    static func render(_ view: AnyView, scheme: ColorScheme, desk: Bool = true) -> (png: Data, bitmap: Bitmap)? {
        let themed = view.stationTheme(scheme).environment(\.stationNow, now).environment(\.stationStill, true)
        let content = desk ? AnyView(themed.fixedSize().padding(24).background(Desk(scheme: scheme))) : AnyView(themed)
        let renderer = ImageRenderer(content: content)
        renderer.scale = 2
        guard let image = renderer.cgImage, let bitmap = Bitmap(image) else { return nil }
        let rep = NSBitmapImageRep(cgImage: image)
        guard let png = rep.representation(using: .png, properties: [:]) else { return nil }
        return (png, bitmap)
    }
}

enum Golden {
    static let update = ProcessInfo.processInfo.environment["STATION_UPDATE_GOLDENS"] == "1"

    // invariant: a view is proved by its golden PNG, drawn offscreen: nil
    // when it matches within 0.1 % of its pixels, else what differs.
    @MainActor
    static func check(_ name: String, _ view: AnyView, scheme: ColorScheme, desk: Bool = true, tolerance: Int = 16) throws -> String? {
        let file = "\(name).\(scheme == .dark ? "dark" : "light")"
        let golden = Tree.goldens.appendingPathComponent(file + ".png")
        guard let (png, bitmap) = Snapshot.render(view, scheme: scheme, desk: desk) else { return "\(file): no image rendered" }
        if bitmap.placeholder >= 20 { return "\(file): \(bitmap.placeholder) pixels of the unsupported placeholder" }
        if update {
            try png.write(to: golden)
            return nil
        }
        guard let data = try? Data(contentsOf: golden), let want = Bitmap(png: data) else { return "\(file): no golden (STATION_UPDATE_GOLDENS=1 swift test writes it)" }
        let actual = Tree.goldens.appendingPathComponent(file + ".actual.png")
        guard want.width == bitmap.width, want.height == bitmap.height else {
            try png.write(to: actual)
            return "\(file): \(bitmap.width)x\(bitmap.height), golden \(want.width)x\(want.height)"
        }
        let differing = bitmap.differing(want, tolerance: tolerance)
        if Double(differing) > Double(bitmap.width * bitmap.height) * 0.001 {
            try png.write(to: actual)
            return "\(file): \(differing) pixels differ"
        }
        try? FileManager.default.removeItem(at: actual)
        return nil
    }

    // invariant: a state of the menu bar is proved in both schemes, and
    // in each it stays glass: under 40 % of it painted opaque.
    @MainActor
    static func check(_ name: String, _ view: AnyView) throws -> String? {
        var failures: [String] = []
        for scheme in [ColorScheme.dark, .light] {
            if let f = try check(name, view, scheme: scheme) { failures.append(f) }
            let share = GlassTests.opaque(view, scheme: scheme) ?? 1
            if share >= 0.4 { failures.append("\(name) \(scheme): \(Int(share * 100)) % of it is opaque") }
        }
        return failures.isEmpty ? nil : failures.joined(separator: "; ")
    }
}

struct Desk: View {
    let scheme: ColorScheme

    var body: some View {
        ZStack {
            LinearGradient(colors: Token.Glass.wallBase(scheme), startPoint: UnitPoint(x: 0.3, y: 0), endPoint: UnitPoint(x: 0.7, y: 1))
            ForEach(Array(Token.Glass.wallpaper(scheme).enumerated()), id: \.offset) { _, blob in
                EllipticalGradient(colors: [blob.0, blob.0.opacity(0)], center: blob.1, startRadiusFraction: 0, endRadiusFraction: blob.2)
            }
        }
        .drawingGroup()
        .saturation(scheme == .dark ? 1.85 : 1)
    }
}
