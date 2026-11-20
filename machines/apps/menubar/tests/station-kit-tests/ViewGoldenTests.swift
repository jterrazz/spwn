import StationKit
import SwiftUI
import XCTest

final class ViewGoldenTests: XCTestCase {
    func line(_ mark: ViewMark, _ x: CGFloat, _ y: CGFloat, _ w: CGFloat, _ h: CGFloat, _ children: [ViewLine] = []) -> ViewLine {
        ViewLine(mark: mark, frame: CGRect(x: x, y: y, width: w, height: h), children: children)
    }

    var base: [ViewLine] {
        [line(ViewMark(.group, kit: "Panel"), 24, 24, 322, 200, [
            line(ViewMark(.listitem, "api", kit: "RowView"), 36, 60, 298, 34),
            line(ViewMark(.listitem, "web", kit: "RowView"), 36, 100, 298, 34),
            line(ViewMark(.text, "Find", kit: "FootNote"), 36, 180, 100, 16),
        ])]
    }

    func testALineSaysItsStatesAndHowManyLinesItsWordsWrapTo() {
        let lines = [line(ViewMark(.group, kit: "Card"), 0, 0, 200, 60, [
            line(ViewMark(.text, "two lines", kit: "SheetLine", pitch: 16), 10, 10, 120, 32),
            line(ViewMark(.text, "one line", kit: "SheetLine", pitch: 16), 10, 42, 120, 16),
            line(ViewMark(.switch, kit: "Switch", states: ViewState.of([.on: true, .busy: false, .lit: true])), 150, 10, 30, 18),
        ])]
        XCTAssertEqual(ViewGolden.text(lines, canvas: ["scheme: dark"]), """
        canvas: { scheme: dark }
        tree:
        - group [kit=Card] [at=0,0] [size=200x60]:
          - text "two lines" [kit=SheetLine] [at=10,10] [size=120x32] [lines=2]
          - text "one line" [kit=SheetLine] [at=10,42] [size=120x16]
          - switch [kit=Switch] [at=150,10] [size=30x18] [state=on] [state=lit]

        """)
    }

    func testAnOverlayIsWrittenAsWhatItOwnsOverItsBase() {
        let over = [line(ViewMark(.group, kit: "Panel"), 24, 24, 322, 200, [
            line(ViewMark(.listitem, "api", kit: "RowView", states: [.lit]), 36, 60, 298, 34),
            line(ViewMark(.listitem, "web", kit: "RowView"), 36, 100, 298, 34),
        ]), line(ViewMark(.menu, kit: "MenuPreview"), 60, 90, 200, 80, [
            line(ViewMark(.menuitem, "Open", kit: "MenuLine"), 65, 95, 190, 24),
        ])]
        XCTAssertEqual(ViewGolden.text(over, canvas: ["scheme: dark"], base: "mac.this.overview.dark", over: base), """
        canvas: { scheme: dark }
        base: ./mac.this.overview.dark.view.yaml
        changes:
        - in group [kit=Panel]:
          - listitem "api" [kit=RowView] [at=12,36] [size=298x34] [state=lit]
        added:
        - in root:
          - menu [kit=MenuPreview] [at=60,90] [size=200x80]:
            - menuitem "Open" [kit=MenuLine] [at=5,5] [size=190x24]
        removed:
        - in group [kit=Panel]:
          - text "Find" [kit=FootNote]

        """)
    }

    func testAnOverlayThatDrawsItsBaseOwnsNothing() {
        XCTAssertEqual(ViewGolden.text(base, canvas: ["scheme: dark", "contrast: increased"], base: "menubar.down.dark", over: base),
                       "canvas: { scheme: dark, contrast: increased }\nbase: ./menubar.down.dark.view.yaml\n")
    }

    func testSiblingsOfOneKeyArePathedByTheirRank() {
        let rows = { (lit: Bool) in
            [self.line(ViewMark(.list, kit: "Rows"), 0, 0, 100, 40, [
                self.line(ViewMark(.listitem, kit: "RowView"), 0, 0, 100, 20),
                self.line(ViewMark(.listitem, kit: "RowView"), 0, 20, 100, 20, [
                    self.line(ViewMark(.img, kit: "Chevron", states: lit ? [.expanded] : []), 90, 25, 8, 8),
                ]),
            ])]
        }
        XCTAssertEqual(ViewGolden.delta(base: ViewGolden.nodes(rows(false)), over: ViewGolden.nodes(rows(true))).changes, [
            "- in list [kit=Rows] › listitem [kit=RowView] #2:",
            "  - img [kit=Chevron] [at=90,5] [size=8x8] [state=expanded]",
        ])
    }
}
