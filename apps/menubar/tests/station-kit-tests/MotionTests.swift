@testable import StationKit
import SwiftUI
import XCTest

final class MotionTests: XCTestCase {
    static let moments = [0, 0.35, 1]
    static let smooth = { (t: Double) in Token.Motion.smoothCurve.value(at: t) }
    static let snappy = { (t: Double) in Token.Motion.snappyCurve.value(at: t) }

    func near(_ a: Double, _ b: Double, _ what: String, file: StaticString = #filePath, line: UInt = #line) {
        XCTAssertEqual(a, b, accuracy: 0.0001, what, file: file, line: line)
    }

    func testEveryRowOfSpecA8HasItsModifierOrItsNamedExemption() {
        let drawn = MotionSheet.cells.map(\.id)
        let exempt = Set(MotionSheet.exempt.keys)
        XCTAssertEqual(Set(drawn).count, drawn.count)
        XCTAssertTrue(exempt.isDisjoint(with: drawn))
        XCTAssertEqual(exempt.union(drawn), Set((1...31).map { "M\($0)" }))
        XCTAssertEqual(exempt, ["M13", "M14", "M16", "M29"])
    }

    @MainActor
    func testTheMotionSheetDrawsEveryMovementAtAThirdOfItsRun() throws {
        let failures = try [ColorScheme.dark, .light].compactMap { scheme in
            try Golden.check("kit/motion", AnyView(MotionSheet().stationAccessibility()), scheme: scheme, desk: false)
        }
        XCTAssertEqual(failures, [])
    }

    func testTheLiveAppAnimatesAMovementThroughItsProgress() {
        var m = PanelAppear(progress: 0)
        m.animatableData = 0.4
        XCTAssertEqual(m.progress, 0.4)
    }

    func testAFrozenMomentIsTheShareOfTheWayItsCurveHasMade() {
        for t in Self.moments { near(Token.Motion.freeze(t, Token.Motion.smoothCurve), Self.smooth(t), "smooth at \(t)") }
        near(Token.Motion.freeze(-1, Token.Motion.smoothCurve), 0, "before the start")
        near(Token.Motion.freeze(2, Token.Motion.smoothCurve), 1, "past the end")
        near(Token.Motion.freeze(0.35, .bouncy), Token.Motion.bouncy(at: 0.35), "the success spring")
        XCTAssertGreaterThan(Self.smooth(0.35), 0.5, "smooth is front-loaded: past half its way at a third of its time")
    }

    func testThePanelGrowsOutOfItsItemAndClosesToASlighterScale() {
        for t in Self.moments {
            let look = PanelAppear.look(PanelAppear.frozen(t, closing: false), reduce: false)
            near(look.opacity, Self.smooth(t), "opacity at \(t)")
            near(look.scale.width, 0.96 + 0.04 * Self.smooth(t), "scale at \(t)")
        }
        let closed = PanelAppear.look(PanelAppear.frozen(1, closing: true), closing: true, reduce: false)
        near(closed.scale.width, 0.98, "closed")
        near(closed.opacity, 0, "closed")
        near(PanelAppear.frozen(0.35, closing: true), 1 - Self.snappy(0.35), "it closes on snappy")
    }

    func testAPushedPageSlidesInOpaqueOverThePageBelowWhichDimsAThirdAway() {
        for t in Self.moments {
            let p = Self.smooth(t)
            let top = PushTransition.look(PushTransition.frozen(t, role: .top, way: .push), role: .top, way: .push, reduce: false)
            let under = PushTransition.look(PushTransition.frozen(t, role: .under, way: .push), role: .under, way: .push, reduce: false)
            near(top.slide, 1 - p, "the top page's slide at \(t)")
            near(top.opacity, 1, "the top page is opaque at \(t)")
            near(under.slide, -0.3 * p, "the page below at \(t)")
            let dim = p <= 0.3 ? 1 - 0.75 * p / 0.3 : 0.25 * (1 - (p - 0.3) / 0.7)
            near(under.opacity, dim, "the page below's dim at \(t)")
        }
        XCTAssertEqual(PushTransition.look(0.5, role: .top, way: .push, reduce: false).shade, 1, "a sliding page casts its shade")
        XCTAssertEqual(PushTransition.look(1, role: .top, way: .push, reduce: false).shade, 0, "a page at rest casts none")
    }

