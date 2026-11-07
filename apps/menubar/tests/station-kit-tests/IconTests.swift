import StationKit
import SwiftUI
import XCTest

@MainActor
final class IconTests: XCTestCase {
    func icon(_ fixture: String, _ connection: Connection = .live, sparkline: Bool = false) throws -> MenuBarIcon {
        MenuBarIcon.of(connection: connection, page: try Tree.page(fixture), sparkline: sparkline)
    }

    func drawn(_ name: String, _ icon: MenuBarIcon) throws {
        for scheme in [ColorScheme.dark, .light] {
            XCTAssertNil(try Golden.check("icon." + name, AnyView(StatusItem(icon, scheme: scheme)), scheme: scheme, desk: false))
        }
    }

    // invariant: MB-01: nothing needs you on This Mac alone: the Orbit
    // alone.
    func testIconPlain() throws {
        let icon = try icon("menubar.home.one-mac")
        XCTAssertEqual(icon, MenuBarIcon(.plain))
        XCTAssertEqual(icon.label, "station: nothing needs you")
        XCTAssertEqual(icon.symbol, Token.Icon.orbit)
        try drawn("plain", icon)
    }

    // invariant: MB-02: five findings and no server down: the dot cut into
    // the Orbit's corner, the count said by its label.
    func testIconNeedsYouDot() throws {
        let icon = try icon("menubar.home.two-macs")
        XCTAssertEqual(icon, MenuBarIcon(.needs(5)))
        XCTAssertEqual(icon.label, "station: 5 things need you")
        XCTAssertEqual(icon.symbol, Token.Icon.badged)
        try drawn("needs", icon)
    }

    // invariant: MB-03: infra in a loop on the mini: the Orbit slashed,
    // whatever the count beside it.
    func testIconRedServerDown() throws {
        let page = try Tree.page("menubar.home.infra-down")
        XCTAssertEqual(page.badge?.needs, 5)
        let icon = try icon("menubar.home.infra-down")
        XCTAssertEqual(icon, MenuBarIcon(.down))
        XCTAssertEqual(icon.label, "station: a server is down")
        XCTAssertEqual(icon.symbol, Token.Icon.slashed)
        try drawn("down", icon)
    }

    // invariant: MB-04: the CPU line is the owner's option, This Mac's CPU
    // from the page's strip, beside the symbol and its badge.
    func testIconCPUSpark() throws {
        let spark = try XCTUnwrap(Tree.page("menubar.home.two-macs").strip?.spark)
        XCTAssertFalse(spark.isEmpty)
        let icon = try icon("menubar.home.two-macs", sparkline: true)
        XCTAssertEqual(icon, MenuBarIcon(.needs(5), spark: spark))
        try drawn("spark", icon)
        XCTAssertEqual(try self.icon("menubar.home.engine-stopped", .stopped, sparkline: true).spark, nil)
    }

    // invariant: EN-10, MB-11: the engine stopped dims the slashed Orbit
    // and says so; a connection still being made says that, with no badge either way.
    func testIconEngineStopped() throws {
        let stopped = try icon("menubar.home.engine-stopped", .stopped)
        XCTAssertEqual(stopped, MenuBarIcon(.stopped))
        XCTAssertEqual(stopped.label, "station: engine stopped")
        XCTAssertEqual(stopped.symbol, Token.Icon.slashed)
        XCTAssertEqual(try icon("menubar.home.two-macs", .notInstalled), MenuBarIcon(.stopped))
        XCTAssertEqual(try icon("menubar.home.infra-down", .connecting).label, "station: connecting")
        XCTAssertEqual(MenuBarIcon.of(connection: .live, page: try Tree.page("menubar.loading")), MenuBarIcon(.plain))
        try drawn("stopped", stopped)
    }

    // invariant: a connection being made dims the Mac and its ring and
    // keeps the two machines lit, with no badge.
    func testIconConnecting() throws {
        let icon = try icon("menubar.home.two-macs", .connecting)
        XCTAssertEqual(icon, MenuBarIcon(.connecting))
        XCTAssertEqual(icon.symbol, Token.Icon.orbit)
        try drawn("connecting", icon)
    }

