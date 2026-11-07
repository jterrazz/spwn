import AppKit
@testable import StationKit
import SwiftUI
import XCTest

final class FrameSink: @unchecked Sendable {
    var frames: [RowPart: CGRect] = [:]
}

final class MoleculeTests: XCTestCase {
    typealias Sheet = MoleculeSheet

    @MainActor
    func testEveryMoleculeIsDrawnInBothSchemesAndNonePaintsThePlaceholder() throws {
        let failures = try Sheet.Family.allCases.flatMap { family in
            try [ColorScheme.dark, .light].compactMap { scheme in
                try Golden.check("kit/molecules-\(family.rawValue)", AnyView(Sheet(family).stationAccessibility()), scheme: scheme, desk: false)
            }
        }
        XCTAssertEqual(failures, [])
    }

    @MainActor
    static func frames(_ row: Row, highlighted: Bool) -> [RowPart: CGRect] {
        let sink = FrameSink()
        let content = RowView(row, columns: Sheet.columns, actions: Sheet.actions, highlighted: highlighted)
            .frame(width: Token.Metric.Panel.width - Token.Metric.Panel.body.leading - Token.Metric.Panel.body.trailing)
            .onPreferenceChange(RowFrames.self) { sink.frames = $0 }
            .environment(\.stationSamples, Sheet.samples)
            .environment(\.stationStill, true)
        let renderer = ImageRenderer(content: content)
        _ = renderer.cgImage
        return sink.frames
    }

    @MainActor
    func testNothingInARowMovesUnderThePointer() {
        for row in Sheet.agents + Sheet.accessories {
            let rest = Self.frames(row, highlighted: false)
            let lit = Self.frames(row, highlighted: true)
            XCTAssertFalse(rest.isEmpty, row.id ?? "")
            XCTAssertEqual(rest, lit, "\(row.id ?? ""): a part moved when the row lit")
        }
    }

    @MainActor
    func testTheQuickSlotIsReservedBeforeTheTrailingValue() throws {
        let mac = try XCTUnwrap(Sheet.accessories.first { $0.id == "This Mac" })
        let frames = Self.frames(mac, highlighted: false)
        let quick = try XCTUnwrap(frames[.quick])
        let trail = try XCTUnwrap(frames[.trail])
        XCTAssertGreaterThan(quick.width, 0)
        XCTAssertLessThanOrEqual(quick.maxX, trail.minX)
    }

    @MainActor
    func testEveryRowOfThePanelHasOneLeftEdge() {
        let edges = Set((Sheet.agents + Sheet.accessories).map { Self.frames($0, highlighted: false)[.name]?.minX ?? -1 })
        XCTAssertEqual(edges, [Token.Metric.Row.padding + Token.Metric.Row.accessory + Token.Metric.Row.gap])
    }

    @MainActor
    func testARowWithAVerbHasNoChevronAndARowThatPushesHasOne() throws {
        let verb = try XCTUnwrap(Sheet.agents.last)
        let nav = try XCTUnwrap(Sheet.agents.first)
        XCTAssertGreaterThan(Self.frames(verb, highlighted: false)[.control]?.width ?? 0, Token.Metric.Control.verbMinWidth - 1)
        XCTAssertLessThan(Self.frames(nav, highlighted: false)[.control]?.width ?? .infinity, Token.Metric.Row.accessory)
    }

    @MainActor
    func testAFinishedAgentThatAlsoPushesShowsItsVerbAndNoChevron() throws {
        var finished = try XCTUnwrap(Sheet.agents.last)
        XCTAssertEqual(finished.verb, "delete")
        finished.nav = "menubar.machine:agent-7f"
        let alone = Self.frames(finished, highlighted: false)
        finished.nav = nil
        XCTAssertEqual(alone[.control], Self.frames(finished, highlighted: false)[.control])
    }

    func testOnlyATintedRowHidesTheHairlinesAroundIt() throws {
        let rows = Sheet.accessories
        let mac = try XCTUnwrap(rows.firstIndex { $0.id == "This Mac" })
        let vite = try XCTUnwrap(rows.firstIndex { $0.id == "Vite" })
        XCTAssertNotNil(rows[mac].nav)
        XCTAssertNil(rows[vite].nav)
        let shown = { (hovered: Int?, highlighted: String?) in
            rows.indices.filter { RowStack.hairline($0, of: rows, hovered: hovered, highlighted: highlighted) }
        }
        let all = Array(1 ..< rows.count)
        XCTAssertEqual(shown(nil, nil), all)
        XCTAssertEqual(shown(mac, nil), all.filter { $0 != mac && $0 != mac + 1 })
        XCTAssertEqual(shown(nil, "This Mac"), shown(mac, nil))
        XCTAssertEqual(shown(vite, nil), all)
    }

    func testTheAccessorySetsWhereTheHairlineStarts() {
        let insets = Sheet.accessories.map { RowView.inset(RowView.accessory($0)) }
        let m = Token.Metric.Row.self
        XCTAssertEqual(insets, [m.inset, m.inset, m.insetRing, m.insetRing, m.insetRing, m.insetGlyph, m.insetGlyph])
    }

    func testTheTrailingValueIsTheCellOfTheColumnTheRowNames() {
        var row = Row()
        row.cells = [Sheet.percent(370)]
        row.trail = "value"
        XCTAssertEqual(RowView.trail(row, columns: Sheet.columns), Sheet.percent(370))
        row.trail = "name"
        XCTAssertNil(RowView.trail(row, columns: Sheet.columns))
        row.trail = "cpu"
        XCTAssertNil(RowView.trail(row, columns: Sheet.columns))
    }

    func testAFigureIsSplitFromItsUnitAtTheNoBreakSpace() {
        XCTAssertEqual(Figure.split(Format.show(Sheet.free(23))).figure, "23")
        XCTAssertEqual(Figure.split(Format.show(Sheet.free(23))).unit, "GB free")
        XCTAssertEqual(Figure.split(Format.show(Sheet.percent(370))).unit, "")
        XCTAssertEqual(Figure.split(Format.show(Sheet.left(14400, of: 14400))).figure, "4")
    }

    func testALogLineIsSaidByItsTimeAndItsSource() {
        let p = LogLines.parts("14:30:59 kernel Out of memory: killed k3s-server")
        XCTAssertEqual([p.at, p.source, p.said], ["14:30:59", "kernel", "Out of memory: killed k3s-server"])
        XCTAssertEqual(LogLines.parts("rebooting").said, "rebooting")
    }

    func testACheckIsPassedRunningNextOrFailed() {
        func check(_ signal: String?) -> CheckRow.State {
            var r = Row()
            r.signal = signal
            return CheckRow.state(r)
        }
        XCTAssertEqual([check("well"), check("active"), check(nil), check("failing")], [.passed, .running, .next, .failed])
    }
}