    func testAPoppedPageLeavesRightFadingWhileThePageBelowReturns() {
        for t in Self.moments {
            let p = Self.smooth(t)
            let top = PushTransition.look(PushTransition.frozen(t, role: .top, way: .pop), role: .top, way: .pop, reduce: false)
            let under = PushTransition.look(PushTransition.frozen(t, role: .under, way: .pop), role: .under, way: .pop, reduce: false)
            near(top.slide, p, "the leaving page at \(t)")
            near(top.opacity, 1 - 0.4 * p, "the leaving page fades to .6 at \(t)")
            near(under.slide, -0.3 * (1 - p), "the returning page at \(t)")
            near(under.opacity, p, "the returning page at \(t)")
        }
    }

    func testUnderReduceMotionTwoPagesNeverPrintOverEachOther() {
        for way in [PushTransition.Way.push, .pop] {
            for step in 0...20 {
                let t = Double(step) / 20
                let top = PushTransition.look(PushTransition.frozen(t, role: .top, way: way, reduce: true), role: .top, way: way, reduce: true)
                let under = PushTransition.look(PushTransition.frozen(t, role: .under, way: way, reduce: true), role: .under, way: way, reduce: true)
                XCTAssertEqual(min(top.opacity, under.opacity), 0, accuracy: 0.0001, "\(way) at \(t)")
                XCTAssertEqual(top.slide, 0)
                XCTAssertEqual(under.slide, 0)
            }
        }
        let gone = PushTransition.look(PushTransition.frozen(0.45, role: .under, way: .push, reduce: true), role: .under, way: .push, reduce: true)
        XCTAssertLessThan(gone.opacity, 0.2)
    }

    func testAJumpFadesTheOldPageOutBeforeTheNewOneShows() {
        for step in 0...20 {
            let t = Double(step) / 20
            let leaving = JumpTransition.look(JumpTransition.frozen(t, role: .leaving), role: .leaving).opacity
            let arriving = JumpTransition.look(JumpTransition.frozen(t, role: .arriving), role: .arriving).opacity
            XCTAssertLessThan(min(leaving, arriving), 0.1, "no double exposure at \(t)")
        }
        XCTAssertEqual(JumpTransition.look(JumpTransition.frozen(0.35, role: .leaving), role: .leaving).opacity, 0, accuracy: 0.05)
        XCTAssertEqual(JumpTransition.look(JumpTransition.frozen(1, role: .arriving), role: .arriving).opacity, 1)
        XCTAssertEqual(JumpTransition.look(JumpTransition.frozen(0, role: .leaving), role: .leaving).opacity, 1)
    }

    func testTheSegmentThumbSlidesBetweenTheTwoMeasuredFramesAndTheBodyCrossSlides() {
        let from = CGRect(x: 0, y: 0, width: 60, height: 20), to = CGRect(x: 60, y: 0, width: 80, height: 20)
        for t in Self.moments {
            let p = Self.snappy(t)
            let frame = SegmentThumb.frame(SegmentThumb.frozen(t), from: from, to: to)
            near(frame.minX, 60 * p, "x at \(t)")
            near(frame.width, 60 + 20 * p, "width at \(t)")
            near(Double(SegmentSlide.look(SegmentSlide.frozen(t), forward: true, reduce: false).x), 10 * (1 - p), "body at \(t)")
        }
        XCTAssertEqual(SegmentSlide.look(0, forward: false, reduce: false).x, -10)
    }

    func testThePanelsHeightFollowsItsContent() {
        for t in Self.moments {
            near(Double(HeightMorph.height(HeightMorph.frozen(t, reduce: false), from: 200, to: 300)), 200 + 100 * Self.smooth(t), "at \(t)")
        }
        near(HeightMorph.frozen(0.35, reduce: true), UnitCurve.easeInOut.value(at: 0.35), "heights still ease under Reduce Motion")
    }

