@testable import StationKit
import SwiftUI
import XCTest

final class ChartTests: XCTestCase {
    @MainActor
    func testEveryChartIsDrawnInBothSchemesAndNonePaintsThePlaceholder() throws {
        _ = Snapshot.render(AnyView(GlyphView(ChartSheet.agent)), scheme: .dark, desk: false)
        let failures = try ChartSheet.Family.allCases.flatMap { family in
            try [ColorScheme.dark, .light].compactMap { scheme in
                try Golden.check("kit/charts-\(family.rawValue)", AnyView(ChartSheet(family).stationAccessibility()), scheme: scheme, desk: false)
            }
        }
        XCTAssertEqual(failures, [])
    }

    func testAChartReachesTenPercentAboveItsPeak() {
        XCTAssertEqual(ChartScale.top([[12, 71, 40]], unit: "%"), 78.1, accuracy: 0.001)
    }

    func testAQuietPercentChartIsNeverScaledUnderTenPercent() {
        XCTAssertEqual(ChartScale.top([[2, 3, 4]], unit: "%"), 10)
        XCTAssertEqual(ChartScale.top([[]], unit: "%"), 10)
    }

    func testAQuietRateChartKeepsAFloorOfItsOwn() {
        XCTAssertEqual(ChartScale.top([[0.01, 0.02]], unit: "MB/s"), 0.1, accuracy: 0.0001)
    }

    func testBothSidesOfAMirroredChartShareOneScale() {
        XCTAssertEqual(ChartScale.top([[0.4, 0.5], [1.0, 0.2]], unit: "MB/s"), 1.1, accuracy: 0.0001)
    }

    func testTheFirstHighestPointIsThePeak() {
        XCTAssertEqual(ChartScale.peak([3, 9, 4, 9]), 1)
        XCTAssertNil(ChartScale.peak([]))
    }

    func testASparkIsScaledToItsOwnRangeNotToAHundred() {
        let low = ChartScale.spark([4, 5, 6])
        XCTAssertGreaterThan(low.lowerBound, 0)
        XCTAssertLessThan(low.upperBound, 10)
        XCTAssertTrue(low.contains(4) && low.contains(6))
    }

    func testAFlatSparkStillHasARangeAroundItsLevel() {
        let flat = ChartScale.spark([12, 12, 12])
        XCTAssertLessThan(flat.lowerBound, 12)
        XCTAssertGreaterThan(flat.upperBound, 12)
        XCTAssertEqual((flat.lowerBound + flat.upperBound) / 2, 12, accuracy: 0.0001)
    }

    func testAChipStaysInsideThePlot() {
        XCTAssertEqual(ChartScale.edge(2, width: 292), 44)
        XCTAssertEqual(ChartScale.edge(290, width: 292), 248)
        XCTAssertEqual(ChartScale.edge(150, width: 292), 150)
    }

    func testAScaleLabelSaysItsUnit() {
        XCTAssertEqual(ChartScale.say(45, unit: "%"), "45\u{00A0}%")
        XCTAssertEqual(ChartScale.say(0.55, unit: "MB/s"), "0.6\u{00A0}MB/s")
    }

    func testAMemoryBarTurnsOrangePast85AndRedPast95() {
        XCTAssertEqual(Bar.level(0.85), .calm)
        XCTAssertEqual(Bar.level(0.86), .warn)
        XCTAssertEqual(Bar.level(0.96), .critical)
        XCTAssertEqual(Bar(.capacity(os: 9, vms: [16, 8, 12, 12], total: 64)).used, 57.0 / 64, accuracy: 0.0001)
    }

    func testAScrubOutsideTheSeriesDrawsNoRule() {
        let chart = ChartSheet.chart(.line, [[1, 2, 3]], unit: "%", ends: ChartSheet.hourEnds)
        XCTAssertEqual(ActivityCharts(chart, scrub: 1).picked, 1)
        XCTAssertNil(ActivityCharts(chart, scrub: 3).picked)
    }

    func testAScrubbedPointSaysItsValueAndWhenItWas() {
        var chart = ChartSheet.chart(.bars, [[30, 42, 35]], unit: "%", ends: ChartSheet.hourEnds)
        chart.at = ["3 min ago", "2 min ago", "1 min ago"]
        let said = ActivityCharts(chart, scrub: 1).said(1)
        XCTAssertEqual(said.value, "42\u{00A0}%")
        XCTAssertEqual(said.when, "2 min ago")
    }

    func testAStorysScrubIsThePanelsScrub() {
        var ui = StoryUI()
        ui.scrub = 36
        let story = Story(file: StoryFile(id: "mac.mini.activity.scrub", group: .mac, note: "", pages: [], ui: ui), pages: [Page()])
        XCTAssertEqual(story.state.scrub, 36)
    }

    func testTopProcessesAreSortedByTheShareTheyShow() {
        let s = ChartSheet.processes
        XCTAssertEqual(TopProcesses.ranked(s) { $0 }.map(\.label), ["linux-3", "infra", "mds_stores", "review-bot"])
        let live = TopProcesses.ranked(s) { v in
            var v = v
            if v.n == 20 { v.n = 900 }
            return v
        }
        XCTAssertEqual(live.map(\.label), ["mds_stores", "review-bot", "linux-3", "infra"])
    }
}
