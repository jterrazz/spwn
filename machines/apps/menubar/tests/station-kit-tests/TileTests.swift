@testable import StationKit
import SwiftUI
import XCTest

// invariant: L50d amendment: a row's icon is a 24 pt rounded square, not a
// circle, centred in the vision's 26 pt icon slot, the same tile whatever
// the state (only its glyph takes the accent), and each concept has its
// one symbol, set in Token.
@MainActor
final class TileTests: XCTestCase {
    func bitmap(_ view: some View, scheme: ColorScheme) throws -> Bitmap {
        let renderer = ImageRenderer(content: view.stationTheme(scheme).environment(\.stationStill, true))
        renderer.scale = 1
        return try XCTUnwrap(renderer.cgImage.flatMap(Bitmap.init))
    }

    func pixel(_ b: Bitmap, x: Int, y: Int) -> [UInt8] {
        let i = (y * b.width + x) * 4
        return Array(b.pixels[i..<i + 4])
    }

    func testATileIsARoundedSquare() throws {
        for scheme in [ColorScheme.dark, .light] {
            let b = try bitmap(IconTile(Token.Symbol.mac), scheme: scheme)
            let inset = Int(Token.Tile.slot - Token.Tile.icon) / 2
            XCTAssertEqual(b.width, Int(Token.Tile.slot))
            XCTAssertEqual(b.height, Int(Token.Tile.icon))
            XCTAssertNotEqual(pixel(b, x: inset + 3, y: 3)[3], 0, "\(scheme): the tile's corner is cut as a circle's")
            XCTAssertEqual(pixel(b, x: inset, y: 0)[3], 0, "\(scheme): the tile has no rounded corner")
        }
    }

    func testStateNeverColoursTheTile() throws {
        for scheme in [ColorScheme.dark, .light] {
            let idle = try bitmap(IconTile(Token.Symbol.mac), scheme: scheme)
            let active = try bitmap(IconTile(Token.Symbol.mac, active: true), scheme: scheme)
            XCTAssertEqual(pixel(idle, x: 3, y: 12), pixel(active, x: 3, y: 12), "\(scheme): an active tile changes its fill")
        }
    }

    func testEachConceptHasItsSymbol() {
        XCTAssertEqual(Token.Symbol.machine("Mac"), "desktopcomputer")
        XCTAssertEqual(Token.Symbol.machine("Workspace"), "shippingbox")
        XCTAssertEqual(Token.Symbol.machine("Server"), "server.rack")
        XCTAssertEqual(Token.Symbol.line("ports"), "network")
        XCTAssertEqual(Token.Symbol.accounts, "person.2")
        XCTAssertEqual(Token.Symbol.awake, "sun.max")
        XCTAssertEqual(Token.Symbol.activity, "chart.bar")
    }
}