    func testAFoldOpensItsHeightItsRowsFadeInFirstAndSettleAndItsChevronTurns() {
        for t in Self.moments {
            let p = Self.smooth(t)
            let look = Fold.look(Fold.frozen(t, reduce: false), reduce: false)
            near(look.height, p, "height at \(t)")
            near(look.opacity, min(p / (0.12 / 0.34), 1), "rows at \(t)")
            near(Double(look.y), -4 * (1 - p), "settle at \(t)")
            near(FoldChevron.look(p, reduce: false).turn, 180 * p, "chevron at \(t)")
        }
    }

    @MainActor
    func testAFoldHalfOpenIsHalfItsHeightInOnePass() throws {
        let content = Rectangle().frame(width: 40, height: 40)
        let half = try XCTUnwrap(Snapshot.render(AnyView(content.modifier(Fold(progress: 0.5))), scheme: .dark, desk: false))
        XCTAssertEqual(half.bitmap.height, 40)
        let leaving = try XCTUnwrap(Snapshot.render(AnyView(content.modifier(RowLeave(progress: 0.25))), scheme: .dark, desk: false))
        XCTAssertEqual(leaving.bitmap.height, 20)
    }

    func testTheHeroMorphsItsTopFirstItsRowsLaterAndItsGlyphOnTheSpring() {
        for t in Self.moments {
            let top = HeroMorph.look(HeroMorph.frozen(t, part: .top, reduce: false), part: .top, reduce: false)
            let rows = HeroMorph.look(HeroMorph.frozen(t, part: .rows, reduce: false), part: .rows, reduce: false)
            let glyph = HeroMorph.look(HeroMorph.frozen(t, part: .glyph, reduce: false), part: .glyph, reduce: false)
            near(top.opacity, Self.smooth(t), "top at \(t)")
            near(Double(top.y), -3 * (1 - Self.smooth(t)), "top's drop at \(t)")
            XCTAssertLessThanOrEqual(rows.opacity, top.opacity, "rows follow at \(t)")
            near(glyph.scale.width, 0.6 + 0.4 * Token.Motion.bouncy(at: t), "glyph at \(t)")
        }
    }

    func testARowLeavesCollapsingSlidingAndFadingAndARowLandsFromAbove() {
        for t in Self.moments {
            let shown = 1 - Self.smooth(t)
            let leave = RowLeave.look(RowLeave.frozen(t, reduce: false), reduce: false)
            near(leave.height, shown, "leaving height at \(t)")
            near(leave.opacity, shown, "leaving opacity at \(t)")
            near(Double(leave.x), 10 * (1 - shown), "leaving slide at \(t)")
            let land = RowLand.look(RowLand.frozen(t), reduce: false)
            near(land.scale.width, 0.97 + 0.03 * Self.smooth(t), "landing scale at \(t)")
            near(Double(land.y), -3 * (1 - Self.smooth(t)), "landing drop at \(t)")
        }
    }

    func testAFreshRowHoldsItsAccentForAThirdOfItsRunThenFades() {
        for t in Self.moments {
            let p = UnitCurve.easeOut.value(at: t)
            let want = p <= 0.35 ? 0.16 : 0.16 * (1 - (p - 0.35) / 0.65)
            near(FreshTint.look(FreshTint.frozen(t)).tint, want, "at \(t)")
        }
        near(FreshTint.look(FreshTint.frozen(1, reduce: true)).tint, 0, "a colour, kept under Reduce Motion")
    }

    func testTheSuccessCheckIsBornSmallAndOvershootsOnTheSpring() {
        near(Born.look(Born.frozen(0), reduce: false).scale.width, 0.3, "born")
        XCTAssertGreaterThan(Born.look(Born.frozen(0.35), reduce: false).scale.width, 1, "the spring overshoots")
        XCTAssertEqual(Born.look(Born.frozen(1), reduce: false).scale.width, 1, accuracy: 0.001, "at rest")
    }