    // invariant: every state is a template image with no colour of its
    // own: macOS paints it for the bar's appearance and tints it while the
    // panel is open, and both schemes draw the same alpha.
    func testEveryStateIsATemplate() throws {
        let marks: [MenuBarIcon.Mark] = [.plain, .needs(5), .needs(120), .down, .stopped, .connecting]
        for mark in marks {
            let icon = MenuBarIcon(mark, spark: [10, 60, 30])
            XCTAssertTrue(icon.nsImage(scheme: .dark).isTemplate, "\(mark)")
            let dark = try XCTUnwrap(icon.image(scheme: .dark).flatMap(Bitmap.init))
            let light = try XCTUnwrap(icon.image(scheme: .light).flatMap(Bitmap.init))
            XCTAssertEqual(dark.width, light.width)
            XCTAssertEqual(dark.pixels.count, light.pixels.count, "\(mark)")
            let apart = stride(from: 3, to: min(dark.pixels.count, light.pixels.count), by: 4).map { abs(Int(dark.pixels[$0]) - Int(light.pixels[$0])) }.max() ?? 0
            XCTAssertLessThanOrEqual(apart, 2, "\(mark)")
        }
    }

    // invariant: the mark is the Orbit, a Mac inside a ring, never a
    // bubble again: across its middle row and its middle column, at 1x and
    // at 2x, the ink crosses the ring, the Mac, the hole inside it, the Mac
    // and the ring, four strokes with its centre left empty.
    func testTheMarkIsAMacInsideARing() throws {
        for scale in [1, 2] as [CGFloat] {
            let image = try XCTUnwrap(MenuBarIcon(.plain).image(scheme: .light, scale: scale))
            let ink = try XCTUnwrap(Ink(image))
            func strokes(_ at: (Int) -> Bool, _ n: Int) -> Int {
                var count = 0, was = false
                for i in 0..<n {
                    let now = at(i)
                    if now && !was { count += 1 }
                    was = now
                }
                return count
            }
            let box = ink.box, cx = box.midX, cy = box.midY
            XCTAssertEqual(strokes({ ink.inked(box.minX + $0, cy) }, box.width), 4, "middle row at \(scale)x")
            XCTAssertEqual(strokes({ ink.inked(cx, box.minY + $0) }, box.height), 4, "middle column at \(scale)x")
            XCTAssertFalse(ink.inked(cx, cy), "the Mac's hole at \(scale)x")
        }
    }

    // invariant: the Orbit has the strokes of the bar's own symbols: drawn
    // as the status bar draws Spotlight's magnifying glass and a circle,
    // 13 pt medium, its ring is as thick within half a pixel at 2x and as
    // tall within one, and Wi-Fi's shorter fan stays shorter.
    func testTheOrbitHasTheStrokesOfTheBarsSymbols() throws {
        let orbit = try XCTUnwrap(Ink(MenuBarIcon(.plain).image(scheme: .light)))
        for name in ["magnifyingglass", "circle"] {
            let native = try XCTUnwrap(Ink(Self.native(name)))
            XCTAssertEqual(orbit.stroke, native.stroke, accuracy: 0.5, name)
            XCTAssertEqual(Double(orbit.box.height), Double(native.box.height), accuracy: 1, name)
        }
        let wifi = try XCTUnwrap(Ink(Self.native("wifi")))
        XCTAssertLessThan(wifi.box.height, orbit.box.height)
        XCTAssertGreaterThan(orbit.stroke, 2)
        XCTAssertLessThan(orbit.stroke, 3.2)
    }

    static func native(_ name: String) -> CGImage? {
        let renderer = ImageRenderer(content: Image(systemName: name)
            .font(.system(size: Token.Icon.size, weight: Token.Icon.weight))
            .imageScale(.medium)
            .foregroundStyle(Token.Icon.ink(.light)))
        renderer.scale = 2
        return renderer.cgImage
    }

