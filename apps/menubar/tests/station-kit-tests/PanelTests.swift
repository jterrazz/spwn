import AppKit
@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class PanelTests: XCTestCase {
    struct Frame: View {
        let rows: Int

        var body: some View {
            Panel {
                Text("All machines").font(Token.Role.homeTitle.font)
            } content: {
                VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
                    ForEach(0..<rows, id: \.self) { i in
                        Text("linux-\(i + 1)")
                            .font(Token.Role.rowName.font)
                            .frame(maxWidth: .infinity, minHeight: Token.Metric.Row.height, alignment: .leading)
                    }
                }
            } footer: {
                Text("Find a machine, a port, an action…").font(Token.Role.findResult.font)
            }
        }
    }

    static func height(_ view: some View, room: CGFloat? = nil, still: Bool = true) -> CGFloat? {
        let renderer = ImageRenderer(content: view.fixedSize().stationTheme(.dark).environment(\.stationStill, still).environment(\.stationPanelRoom, room))
        renderer.scale = 1
        return renderer.cgImage.map { CGFloat($0.height) }
    }

    static let variants: [(name: String, contrast: ColorSchemeContrast, reduceTransparency: Bool)] = [
        ("kit/panel", .standard, false), ("kit/panel.ic", .increased, false), ("kit/panel.rt", .standard, true),
    ]

    func testThePanelFrameIsDrawnInEverySchemeAndForEachAccessibilitySetting() throws {
        let failures = try Self.variants.flatMap { variant in
            try [ColorScheme.dark, .light].compactMap { scheme in
                try Golden.check(variant.name, AnyView(Frame(rows: 12).environment(\.stationPanelRoom, 300)
                        .stationAccessibility(contrast: variant.contrast, reduceTransparency: variant.reduceTransparency)),
                    scheme: scheme)
            }
        }
        XCTAssertEqual(failures, [])
    }

    func testThePanelIsGlassUnlessContrastOrTransparencyAskForASolidOne() throws {
        for scheme in [ColorScheme.dark, .light] {
            let glass = try XCTUnwrap(GlassTests.opaque(AnyView(Frame(rows: 4).stationAccessibility()), scheme: scheme))
            XCTAssertLessThan(glass, 0.4, "\(scheme)")
            for variant in Self.variants.dropFirst() {
                let solid = AnyView(Frame(rows: 4).environment(\.stationStill, true)
                    .stationAccessibility(contrast: variant.contrast, reduceTransparency: variant.reduceTransparency))
                XCTAssertGreaterThan(try XCTUnwrap(GlassTests.opaque(solid, scheme: scheme)), 0.8, "\(variant.name) \(scheme)")
            }
        }
    }

    // invariant: live the panel paints nothing, so the window's tinted
    // glass is its one background; drawn still, it draws that tinted glass,
    // the plate over the stand-in's shadows and rim.
    func testLiveThePanelPaintsNothingAndDrawnStillItsTintedGlass() throws {
        func alpha(still: Bool, below: Bool) throws -> UInt8 {
            let pane = Panel { Color.clear.frame(width: Token.Metric.Panel.width, height: 120) }.padding(60)
            let renderer = ImageRenderer(content: pane.stationTheme(.dark).environment(\.stationStill, still))
            renderer.scale = 1
            let bitmap = try XCTUnwrap(renderer.cgImage.flatMap(Bitmap.init))
            let y = below ? bitmap.height - 40 : bitmap.height / 2
            return bitmap.pixels[(y * bitmap.width + bitmap.width / 2) * 4 + 3]
        }
        XCTAssertEqual(try alpha(still: false, below: false), 0)
        XCTAssertEqual(try alpha(still: false, below: true), 0)
        XCTAssertGreaterThanOrEqual(Double(try alpha(still: true, below: false)) / 255, 0.80)
        XCTAssertGreaterThan(try alpha(still: true, below: true), 0)
    }

    func testLiveThePanelPaintsNoRimOnItsEdge() throws {
        for scheme in [ColorScheme.dark, .light] {
            let pane = Panel { Color.clear.frame(width: Token.Metric.Panel.width, height: 120) }
            let renderer = ImageRenderer(content: pane.stationTheme(scheme).environment(\.stationStill, false))
            renderer.scale = 2
            let bitmap = try XCTUnwrap(renderer.cgImage.flatMap(Bitmap.init))
            XCTAssertEqual(stride(from: 3, to: bitmap.pixels.count, by: 4).filter { bitmap.pixels[$0] != 0 }.count, 0, "\(scheme)")
        }
    }

    func testThePanelIsAsTallAsItsContentUnderTheCap() throws {
        let short = try XCTUnwrap(Self.height(Frame(rows: 3)))
        let longer = try XCTUnwrap(Self.height(Frame(rows: 6)))
        XCTAssertLessThan(short, Token.Metric.Panel.cap)
        XCTAssertEqual(longer - short, (Token.Metric.Row.height + Token.Metric.Panel.blocks) * 3, accuracy: 1)
    }

    func testPastTheCapTheBodyIsCutAndTheHeaderAndFooterStay() throws {
        XCTAssertEqual(try XCTUnwrap(Self.height(Frame(rows: 40))), Token.Metric.Panel.cap)
        XCTAssertEqual(try XCTUnwrap(Self.height(Frame(rows: 40), room: 300)), 300)
        XCTAssertGreaterThan(try XCTUnwrap(Self.height(Frame(rows: 40).environment(\.stationPanelWidth, Token.Metric.Panel.width))), 0)
    }

    func testAnUnboundedPanelIsItsWholeContent() throws {
        let unbounded = Panel(bounded: false) {
            VStack(spacing: 0) { ForEach(0..<40, id: \.self) { _ in Color.clear.frame(height: Token.Metric.Row.height) } }
        }
        XCTAssertEqual(try XCTUnwrap(Self.height(unbounded)), Token.Metric.Row.height * 40 + Token.Metric.Panel.body.top + Token.Metric.Panel.body.bottom)
    }

    func testTheCapIsSevenHundredOrTheScreenLessFortyEight() {
        XCTAssertEqual(PanelHeight.cap(visible: nil), 700)
        XCTAssertEqual(PanelHeight.cap(visible: 1000), 700)
        XCTAssertEqual(PanelHeight.cap(visible: 748), 700)
        XCTAssertEqual(PanelHeight.cap(visible: 600), 552)
    }

    func testTheBodyScrollsOnlyPastTheRoomTheHeaderAndFooterLeave() {
        let room = PanelHeight.room(cap: 700, header: 44, footer: 48)
        XCTAssertEqual(room, 608)
        XCTAssertEqual(PanelHeight.body(500, room: room), 500)
        XCTAssertFalse(PanelHeight.scrolls(608, room: room))
        XCTAssertEqual(PanelHeight.body(900, room: room), 608)
        XCTAssertTrue(PanelHeight.scrolls(900, room: room))
        XCTAssertEqual(PanelHeight.room(cap: 100, header: 80, footer: 80), 0)
    }

    func testThePageStackIsAtMostThreeDeep() {
        var stack = PageStack("home")
        XCTAssertEqual(stack.top, "home")
        stack.push("mac-mini")
        stack.push("ports")
        XCTAssertEqual(stack.pages, ["home", "mac-mini", "ports"])
        stack.push("linux-1")
        XCTAssertEqual(stack.pages, ["home", "mac-mini", "linux-1"])
        stack.pop()
        XCTAssertEqual(stack.top, "mac-mini")
        stack.pop()
        stack.pop()
        XCTAssertEqual(stack.pages, ["home"])
        stack.push("needs")
        stack.push("linux-1")
        stack.home()
        XCTAssertEqual(stack.depth, 1)
        XCTAssertEqual(stack.root, "home")
    }

    func testEachLayerDrawsOverThePageWithItsScrim() throws {
        func share(_ layers: PanelLayers) throws -> Double {
            try XCTUnwrap(GlassTests.opaque(AnyView(Panel(layers: layers) { Color.clear.frame(height: 200) }.stationAccessibility()), scheme: .dark))
        }
        let bare = try share(PanelLayers())
        let solid = AnyView(Token.Pane.opaque.frame(width: 100, height: 60))
        for layers in [PanelLayers(find: solid), PanelLayers(sheet: solid), PanelLayers(alert: solid)] {
            XCTAssertGreaterThan(try share(layers), bare)
        }
        let pinned = try XCTUnwrap(GlassTests.opaque(AnyView(Panel {
            Color.clear.frame(height: 200).popoverPin(true) { Token.Pane.opaque.frame(width: 100, height: 60) }
        }.stationAccessibility()), scheme: .dark))
        XCTAssertGreaterThan(pinned, bare, "a popover pinned in the page draws over it")
    }
}