    func testAPingIsOneRingGrowingAndFadingAndNoneUnderReduceMotion() {
        for t in Self.moments {
            let p = Self.smooth(t)
            let look = Ping.look(Ping.frozen(t), reduce: false)
            near(look.scale.width, 1 + 1.6 * p, "ring at \(t)")
            near(look.opacity, 0.55 * (1 - p), "ring's opacity at \(t)")
            XCTAssertEqual(Ping.look(Ping.frozen(t, reduce: true), reduce: true).opacity, 0)
        }
    }

    func testTheSkeletonsShimmerCrossesOnceAndStandsStillUnderReduceMotion() {
        for t in Self.moments { near(Skeleton.band(Skeleton.frozen(t), reduce: false) ?? .nan, 1.2 * (1 - 2 * t), "band at \(t)") }
        XCTAssertNil(Skeleton.band(0.35, reduce: true))
    }

    func testStaleValuesAndTheEmptyGlyphRevealThemselves() {
        for t in Self.moments {
            near(Reveal.look(Reveal.frozen(t)).opacity, Self.smooth(t), "values at \(t)")
            let p = Self.smooth(t), d = Token.Motion.drawOnShare
            near(GlyphDraw.look(GlyphDraw.frozen(t), reduce: false).reveal, min(max((p - d) / (1 - d), 0), 1), "glyph at \(t)")
            XCTAssertEqual(GlyphDraw.look(t, reduce: true).reveal, 1, "shown whole under Reduce Motion")
        }
    }

    func testASheetUnrollsBelowItsHeaderWithADrop() {
        for t in Self.moments {
            let p = Self.smooth(t)
            let look = SheetReveal.look(SheetReveal.frozen(t), reduce: false)
            near(look.reveal, p, "reveal at \(t)")
            near(Double(look.y), -6 * (1 - p), "drop at \(t)")
        }
        let rect = CGRect(x: 0, y: 0, width: 300, height: 200)
        XCTAssertEqual(Unroll(share: 0, least: 46).path(in: rect).boundingRect.height, 46, "the header is there from the first frame")
        XCTAssertEqual(Unroll(share: 0.5, least: 46).path(in: rect).boundingRect.height, 123)
        near(SheetReveal.frozen(0.35, leaving: true), 1 - Self.snappy(0.35), "it leaves on snappy")
    }

    func testEachLayerAppearsFromItsOwnScale() {
        let kinds: [(PopAppear.Kind, Bool, Double)] = [(.popover, false, 0.94), (.popover, true, 0.97), (.menu, false, 0.97), (.alert, false, 1.08), (.alert, true, 0.96)]
        for (kind, leaving, from) in kinds {
            for t in Self.moments {
                let p = PopAppear.frozen(t, kind: kind, leaving: leaving)
                let look = PopAppear.look(p, kind: kind, leaving: leaving, reduce: false)
                near(look.scale.width, from + (1 - from) * p, "\(kind) leaving \(leaving) at \(t)")
                near(look.opacity, p, "\(kind) at \(t)")
            }
        }
        near(PopAppear.frozen(0.35, kind: .alert), Self.smooth(0.35), "an alert arrives on smooth")
        near(PopAppear.frozen(0.35, kind: .popover), Self.snappy(0.35), "a popover on snappy")
    }

    func testFindFadesInAndItsFieldSettles() {
        for t in Self.moments {
            let layer = FindAppear.look(FindAppear.frozen(t, field: false), field: false, reduce: false)
            let field = FindAppear.look(FindAppear.frozen(t, field: true), field: true, reduce: false)
            near(layer.opacity, Self.snappy(t), "layer at \(t)")
            XCTAssertEqual(layer.y, 0)
            near(Double(field.y), -6 * (1 - Self.smooth(t)), "field at \(t)")
        }
    }

    func testAHoverRevealsTheQuickActionSlidingIn() {
        for t in Self.moments {
            let p = Self.snappy(t)
            let look = HoverReveal.look(HoverReveal.frozen(t), reduce: false)
            near(look.opacity, p, "at \(t)")
            near(Double(look.x), 4 * (1 - p), "at \(t)")
        }
    }