    // invariant: beside the bar's own symbols, on a light bar and on a dark
    // one, the Orbit in each of its states reads as one of them.
    func testTheOrbitBesideTheBarsSymbols() throws {
        for scheme in [ColorScheme.dark, .light] {
            let ink = Token.Icon.ink(scheme)
            let marks: [MenuBarIcon.Mark] = [.plain, .needs(5), .down, .stopped, .connecting]
            let row = HStack(spacing: 10) {
                ForEach(["magnifyingglass", "wifi", "switch.2"], id: \.self) { name in
                    Image(systemName: name).font(.system(size: Token.Icon.size, weight: Token.Icon.weight)).foregroundStyle(ink)
                }
                ForEach(marks.indices, id: \.self) { i in StatusItem(MenuBarIcon(marks[i]), scheme: scheme) }
            }
            .padding(.horizontal, 10)
            .frame(height: 24)
            .background(scheme == .dark ? Color(white: 0.12) : Color(white: 0.94))
            XCTAssertNil(try Golden.check("icon.beside-native", AnyView(row), scheme: scheme, desk: false))
        }
    }

    func testABusyItemTurnsItsMarkOneFrameAtATime() throws {
        let busy = MenuBarIcon(.plain, busy: true)
        func bitmap(_ turn: Double) throws -> Bitmap { try XCTUnwrap(busy.image(scheme: .dark, turn: turn).flatMap(Bitmap.init)) }
        XCTAssertEqual(try bitmap(0.25).pixels, try bitmap(0.25).pixels)
        XCTAssertNotEqual(try bitmap(0).pixels, try bitmap(0.25).pixels)
        XCTAssertNotEqual(try XCTUnwrap(MenuBarIcon(.plain).image(scheme: .dark)).width, try bitmap(0).width)
        XCTAssertEqual(busy.nsImage(scheme: .dark, turn: 0.5).representations.count, 3)
        XCTAssertEqual(Token.Busy.frames, 8)
        XCTAssertEqual(Token.Busy.turn(frame: 2), 0.25)
        XCTAssertEqual(Token.Busy.turn(frame: 8), 0)
        for scheme in [ColorScheme.dark, .light] {
            XCTAssertNil(try Golden.check("icon.busy", AnyView(StatusItem(busy, scheme: scheme, turn: 0.25)), scheme: scheme, desk: false))
        }
    }
}

// value: a drawn symbol's ink: its box, and its stroke measured down the
// column through its topmost ink, the top of a ring or of a lens.
struct Ink {
    let bitmap: Bitmap
    let box: (minX: Int, minY: Int, width: Int, height: Int, midX: Int, midY: Int)

    init?(_ image: CGImage?) {
        guard let image, let bitmap = Bitmap(image) else { return nil }
        self.bitmap = bitmap
        var minX = bitmap.width, maxX = -1, minY = bitmap.height, maxY = -1
        for y in 0..<bitmap.height {
            for x in 0..<bitmap.width where bitmap.pixels[(y * bitmap.width + x) * 4 + 3] > 127 {
                minX = min(minX, x); maxX = max(maxX, x); minY = min(minY, y); maxY = max(maxY, y)
            }
        }
        guard maxX >= minX else { return nil }
        box = (minX, minY, maxX - minX + 1, maxY - minY + 1, (minX + maxX) / 2, (minY + maxY) / 2)
    }

    func alpha(_ x: Int, _ y: Int) -> Double { Double(bitmap.pixels[(y * bitmap.width + x) * 4 + 3]) / 255 }

    func inked(_ x: Int, _ y: Int) -> Bool { alpha(x, y) > 110 / 255 }

    var stroke: Double {
        let top = (0..<bitmap.width).filter { inked($0, box.minY) }
        guard let first = top.first, let last = top.last else { return 0 }
        let x = (first + last) / 2
        var sum = 0.0, y = 0
        while y < bitmap.height && alpha(x, y) < 0.03 { y += 1 }
        while y < bitmap.height && alpha(x, y) >= 0.03 { sum += alpha(x, y); y += 1 }
        return sum
    }

}