final class Clicks: NSObject {
    var n = 0
    @objc func clicked(_: Any?) { n += 1 }
}

@MainActor
final class PanelWindowTests: XCTestCase {
    // invariant: the panel closes through the status item's button, so the
    // next click on the icon opens it; its window is never closed behind
    // the extra's back.
    func testThePanelClosesThroughItsStatusButton() {
        let clicks = Clicks()
        let button = NSStatusBarButton(frame: .zero)
        button.target = clicks
        button.action = #selector(Clicks.clicked(_:))
        let holder = NSView()
        let inner = NSView()
        holder.addSubview(inner)
        inner.addSubview(button)
        XCTAssertTrue(PanelWindow.statusButton(in: holder) === button)
        let panel = NSWindow(contentRect: .zero, styleMask: [], backing: .buffered, defer: true)
        panel.isReleasedWhenClosed = false
        PanelWindow.close(panel: panel, button: button)
        XCTAssertEqual(clicks.n, 1)
    }

    // invariant: the panel is made key again after it shows, not only at
    // once: macOS shows its window after the content appears.
    func testPanelIsMadeKeyAfterItShows() {
        XCTAssertGreaterThan(PanelWindow.tries.count, 1)
        XCTAssertGreaterThanOrEqual(PanelWindow.tries.max() ?? 0, 0.5)
    }
}
