import AppKit
import Foundation
import StationKit
import SwiftUI

enum Tree {
    static let root = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
    static let pages = root.appendingPathComponent("apps/engine/ui/testdata/pages")
    static let stream = root.appendingPathComponent("apps/engine/ui/testdata/stream/menubar.jsonl")
    static let events = root.appendingPathComponent("apps/engine/ui/testdata/stream/events.jsonl")
    static let goldens = URL(fileURLWithPath: #filePath).deletingLastPathComponent().appendingPathComponent("testdata")
    static let sources = root.appendingPathComponent("apps/menubar/sources/station-kit")
    static let app = root.appendingPathComponent("apps/menubar/sources/station-menu-bar")
    static let calls = root.appendingPathComponent("apps/engine/remote/api/calls.json")
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

    // value: the pixels as an 8-bit sRGB image, what a golden PNG keeps
    // whatever depth the renderer drew at.
    var image: CGImage? {
        guard let provider = CGDataProvider(data: Data(pixels) as CFData) else { return nil }
        return CGImage(width: width, height: height, bitsPerComponent: 8, bitsPerPixel: 32, bytesPerRow: width * 4,
                       space: CGColorSpace(name: CGColorSpace.sRGB)!, bitmapInfo: CGBitmapInfo(rawValue: CGImageAlphaInfo.premultipliedLast.rawValue),
                       provider: provider, decode: nil, shouldInterpolate: false, intent: .defaultIntent)
    }

    func differing(_ other: Bitmap, tolerance: Int = 0) -> Int {
        var n = 0
        for i in stride(from: 0, to: pixels.count, by: 4) {
            for c in 0..<4 where abs(Int(pixels[i + c]) - Int(other.pixels[i + c])) > tolerance {
                n += 1
                break
            }
        }
        return n
    }

    func widest(_ other: Bitmap) -> Int {
        zip(pixels, other.pixels).reduce(0) { max($0, abs(Int($1.0) - Int($1.1))) }
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
    // bar's own, or a machine's page pushed over Home.
    static func state(_ name: String, _ page: Page) -> PanelState {
        let connection: Connection = name.hasSuffix(".error") || name.hasSuffix(".engine-stopped") || name.hasSuffix(".engine-outdated") ? .stopped : .live
        guard page.id == PageID.machine else { return PanelState(page: page, connection: connection) }
        return PanelState(page: nil, connection: connection, pushed: page)
    }

    static func popover(_ page: Page) -> Bool { page.id == PageID.home || page.id == PageID.machine }

    @MainActor
    static func view(_ name: String, _ page: Page) -> AnyView {
        if popover(page) { return AnyView(Popover(state(name, page), scrolls: false)) }
        return MenuPage.view(state(name, page), scrolls: false) ?? AnyView(Popover(state(name, page), scrolls: false))
    }

    // invariant: a page is drawn over a fixed colourful desk, the vision's,
    // so its glass reads in the PNG; the menu bar icon is drawn bare.
    // In dark the desk is saturated as the vision's glass saturates what
    // it blurs (backdrop-filter saturate 185 %), which the stand-in cannot.
    // The desk is drawn with no offscreen GPU group: one would be drawn by
    // the CPU until Metal is ready, or when it is busy, so its dark goldens
    // would match only on a quiet machine.
    @MainActor
    static func render(_ view: AnyView, scheme: ColorScheme, desk: Bool = true, scale: CGFloat = 2) -> (png: Data, bitmap: Bitmap)? {
        render(view, scheme: scheme, over: desk ? AnyView(Desk(scheme: scheme)) : nil, scale: scale)
    }

    @MainActor
    static func canvas(_ view: AnyView, scheme: ColorScheme, over desk: AnyView?) -> AnyView {
        let themed = view.stationTheme(scheme).environment(\.stationNow, now).environment(\.stationStill, true)
        return desk.map { AnyView(themed.fixedSize().padding(24).background($0)) } ?? AnyView(themed)
    }

    @MainActor
    static func render(_ view: AnyView, scheme: ColorScheme, over desk: AnyView?, scale: CGFloat = 2) -> (png: Data, bitmap: Bitmap)? {
        let renderer = ImageRenderer(content: canvas(view, scheme: scheme, over: desk))
        renderer.scale = scale
        guard let image = renderer.cgImage, let bitmap = Bitmap(image), let flat = bitmap.image else { return nil }
        let rep = NSBitmapImageRep(cgImage: flat)
        guard let png = rep.representation(using: .png, properties: [:]) else { return nil }
        return (png, bitmap)
    }
}

enum Expectation {
    // invariant: STATION_EXPECT_GOLDENS names a file of golden paths, relative to the
    // repository root, one per line: only those goldens are rewritten, and only when
    // they differ; every other golden is compared exactly as without it.
    static func paths(_ file: String?) -> Set<String> {
        guard let file, let text = try? String(contentsOfFile: file, encoding: .utf8) else { return [] }
        return Set(text.split(separator: "\n").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty && !$0.hasPrefix("#") })
    }

    static func listed(_ golden: URL, in set: Set<String>) -> Bool {
        let root = Tree.root.standardizedFileURL.path + "/"
        let path = golden.standardizedFileURL.path
        return path.hasPrefix(root) && set.contains(String(path.dropFirst(root.count)))
    }
}

enum Golden {
    static let update = ProcessInfo.processInfo.environment["STATION_UPDATE_GOLDENS"] == "1"
    static let expected: Set<String> = Expectation.paths(ProcessInfo.processInfo.environment["STATION_EXPECT_GOLDENS"])

