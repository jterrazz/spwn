import StationKit
import SwiftUI
import XCTest

@MainActor
final class GlassTests: XCTestCase {
    static func bitmap(_ view: AnyView, over backdrop: Color, scheme: ColorScheme) -> Bitmap? {
        let themed = view.stationTheme(scheme).environment(\.stationNow, Snapshot.now).environment(\.stationStill, true)
        let renderer = ImageRenderer(content: themed.fixedSize().background(backdrop))
        renderer.scale = 1
        return renderer.cgImage.flatMap(Bitmap.init)
    }

    // value: the share of a drawing's pixels that stay the same whatever
    // lies behind it: what the view paints opaque.
    static func opaque(_ view: AnyView, scheme: ColorScheme) -> Double? {
        guard let black = bitmap(view, over: .black, scheme: scheme), let white = bitmap(view, over: .white, scheme: scheme),
              black.width == white.width, black.height == white.height else { return nil }
        var same = 0
        for i in stride(from: 0, to: black.pixels.count, by: 4)
        where (0..<3).allSatisfy({ abs(Int(black.pixels[i + $0]) - Int(white.pixels[i + $0])) <= 24 }) {
            same += 1
        }
        return Double(same) / Double(black.width * black.height)
    }

    func testAPaintedPanelIsCaught() throws {
        let painted = AnyView(Rectangle().fill(Color.gray).frame(width: 100, height: 100))
        XCTAssertGreaterThan(try XCTUnwrap(Self.opaque(painted, scheme: .dark)), 0.9)
    }
}