    func testATilesPlusOneRisesIntoViewThenLiftsAway() {
        let rest = PlusOne.look(0, reduce: false), peak = PlusOne.look(0.25, reduce: false), gone = PlusOne.look(1, reduce: false)
        XCTAssertEqual(rest.opacity, 0)
        XCTAssertEqual(rest.y, 6)
        XCTAssertEqual(peak.opacity, 1)
        XCTAssertEqual(peak.y, 0)
        XCTAssertEqual(gone.opacity, 0)
        XCTAssertEqual(gone.y, -12)
        let p = Self.smooth(0.35), s = (p - 0.25) / 0.75
        near(PlusOne.look(PlusOne.frozen(0.35), reduce: false).opacity, 1 - s, "at a third")
    }

    func testAChartRegrowsFromItsBaseAndTheFooterRises() {
        for t in Self.moments {
            let p = Self.smooth(t)
            let chart = ChartRegrow.look(ChartRegrow.frozen(t), reduce: false)
            near(chart.scale.height, 0.92 + 0.08 * p, "chart at \(t)")
            XCTAssertEqual(chart.scale.width, 1)
            near(chart.opacity, 0.3 + 0.7 * p, "chart at \(t)")
            near(Double(FooterRise.look(FooterRise.frozen(t), reduce: false).y), 4 * (1 - p), "footer at \(t)")
        }
    }

    func testTheHUDOpensHoldsAndFoldsBack() {
        let shown = [(0.0, 0.0), (0.05, 0.5), (0.5, 1), (0.925, 0.5), (1, 0)]
        for (t, want) in shown {
            near(HUDReveal.look(t, reduce: false).reveal, want, "at \(t)")
            near(HUDReveal.look(t, reduce: false).opacity, want, "at \(t)")
        }
        XCTAssertEqual(HUDReveal.look(0.05, reduce: true).reveal, 1, "no mask under Reduce Motion")
    }

    func testAProgressFillEasesBetweenItsShares() {
        for t in Self.moments {
            near(ProgressEase.share(ProgressEase.frozen(t, reduce: false), from: 0.2, to: 0.8), 0.2 + 0.6 * Self.smooth(t), "at \(t)")
        }
    }

    func testUnderReduceMotionEveryMovementIsADissolve() {
        let t = 0.35
        let looks: [(String, Look)] = [
            ("panel", PanelAppear.look(t, reduce: true)),
            ("push", PushTransition.look(t, role: .top, way: .push, reduce: true)),
            ("segment", SegmentSlide.look(t, forward: true, reduce: true)),
            ("fold", Fold.look(t, reduce: true)),
            ("hero", HeroMorph.look(t, part: .top, reduce: true)),
            ("glyph", HeroMorph.look(t, part: .glyph, reduce: true)),
            ("leave", RowLeave.look(t, reduce: true)),
            ("land", RowLand.look(t, reduce: true)),
            ("born", Born.look(t, reduce: true)),
            ("sheet", SheetReveal.look(t, reduce: true)),
            ("alert", PopAppear.look(t, kind: .alert, reduce: true)),
            ("find", FindAppear.look(t, field: true, reduce: true)),
            ("hover", HoverReveal.look(t, reduce: true)),
            ("plus one", PlusOne.look(t, reduce: true)),
            ("chart", ChartRegrow.look(t, reduce: true)),
            ("footer", FooterRise.look(t, reduce: true)),
            ("hud", HUDReveal.look(t, reduce: true)),
        ]
        for (name, look) in looks {
            XCTAssertEqual(look.scale, CGSize(width: 1, height: 1), name)
            XCTAssertEqual(look.x, 0, name)
            XCTAssertEqual(look.y, 0, name)
            XCTAssertEqual(look.slide, 0, name)
            XCTAssertEqual(look.reveal, 1, name)
        }
        XCTAssertLessThan(Fold.look(t, reduce: true).height, 1, "a fold's height still eases")
        near(PanelAppear.frozen(t, closing: false, reduce: true), UnitCurve.easeInOut.value(at: t), "on the reduced ease")
    }
}