    // invariant: a view is proved by its golden PNG, drawn offscreen: nil
    // when its decoded pixels are the golden's, every channel of every
    // pixel, else what differs.
    @MainActor
    static func check(_ name: String, _ view: AnyView, scheme: ColorScheme, desk: Bool = true, scale: CGFloat = 2) throws -> String? {
        let file = "\(name).\(scheme == .dark ? "dark" : "light")"
        return try match(Snapshot.render(view, scheme: scheme, desk: desk, scale: scale), file: file, in: Tree.goldens)
    }

    static func gap(_ got: Bitmap, _ want: Bitmap, file: String) -> String? {
        if want.width != got.width || want.height != got.height { return "\(file): \(got.width)x\(got.height), golden \(want.width)x\(want.height)" }
        if got.pixels == want.pixels { return nil }
        return "\(file): \(got.differing(want)) pixels differ, by up to \(got.widest(want)) in a channel"
    }

    // invariant: a golden is written only when its pixels change, so a
    // drawing that did not move leaves its file's bytes alone.
    static func match(_ rendered: (png: Data, bitmap: Bitmap)?, file: String, in folder: URL) throws -> String? {
        let golden = folder.appendingPathComponent(file + ".png")
        guard let (png, bitmap) = rendered else { return "\(file): no image rendered" }
        if bitmap.placeholder >= 20 { return "\(file): \(bitmap.placeholder) pixels of the unsupported placeholder" }
        guard let got = Bitmap(png: png) else { return "\(file): the drawing's PNG does not decode" }
        let listed = Expectation.listed(golden, in: expected)
        let actual = folder.appendingPathComponent(file + ".actual.png")
        let failure: String?
        if let data = try? Data(contentsOf: golden), let want = Bitmap(png: data) {
            failure = gap(got, want, file: file)
        } else {
            failure = "\(file): no golden (STATION_UPDATE_GOLDENS=1 swift test writes it)"
        }
        guard let failure else {
            try? FileManager.default.removeItem(at: actual)
            return listed ? "\(file): listed in STATION_EXPECT_GOLDENS, but it did not change" : nil
        }
        if update || listed {
            try png.write(to: golden)
            try? FileManager.default.removeItem(at: actual)
            return nil
        }
        try png.write(to: actual)
        return failure
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
            LinearGradient(colors: Token.Glass.wallBase(scheme), startPoint: Token.Glass.wallStart, endPoint: Token.Glass.wallEnd)
            ForEach(Array(Token.Glass.wallpaper(scheme).enumerated()), id: \.offset) { _, blob in
                EllipticalGradient(colors: [blob.0, blob.0.opacity(0)], center: blob.1, startRadiusFraction: 0, endRadiusFraction: blob.2)
            }
        }
        .saturation(scheme == .dark ? 1.85 : 1)
    }
}
