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
        XCTAssertEqual(icon, MenuBarIcon(.plain, said: "all machines are fine"))
        XCTAssertEqual(icon.label, "all machines are fine")
        XCTAssertEqual(icon.symbol, Token.Icon.orbit)
        try drawn("plain", icon)
    }

    // invariant: MB-02: five findings of upkeep and no server down: the
    // Orbit alone, since upkeep never counts.
    func testUpkeepNeverCounts() throws {
        XCTAssertEqual(try Tree.page("menubar.home.two-macs").badge?.needs, 0)
        XCTAssertEqual(try icon("menubar.home.two-macs"), MenuBarIcon(.plain, said: "all machines are fine"))
    }

    // invariant: MB-03: infra in a loop on the mini: the Orbit slashed
    // beside the count, never red, and the label names the server.
    func testIconServerDownIsSlashed() throws {
        let icon = try icon("menubar.home.infra-down")
        XCTAssertEqual(icon, MenuBarIcon(.down, count: 1, said: "station, infra down"))
        XCTAssertEqual(icon.symbol, Token.Icon.slashed)
        XCTAssertTrue(icon.nsImage().isTemplate)
    }

    // invariant: what needs you cuts the badge out of the Orbit beside its
    // count; a turning mark or a slash takes its place.
    func testNeedsYouCutsTheBadgeOut() {
        XCTAssertEqual(MenuBarIcon(.plain, count: 2).symbol, Token.Icon.badged)
        XCTAssertEqual(MenuBarIcon(.plain).symbol, Token.Icon.orbit)
        XCTAssertEqual(MenuBarIcon(.down, count: 2).symbol, Token.Icon.slashed)
        XCTAssertEqual(MenuBarIcon(.restarting, count: 2).symbol, Token.Icon.working)
    }

    // invariant: the image the menu bar shows and the words VoiceOver reads
    // of it come from the one badge of the page, for every shell story.
    func testTheOneBadgeDrivesTheImageAndTheLabel() throws {
        for story in try Stories.load(root: Tree.stories) where story.ui.statusItem == true {
            let badge = try XCTUnwrap(story.state.page?.badge, story.id)
            let icon = MenuBarIcon.of(connection: story.state.connection, page: story.state.page)
            XCTAssertEqual(icon.label, badge.label, story.id)
            XCTAssertEqual(icon.count, badge.needs, story.id)
            XCTAssertEqual(icon.mark == .down, badge.down == true, story.id)
            XCTAssertEqual(icon.mark == .restarting, badge.restarting == true && badge.down != true, story.id)
            let image = icon.nsImage()
            XCTAssertEqual(image.accessibilityDescription, badge.label, story.id)
            XCTAssertTrue(image.isTemplate, story.id)
        }
    }

    // invariant: MB-04: the CPU line is the owner's option, This Mac's CPU
    // from the page's strip, beside the symbol and its badge.
    func testIconCPUSpark() throws {
        let spark = try XCTUnwrap(Tree.page("menubar.home.two-macs").strip?.spark)
        XCTAssertFalse(spark.isEmpty)
        let icon = try icon("menubar.home.two-macs", sparkline: true)
        XCTAssertEqual(icon, MenuBarIcon(.plain, spark: spark, said: "all machines are fine"))
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

    // invariant: the image the app hands the status item, as AppKit draws
    // it into the bar at its point size, is a template in every state with
    // no colour baked in, and its Orbit fills the optical box of the bar's
    // own round symbol: on the owner's real bar at 2x Wi-Fi drew 33 by 24
    // pixels and the Display circle 30 by 30, the Orbit drawn at 13 pt only
    // 26, and red while a server was down.
    func testTheItemDrawsAsLargeAsTheBarsSymbolsAndInNoColour() throws {
        let wifi = try XCTUnwrap(Ink(Self.native("wifi")))
        XCTAssertEqual(wifi.box.width, 33)
        XCTAssertEqual(wifi.box.height, 24)
        let circle = try XCTUnwrap(Ink(Self.native("circle")))
        XCTAssertEqual(circle.box.height, 30)
        let icons = [MenuBarIcon(.plain), MenuBarIcon(.plain, count: 5), MenuBarIcon(.plain, count: 120), MenuBarIcon(.down, count: 2),
                     MenuBarIcon(.restarting, count: 2), MenuBarIcon(.stopped), MenuBarIcon(.connecting), MenuBarIcon(.plain, busy: true)]
        for var icon in icons {
            icon.spark = [10, 60, 30]
            let image = icon.nsImage()
            XCTAssertTrue(image.isTemplate, "\(icon.mark)")
            let bar = try XCTUnwrap(Self.drawnInTheBar(image).flatMap(Bitmap.init), "\(icon.mark)")
            XCTAssertFalse(Self.tinted(bar), "\(icon.mark) is tinted")
        }
        for icon in [MenuBarIcon(.plain), MenuBarIcon(.plain, count: 3)] {
            let orbit = try XCTUnwrap(Ink(Self.drawnInTheBar(icon.nsImage())), "\(icon.mark)")
            XCTAssertEqual(Double(orbit.box.height), Double(circle.box.height), accuracy: 1, "\(icon.mark)")
        }
    }

    static func tinted(_ bitmap: Bitmap) -> Bool {
        let p = bitmap.pixels
        for i in stride(from: 0, to: p.count, by: 4) where p[i + 3] > 0 {
            if p[i] != p[i + 1] || p[i + 1] != p[i + 2] { return true }
        }
        return false
    }

    // value: an NSImage drawn as the status bar draws its button's image:
    // by AppKit, at its point size, into a 2x bar picking its 2x bitmap.
    static func drawnInTheBar(_ image: NSImage) -> CGImage? {
        let width = Int(image.size.width * 2), height = Int(image.size.height * 2)
        guard let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: width, pixelsHigh: height, bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true,
                                         isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0) else { return nil }
        rep.size = image.size
        NSGraphicsContext.saveGraphicsState()
        NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: rep)
        image.draw(in: NSRect(origin: .zero, size: image.size))
        NSGraphicsContext.restoreGraphicsState()
        return rep.cgImage
    }

    // invariant: the mark is the Orbit filled, a solid Mac inside a ring,
    // never a bubble again: down its middle column, at 1x and at 2x, the
    // ink crosses the ring, the Mac and the ring, three strokes with its
    // centre inked.
    func testTheMarkIsASolidMacInsideARing() throws {
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
            XCTAssertEqual(strokes({ ink.inked(cx, box.minY + $0) }, box.height), 3, "middle column at \(scale)x")
            XCTAssertTrue(ink.inked(cx, cy), "the solid Mac at \(scale)x")
        }
    }

    // invariant: the Orbit has the weight of the bar's filled symbols:
    // drawn as the status bar draws them, 15 pt medium, it is as tall as a
    // circle within one pixel at 2x, its ring as thick as Wi-Fi's arcs
    // within 0.75 of a pixel and heavier than an outlined circle, and its
    // ink as dense as Wi-Fi's, the battery's and Control Center's span.
    func testTheOrbitHasTheWeightOfTheBarsFilledSymbols() throws {
        let orbit = try XCTUnwrap(Ink(MenuBarIcon(.plain).image(scheme: .light)))
        let circle = try XCTUnwrap(Ink(Self.native("circle")))
        XCTAssertEqual(Double(orbit.box.height), Double(circle.box.height), accuracy: 1)
        XCTAssertGreaterThan(orbit.stroke, circle.stroke + 0.75)
        let wifi = try XCTUnwrap(Ink(Self.native("wifi")))
        XCTAssertEqual(orbit.stroke, wifi.stroke, accuracy: 0.75)
        let filled = try ["wifi", "battery.100", "switch.2"].map { try XCTUnwrap(Ink(Self.native($0)), $0).density }
        XCTAssertGreaterThanOrEqual(orbit.density, try XCTUnwrap(filled.min()))
        XCTAssertLessThanOrEqual(orbit.density, try XCTUnwrap(filled.max()))
        XCTAssertGreaterThan(orbit.density, circle.density * 1.5)
    }

    static func native(_ name: String) -> CGImage? {
        let renderer = ImageRenderer(content: Image(systemName: name)
            .font(.system(size: Token.Icon.glyph, weight: Token.Icon.weight))
            .imageScale(.medium)
            .foregroundStyle(Token.Icon.ink(.light)))
        renderer.scale = 2
        return renderer.cgImage
    }

    // invariant: beside the bar's own filled symbols, on a light bar and on
    // a dark one, the Orbit in each of its states reads as one of them.
    func testTheOrbitBesideTheBarsSymbols() throws {
        for scheme in [ColorScheme.dark, .light] {
            let ink = Token.Icon.ink(scheme)
            let icons = [MenuBarIcon(.plain), MenuBarIcon(.plain, count: 5), MenuBarIcon(.down, count: 2), MenuBarIcon(.stopped), MenuBarIcon(.connecting),
                         MenuBarIcon(.plain, busy: true)]
            let row = HStack(spacing: 10) {
                ForEach(["wifi", "battery.100", "switch.2"], id: \.self) { name in
                    Image(systemName: name).font(.system(size: Token.Icon.glyph, weight: Token.Icon.weight)).foregroundStyle(ink)
                }
                ForEach(icons.indices, id: \.self) { i in StatusItem(icons[i], scheme: scheme) }
            }
            .padding(.horizontal, 10)
            .frame(height: 24)
            .background(scheme == .dark ? Color(white: 0.12) : Color(white: 0.94))
            XCTAssertNil(try Golden.check("icon.beside-native", AnyView(row), scheme: scheme, desk: false))
        }
    }

    // invariant: SPEC A3: the mark turns while a server restarts, as while
    // a call runs, and stops turning once nothing does.
    @MainActor
    func testTheMarkTurnsWhileAServerRestartsAndStopsAfter() throws {
        let story = try XCTUnwrap(try Stories.load(root: Tree.stories).first { $0.id == "menubar.restarting" })
        let restarting = MenuBarIcon.of(connection: story.state.connection, page: story.state.page)
        var turns: [Double] = []
        let turner = Turner { turns.append($0) }
        turner.follow(restarting)
        XCTAssertTrue(turner.running)
        RunLoop.main.run(until: Date().addingTimeInterval(3 / Token.Busy.fps))
        XCTAssertGreaterThan(turner.turn, 0)
        turner.follow(MenuBarIcon(.plain, count: restarting.count))
        XCTAssertFalse(turner.running)
        XCTAssertEqual(turner.turn, 0)
        XCTAssertEqual(turns.last, 0)
        turner.follow(MenuBarIcon(.plain, busy: true))
        XCTAssertTrue(turner.running)
        turner.follow(MenuBarIcon(.down))
        XCTAssertFalse(turner.running)
    }

    // invariant: a turning item draws the Orbit working, its machines a
    // third of a turn along the ring, and the Orbit at rest once it stops.
    func testABusyItemTurnsItsMarkOneFrameAtATime() throws {
        let busy = MenuBarIcon(.plain, busy: true)
        XCTAssertEqual(busy.symbol, Token.Icon.working)
        XCTAssertEqual(MenuBarIcon(.restarting).symbol, Token.Icon.working)
        XCTAssertEqual(MenuBarIcon(.stopped, busy: true).symbol, Token.Icon.slashed)
        func bitmap(_ turn: Double) throws -> Bitmap { try XCTUnwrap(busy.image(scheme: .dark, turn: turn).flatMap(Bitmap.init)) }
        XCTAssertEqual(try bitmap(0.25).pixels, try bitmap(0.25).pixels)
        XCTAssertNotEqual(try bitmap(0).pixels, try bitmap(0.25).pixels)
        XCTAssertNotEqual(try XCTUnwrap(MenuBarIcon(.plain).image(scheme: .dark)).width, try bitmap(0).width)
        XCTAssertEqual(busy.nsImage(turn: 0.5).representations.count, 3)
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

    var density: Double {
        var sum = 0.0
        for y in box.minY..<(box.minY + box.height) {
            for x in box.minX..<(box.minX + box.width) { sum += alpha(x, y) }
        }
        return sum / Double(box.width * box.height)
    }

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
