@testable import StationKit
import SwiftUI
import XCTest

// invariant: L50d, Apple's rules for Liquid Glass: the panel is the only
// glass, rows paint no ground of their own, and the one subtle grouping a
// page may hold is a quinary fill that turns into a stroke when the owner
// asks for contrast.
@MainActor
final class PlainTests: XCTestCase {
    func bitmap(_ view: some View, contrast: ColorSchemeContrast = .standard) throws -> Bitmap {
        let themed = view.stationTheme(.dark).environment(\.stationNow, Snapshot.now).environment(\.stationStill, true)
            .stationAccessibility(contrast: contrast)
        let renderer = ImageRenderer(content: themed)
        renderer.scale = 1
        return try XCTUnwrap(renderer.cgImage.flatMap(Bitmap.init))
    }

    func alpha(_ b: Bitmap, x: Int, y: Int) -> Int { Int(b.pixels[(y * b.width + x) * 4 + 3]) }

    func lit(_ b: Bitmap, row y: Int) -> Int {
        (0..<b.width).filter { alpha(b, x: $0, y: y) != 0 }.count
    }

    func testARowPaintsNoGround() throws {
        let home = try Tree.page("menubar.home.two-macs")
        let mac = try XCTUnwrap(home.section("machines")?.items?.first?.group?.head)
        let rows: [(String, AnyView)] = [
            ("machine row", AnyView(MachineRow(row: mac, symbol: Token.Symbol.mac, kinds: false, open: { _ in }))),
            ("menu row", AnyView(MenuRow("Apps", subtitle: "38 installed") { EmptyView() } trailing: { Chevron() })),
            ("menu line", AnyView(MenuText(symbol: Token.Symbol.plus, title: "New VM", key: nil, danger: false))),
        ]
        for (name, row) in rows {
            let b = try bitmap(row.frame(width: 300))
            XCTAssertGreaterThan(b.height, 10, name)
            XCTAssertEqual(lit(b, row: 0), 0, "\(name) paints a ground under itself")
            XCTAssertEqual(lit(b, row: b.height - 1), 0, "\(name) paints a ground under itself")
        }
    }

    func testAGroupingIsAFaintFillOrAStroke() throws {
        let group = Color.clear.frame(width: 80, height: 40).stationTile()
        let filled = try bitmap(group)
        let middle = alpha(filled, x: filled.width / 2, y: filled.height / 2)
        XCTAssertGreaterThan(middle, 0)
        XCTAssertLessThan(middle, 40, "a tile is a faint fill, never a painted card")
        let stroked = try bitmap(group, contrast: .increased)
        XCTAssertEqual(alpha(stroked, x: stroked.width / 2, y: stroked.height / 2), 0)
        XCTAssertGreaterThan(lit(stroked, row: stroked.height / 2), 0)
    }

    func testTheMacsIncreaseContrastReachesTheTileThroughThePanelRoot() throws {
        let group = Color.clear.frame(width: 80, height: 40).stationTile()
        let renderer = ImageRenderer(content: group.stationSystemAccessibility().stationTheme(.dark).environment(\._colorSchemeContrast, .increased))
        renderer.scale = 1
        let stroked = try XCTUnwrap(renderer.cgImage.flatMap(Bitmap.init))
        XCTAssertEqual(alpha(stroked, x: stroked.width / 2, y: stroked.height / 2), 0)
        XCTAssertGreaterThan(lit(stroked, row: stroked.height / 2), 0)
    }
}
