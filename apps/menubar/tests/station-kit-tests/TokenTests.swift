import StationKit
import SwiftUI
import XCTest

final class TokenTests: XCTestCase {
    static func css(_ x1: Double, _ y1: Double, _ x2: Double, _ y2: Double, at x: Double) -> Double {
        func bezier(_ t: Double, _ p1: Double, _ p2: Double) -> Double { 3 * (1 - t) * (1 - t) * t * p1 + 3 * (1 - t) * t * t * p2 + t * t * t }
        var lo = 0.0, hi = 1.0
        for _ in 0..<60 {
            let mid = (lo + hi) / 2
            if bezier(mid, x1, x2) < x { lo = mid } else { hi = mid }
        }
        return bezier((lo + hi) / 2, y1, y2)
    }

    static func cssLinear(_ stops: [(at: Double, value: Double)], at x: Double) -> Double {
        for k in 1..<stops.count where x <= stops[k].at {
            let a = stops[k - 1], b = stops[k]
            return a.value + (b.value - a.value) * (x - a.at) / (b.at - a.at)
        }
        return stops.last?.value ?? 1
    }

    static let tenths = (1...10).map { Double($0) / 11 }

    func testEachCurveIsThePrototypesCubicBezier() {
        let curves: [(String, UnitCurve, [Double])] = [
            ("snappy", Token.Motion.snappyCurve, [0.30, 0.90, 0.30, 1]),
            ("smooth", Token.Motion.smoothCurve, [0.32, 0.72, 0, 1]),
        ]
        for (name, curve, p) in curves {
            for x in Self.tenths {
                XCTAssertEqual(curve.value(at: x), Self.css(p[0], p[1], p[2], p[3], at: x), accuracy: 0.001, "\(name) at \(x)")
            }
        }
    }

    func testTheSuccessSpringFollowsThePrototypesBounce() {
        for x in Self.tenths {
            XCTAssertEqual(Token.Motion.bouncy(at: x), Self.cssLinear(Token.Motion.bouncyStops, at: x), accuracy: 0.015, "bouncy at \(x)")
        }
        XCTAssertGreaterThan(Token.Motion.bouncy(at: 0.3), 1.1)
    }

    func testIncreaseContrastAndReduceTransparencyMakeThePanelSolid() {
        for scheme in [ColorScheme.dark, .light] {
            let plain = Token.resolved(scheme: scheme)
            let ic = Token.resolved(scheme: scheme, contrast: .increased)
            let rt = Token.resolved(scheme: scheme, reduceTransparency: true)
            XCTAssertFalse(plain.solid)
            XCTAssertEqual(ic.panel, ic.color.panelSolid)
            XCTAssertEqual(rt.panel, rt.color.panelSolid)
            XCTAssertNotEqual(plain.color.hair, ic.color.hair)
            XCTAssertEqual(plain.color.t1, ic.color.t1)
        }
    }

    func testReduceMotionDissolvesEveryMovement() {
        let reduced = Token.resolved(scheme: .dark, reduceMotion: true)
        XCTAssertEqual(reduced.motion(Token.Motion.smooth), reduced.motion(Token.Motion.snappy))
        XCTAssertNotEqual(reduced.motion(Token.Motion.smooth), Token.Motion.smooth)
        XCTAssertEqual(Token.resolved(scheme: .dark).motion(Token.Motion.smooth), Token.Motion.smooth)
    }

    func testTheAccentIsPinnedUnlessTheAppHandsTheOwnersOne() {
        XCTAssertEqual(Token.resolved(scheme: .dark).palette.accent, Token.accentDark)
        XCTAssertEqual(Token.resolved(scheme: .light).color.accent, Token.accentLight)
        XCTAssertEqual(Token.resolved(scheme: .dark, accent: Color.purple).palette.accent, Color.purple)
    }

    func testAnUnknownBrandWearsTheQuietChip() {
        let tokens = Token.resolved(scheme: .dark)
        XCTAssertEqual(tokens.chip("vite").fill.count, 2)
        XCTAssertEqual(tokens.chip("gitlab").fill, [tokens.color.q])
        XCTAssertEqual(tokens.chip(nil).mark, tokens.color.t2)
    }

    // invariant: the token sheet draws the colour table itself, so a
    // channel off by more than 2 of 255 is a value changed, not a render.
    @MainActor
    func testTheTokenSheet() throws {
        let failures = try [ColorSchemeContrast.standard, .increased].flatMap { contrast in
            try [ColorScheme.dark, .light].compactMap { scheme in
                try Golden.check(contrast == .increased ? "kit/tokens.ic" : "kit/tokens", AnyView(TokenSheet().stationAccessibility(contrast: contrast)), scheme: scheme, desk: false, tolerance: 2)
            }
        }
        XCTAssertEqual(failures, [])
    }
}
