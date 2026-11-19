@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class IconTests: XCTestCase {
    func icon(_ fixture: String, _ connection: Connection = .live) throws -> MenuBarIcon {
        MenuBarIcon.of(connection: connection, page: try Tree.page(fixture))
    }

    func story(_ id: String) throws -> Story {
        try XCTUnwrap(try Stories.load(root: Tree.stories).first { $0.id == id }, id)
    }

    func icon(story id: String) throws -> MenuBarIcon {
        let story = try story(id)
        return MenuBarIcon.of(connection: story.state.connection, page: story.state.page)
    }

    static func bar(_ scheme: ColorScheme) -> Color { scheme == .dark ? Color(white: 0.12) : Color(white: 0.94) }

    // value: a state drawn as the bar draws it: on the bar's own grey, in
    // light and dark, at 1x and at 2x, the wave at the turn it is caught at.
    func drawn(_ name: String, _ icon: MenuBarIcon, turn: Double = 0) throws {
        for scheme in [ColorScheme.dark, .light] {
            for scale in [1, 2] as [CGFloat] {
                let view = StatusItem(icon, scheme: scheme, turn: turn).padding(6).background(Self.bar(scheme))
                XCTAssertNil(try Golden.check("icon.\(name).\(Int(scale))x", AnyView(view), scheme: scheme, desk: false, scale: scale))
            }
        }
    }

    func ink(_ icon: MenuBarIcon, scale: CGFloat = 2, turn: Double = 0) throws -> Ink {
        try XCTUnwrap(Ink(icon.image(scheme: .light, scale: scale, turn: turn)))
    }

    // value: the alpha at a point of the 48 by 44 box, in the pixels of the
    // icon drawn at a scale, the box being `Token.Icon.height` tall.
    func alpha(_ icon: MenuBarIcon, at point: CGPoint, scale: CGFloat, turn: Double = 0) throws -> Double {
        let bitmap = try XCTUnwrap(icon.image(scheme: .light, scale: scale, turn: turn).flatMap(Bitmap.init))
        let k = Token.Icon.height / Token.Icon.box.height * scale
        return Double(bitmap.pixels[(Int(point.y * k) * bitmap.width + Int(point.x * k)) * 4 + 3]) / 255
    }

    // value: the alpha at a dot's centre.
    func alpha(_ icon: MenuBarIcon, dot: Int, scale: CGFloat, turn: Double = 0) throws -> Double {
        let d = Token.Icon.dots[dot]
        return try alpha(icon, at: CGPoint(x: d.x, y: d.y), scale: scale, turn: turn)
    }

    // value: the middle dot's own points on the slash: where the line passes
    // closest to its centre, and a point of the cut-out between the line's
    // stroke and the cut's edge, on the centre's side, still inside the dot.
    func slashPoints() -> (stroke: CGPoint, gap: CGPoint) {
        let from = Token.Icon.slashFrom, to = Token.Icon.slashTo
        let d = Token.Icon.dots[1]
        let (dx, dy) = (to.x - from.x, to.y - from.y)
        let t = ((d.x - from.x) * dx + (d.y - from.y) * dy) / (dx * dx + dy * dy)
        let foot = CGPoint(x: from.x + t * dx, y: from.y + t * dy)
        let away = hypot(d.x - foot.x, d.y - foot.y)
        let reach = (Token.Icon.stroke + Token.Icon.cut) / 4
        return (foot, CGPoint(x: foot.x + (d.x - foot.x) * reach / away, y: foot.y + (d.y - foot.y) * reach / away))
    }

    // invariant: MB-01: the plain state: nothing is down and nothing is
    // restarting, the three dots on their arc in the one ink.
    func testIconPlain() throws {
        let icon = try icon("menubar.home.one-mac")
        XCTAssertEqual(icon, MenuBarIcon(.plain, said: "all machines are fine"))
        XCTAssertEqual(icon.label, "all machines are fine")
        XCTAssertEqual(icon.face, .plain)
        try drawn("plain", icon)
    }

    // invariant: MB-02: five findings of upkeep and no server down: the
    // plain mark, since upkeep never shows in the bar.
    func testUpkeepNeverShows() throws {
        XCTAssertEqual(try icon("menubar.home.two-macs"), MenuBarIcon(.plain, said: "all machines are fine"))
    }

    // invariant: MB-03: infra in a loop on the mini: the mark slashed,
    // never red, and the label names the server.
    func testIconServerDownIsSlashed() throws {
        let icon = try icon("menubar.home.infra-down")
        XCTAssertEqual(icon, MenuBarIcon(.down, said: "Spwn Menu, infra down"))
        XCTAssertEqual(icon.face, .slashed)
        XCTAssertTrue(icon.nsImage().isTemplate)
        try drawn("slashed", icon)
    }

    // invariant: a port exposed slashes the mark as a server down does.
    func testAPortExposedIsSlashedToo() throws {
        let exposed = try icon(story: "home.exposed")
        XCTAssertEqual(exposed.mark, .down)
        XCTAssertEqual(exposed.face, .slashed)
        XCTAssertEqual(exposed.label, "Spwn Menu, port 5000 exposed")
    }

    // invariant: the slash is one line from the top left to the bottom
    // right, crossing the middle dot: its stroke is inked on the line, the
    // dots are cut out of around it, and the other two dots and the plain
    // mark keep theirs, at 1x and at 2x.
    func testTheSlashIsCutThroughTheMiddleDot() throws {
        let points = slashPoints()
        let dot = Token.Icon.dots[1]
        XCTAssertLessThan(hypot(points.stroke.x - dot.x, points.stroke.y - dot.y), dot.r)
        for scale in [1, 2] as [CGFloat] {
            let plain = MenuBarIcon(.plain), slashed = MenuBarIcon(.down)
            let full = scale == 1 ? 0.5 : 0.95, gap = scale == 1 ? 0.5 : 0.05
            XCTAssertGreaterThan(try alpha(plain, at: points.gap, scale: scale), 0.95, "plain, the middle dot at \(scale)x")
            XCTAssertLessThan(try alpha(slashed, at: points.gap, scale: scale), gap, "slashed, the cut-out around the line at \(scale)x")
            XCTAssertGreaterThan(try alpha(slashed, at: points.stroke, scale: scale), full, "slashed, the stroke at \(scale)x")
            for dot in [0, 2] {
                XCTAssertGreaterThan(try alpha(slashed, dot: dot, scale: scale), 0.95, "slashed, dot \(dot) at \(scale)x")
            }
        }
    }

    // invariant: a restart in progress is the wave of its dots, first to
    // last, and the dots come back plain once it ends.
    func testIconRestartingWaves() throws {
        let icon = try icon(story: "menubar.restarting")
        XCTAssertEqual(icon, MenuBarIcon(.restarting, said: "infra restarting, 2 of 4 checks"))
        XCTAssertEqual(icon.face, .waving)
        XCTAssertTrue(icon.turning)
        let low = Token.Icon.low, peak = Token.Icon.peak, lag = Token.Icon.stagger / Token.Busy.period
        for dot in 0..<3 {
            let start = Double(dot) * lag
            XCTAssertEqual(SpwnMark.light(dot: dot, turn: start), low, accuracy: 1e-9, "dot \(dot) at the start of its run")
            XCTAssertEqual(SpwnMark.light(dot: dot, turn: start + peak), 1, accuracy: 1e-9, "dot \(dot) at its peak")
            XCTAssertEqual(SpwnMark.light(dot: dot, turn: start + peak / 2), (1 + low) / 2, accuracy: 1e-9, "dot \(dot) half way up")
            XCTAssertEqual(SpwnMark.light(dot: dot, turn: start + 1), low, accuracy: 1e-9, "dot \(dot) a loop later")
            XCTAssertLessThan(SpwnMark.light(dot: dot, turn: start + peak / 4), SpwnMark.light(dot: dot, turn: start + peak / 2), "dot \(dot) rising")
            XCTAssertGreaterThan(SpwnMark.light(dot: dot, turn: start + peak + 0.1), SpwnMark.light(dot: dot, turn: start + peak + 0.4), "dot \(dot) falling")
        }
        for frame in 0..<Token.Busy.frames {
            for dot in 0..<3 {
                let light = SpwnMark.light(dot: dot, turn: Token.Busy.turn(frame: frame))
                XCTAssertGreaterThanOrEqual(light, low - 1e-9)
                XCTAssertLessThanOrEqual(light, 1)
            }
        }
        XCTAssertEqual(Double(Token.Busy.frames) / Token.Busy.fps, 1.2, accuracy: 1e-9)
        XCTAssertEqual(Token.Icon.stagger, 0.2)
        XCTAssertGreaterThan(try alpha(icon, dot: 0, scale: 2, turn: peak), try alpha(icon, dot: 2, scale: 2, turn: peak) + 0.3)
        XCTAssertGreaterThan(try alpha(icon, dot: 2, scale: 2, turn: peak + 2 * lag), try alpha(icon, dot: 0, scale: 2, turn: peak + 2 * lag) + 0.2)
        try drawn("waving", icon, turn: 0.4)
    }

    // invariant: a call running waves the plain mark as a restart does;
    // the slashed and the stopped marks keep their slash.
    func testACallRunningWavesThePlainMarkOnly() {
        XCTAssertEqual(MenuBarIcon(.plain, busy: true).face, .waving)
        XCTAssertEqual(MenuBarIcon(.down, busy: true).face, .slashed)
        XCTAssertEqual(MenuBarIcon(.stopped, busy: true).face, .away)
        XCTAssertEqual(MenuBarIcon(.connecting, busy: true).face, .connecting)
    }

    // invariant: EN-10, MB-11: the engine stopped, or station not installed,
    // dims the slashed mark, dots and stroke, to 45 % and says so; the icon never says all good
    // while nothing answers.
    func testIconEngineStopped() throws {
        let stopped = try icon("menubar.home.engine-stopped", .stopped)
        XCTAssertEqual(stopped, MenuBarIcon(.stopped))
        XCTAssertEqual(stopped.label, "Spwn Menu: engine stopped")
        XCTAssertEqual(stopped.face, .away)
        XCTAssertNotEqual(stopped.face, MenuBarIcon(.plain).face)
        XCTAssertEqual(try icon("menubar.home.two-macs", .notInstalled), MenuBarIcon(.stopped))
        XCTAssertEqual(MenuBarIcon.of(connection: .live, page: try Tree.page("menubar.loading")), MenuBarIcon(.plain))
        for scale in [1, 2] as [CGFloat] {
            let away = stopped
            XCTAssertEqual(try alpha(away, dot: 0, scale: scale), Token.Icon.dimmed, accuracy: 0.03, "dot 0 at \(scale)x")
            XCTAssertEqual(try alpha(away, dot: 2, scale: scale), Token.Icon.dimmed, accuracy: 0.03, "dot 2 at \(scale)x")
            XCTAssertLessThan(try alpha(away, at: slashPoints().gap, scale: scale), scale == 1 ? 0.5 : 0.05, "the cut-out at \(scale)x")
        }
        XCTAssertEqual(try alpha(stopped, at: slashPoints().stroke, scale: 2), Token.Icon.dimmed, accuracy: 0.03, "the stroke at 45 %")
        try drawn("stopped", stopped)
    }

    // invariant: a connection being made keeps the first dot solid and the
    // two others at 32 %.
    func testIconConnecting() throws {
        let connecting = try icon("menubar.home.two-macs", .connecting)
        XCTAssertEqual(connecting, MenuBarIcon(.connecting))
        XCTAssertEqual(connecting.label, "Spwn Menu: connecting")
        XCTAssertEqual(connecting.face, .connecting)
        for scale in [1, 2] as [CGFloat] {
            let drawing = connecting
            XCTAssertGreaterThan(try alpha(drawing, dot: 0, scale: scale), 0.97, "dot 0 at \(scale)x")
            XCTAssertEqual(try alpha(drawing, dot: 1, scale: scale), Token.Icon.resting, accuracy: 0.03, "dot 1 at \(scale)x")
            XCTAssertEqual(try alpha(drawing, dot: 2, scale: scale), Token.Icon.resting, accuracy: 0.03, "dot 2 at \(scale)x")
        }
        try drawn("connecting", connecting)
    }

    // invariant: the menu open is the plain mark on the system's highlight:
    // the mark itself does not change, the strip paints the pressed pill.
    func testTheMenuOpenIsThePlainMarkOnTheHighlight() throws {
        let open = try story("menubar.open")
        XCTAssertEqual(open.ui.pressed, true)
        XCTAssertEqual(try icon(story: "menubar.open").face, .plain)
        XCTAssertEqual(try icon(story: "menubar.open").nsImage().representations.count, 3)
    }

    // invariant: the image the menu bar shows and the words VoiceOver reads
    // of it come from the one badge of the page, for every shell story.
    func testTheOneBadgeDrivesTheImageAndTheLabel() throws {
        for story in try Stories.load(root: Tree.stories) where story.ui.statusItem == true && story.state.connection == .live {
            let badge = try XCTUnwrap(story.state.page?.badge, story.id)
            let icon = MenuBarIcon.of(connection: story.state.connection, page: story.state.page)
            XCTAssertEqual(icon.label, badge.label, story.id)
            XCTAssertEqual(icon.mark == .down, badge.down == true, story.id)
            XCTAssertEqual(icon.mark == .restarting, badge.restarting == true && badge.down != true, story.id)
            let image = icon.nsImage()
            XCTAssertEqual(image.accessibilityDescription, badge.label, story.id)
            XCTAssertTrue(image.isTemplate, story.id)
        }
    }

    // invariant: nothing but the three dots is drawn: the page's count of
    // what needs you never reaches the item, so a page carrying more or
    // fewer needs draws the same bitmap.
    func testNoCountIsDrawnBesideTheMark() throws {
        let plain = try ink(MenuBarIcon(.plain))
        XCTAssertEqual(Double(plain.bitmap.width), Double(Token.Icon.height / Token.Icon.box.height * Token.Icon.box.width * 2), accuracy: 1)
        XCTAssertEqual(try ink(try icon("menubar.home.two-macs")).bitmap.pixels, plain.bitmap.pixels)
        XCTAssertEqual(try ink(try icon("menubar.home.one-mac")).bitmap.pixels, plain.bitmap.pixels)
    }

    // invariant: the image the app hands the status item, as AppKit draws
    // it into the bar at its point size, is a template in every state with
    // no colour baked in, and its mark is as tall as the bar's own round
    // symbols: in the menu bar VM, the reference, at 2x Wi-Fi draws 34 by 25
    // pixels and the Display circle 30 by 30 (33 by 24 on a host Mac).
    func testTheItemDrawsAsLargeAsTheBarsSymbolsAndInNoColour() throws {
        let wifi = try XCTUnwrap(Ink(Self.native("wifi")))
        XCTAssertEqual(wifi.box.width, 34)
        XCTAssertEqual(wifi.box.height, 25)
        let circle = try XCTUnwrap(Ink(Self.native("circle")))
        XCTAssertEqual(circle.box.height, 30)
        let icons = [MenuBarIcon(.plain), MenuBarIcon(.down), MenuBarIcon(.restarting), MenuBarIcon(.stopped), MenuBarIcon(.connecting), MenuBarIcon(.plain, busy: true)]
        for icon in icons {
            let image = icon.nsImage()
            XCTAssertTrue(image.isTemplate, "\(icon.mark)")
            let bar = try XCTUnwrap(Self.drawnInTheBar(image).flatMap(Bitmap.init), "\(icon.mark)")
            XCTAssertFalse(Self.tinted(bar), "\(icon.mark) is tinted")
        }
        let mark = try XCTUnwrap(Ink(Self.drawnInTheBar(MenuBarIcon(.plain).nsImage())))
        XCTAssertEqual(Double(mark.box.height), Double(circle.box.height), accuracy: 1)
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

    // invariant: the mark is three dots on an arc, never joined: each
    // dot's centre is inked and the ground between the first and the
    // second is empty, at 1x and at 2x.
    func testTheMarkIsThreeDotsOnAnArc() throws {
        for scale in [1, 2] as [CGFloat] {
            let drawing = MenuBarIcon(.plain)
            let ground = try ink(drawing, scale: scale)
            let k = Token.Icon.height / Token.Icon.box.height * scale
            for dot in 0..<3 { XCTAssertGreaterThan(try alpha(drawing, dot: dot, scale: scale), 0.95, "dot \(dot) at \(scale)x") }
            let between = Token.Icon.dots[1].x - Token.Icon.dots[1].r - 4
            XCTAssertLessThan(ground.alpha(Int(between * k), Int(Token.Icon.dots[0].y * k)), 0.05, "between the dots at \(scale)x")
        }
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
    // a dark one, the mark in each of its states reads as one of them.
    func testTheMarkBesideTheBarsSymbols() throws {
        for scheme in [ColorScheme.dark, .light] {
            let ink = Token.Icon.ink(scheme)
            let icons = [MenuBarIcon(.plain), MenuBarIcon(.down), MenuBarIcon(.stopped), MenuBarIcon(.connecting), MenuBarIcon(.restarting)]
            let row = HStack(spacing: 10) {
                ForEach(["wifi", "battery.100", "switch.2"], id: \.self) { name in
                    Image(systemName: name).font(.system(size: Token.Icon.glyph, weight: Token.Icon.weight)).foregroundStyle(ink)
                }
                ForEach(icons.indices, id: \.self) { i in StatusItem(icons[i], scheme: scheme, turn: 0.4) }
            }
            .padding(.horizontal, 10)
            .frame(height: 24)
            .background(Self.bar(scheme))
            XCTAssertNil(try Golden.check("icon.beside-native", AnyView(row), scheme: scheme, desk: false))
        }
    }

    // invariant: SPEC A3: the mark waves while a server restarts, as while
    // a call runs, under Reduce Motion as well, and stops once nothing does.
    func testTheMarkWavesWhileAServerRestartsAndStopsAfter() throws {
        let restarting = try icon(story: "menubar.restarting")
        var turns: [Double] = []
        let turner = Turner { turns.append($0) }
        turner.follow(restarting)
        XCTAssertTrue(turner.running)
        RunLoop.main.run(until: Date().addingTimeInterval(3 / Token.Busy.fps))
        XCTAssertGreaterThan(turner.turn, 0)
        turner.follow(MenuBarIcon(.plain))
        XCTAssertFalse(turner.running)
        XCTAssertEqual(turner.turn, 0)
        XCTAssertEqual(turns.last, 0)
        turner.follow(MenuBarIcon(.plain, busy: true))
        XCTAssertTrue(turner.running)
        turner.follow(MenuBarIcon(.down))
        XCTAssertFalse(turner.running)
        turner.follow(MenuBarIcon(.stopped, busy: true))
        XCTAssertFalse(turner.running)
    }

    // invariant: a waving item draws the dots at the angle of its frame, one
    // frame at a time, 12 to the turn, and the plain mark once it stops.
    func testABusyItemWavesItsDotsOneFrameAtATime() throws {
        let busy = MenuBarIcon(.plain, busy: true)
        func bitmap(_ turn: Double) throws -> Bitmap { try XCTUnwrap(busy.image(scheme: .dark, turn: turn).flatMap(Bitmap.init)) }
        XCTAssertEqual(try bitmap(0.25).pixels, try bitmap(0.25).pixels)
        XCTAssertNotEqual(try bitmap(0).pixels, try bitmap(0.25).pixels)
        XCTAssertEqual(try XCTUnwrap(MenuBarIcon(.plain).image(scheme: .dark)).width, try bitmap(0).width)
        XCTAssertEqual(busy.nsImage(turn: 0.5).representations.count, 3)
        XCTAssertEqual(Token.Busy.frames, 12)
        XCTAssertEqual(Token.Busy.turn(frame: 3), 0.25)
        XCTAssertEqual(Token.Busy.turn(frame: 12), 0)
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
