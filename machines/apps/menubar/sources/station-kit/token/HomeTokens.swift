import SwiftUI

extension Token {
    // value: Home's header, read from the round-7 board: a 40 pt bar, the
    // title at 24 from the panel's edge, its bare 28 pt glyphs ending at
    // 18 from the other; under it the find field, 28 tall from 12 to 12,
    // its 1 pt accent rim inside a 3.5 pt soft ring.
    public enum HomeHeader {
        public static let plus = Symbol.plus
        public static let menu = "ellipsis"
        public static let menuLabel = "More options for station"
        public static let bar: CGFloat = 20
        public static let titleInset: CGFloat = 10
        public static let glyphsEnd: CGFloat = 4
        public static let button: CGFloat = 28
        public static let glyph: CGFloat = 15
        public static let glyphShare = 0.8
        public static let field: CGFloat = 10
        public static let fieldOut: CGFloat = -2
        public static let fieldHeight: CGFloat = 28
        public static let fieldGlyph: CGFloat = 13
        public static let fieldGap: CGFloat = 6
        public static let fieldPadding = EdgeInsets(top: 0, leading: 12, bottom: 0, trailing: 10)
        public static let rim: CGFloat = 1
        public static let ring: CGFloat = 3.5
        public static let ringShare = 0.30
    }

    // value: Home's sections as the board draws them: 12 between two
    // blocks, a head of 8 / 16 / 4 at 12 inside the box's edge, then its
    // box, rows from edge to edge; This Mac's three columns, 8 / 12 / 10
    // around them, a 0.5 pt line between two, each its name, its value and
    // its 72 by 16 spark.
    public enum HomeCard {
        public static let line: CGFloat = 32
        public static let blocks: CGFloat = 12
        public static let top: CGFloat = 2
        public static let head = EdgeInsets(top: 8, leading: 12, bottom: 4, trailing: 12)
        public static let headLine: CGFloat = 16
        public static let headSize: CGFloat = 13
        public static let rows = EdgeInsets(top: 0, leading: 4, bottom: 0, trailing: 4)
        public static let strip = EdgeInsets(top: 8, leading: 12, bottom: 10, trailing: 12)
        public static let cell: CGFloat = 4
        public static let cellAfter: CGFloat = 12
        public static let cellGap: CGFloat = 2
        public static let cellKey: CGFloat = 11
        public static let cellValue: CGFloat = 13
        public static let spark = CGSize(width: 72, height: 16)
        public static let sparkTop: CGFloat = 2
    }

    // value: a row of Home, the board's grammar: 40 tall with a sub-line,
    // 32 without; a 28 pt lead, a Mac's glyph on a 28 pt grey plate; the
    // name semibold, its Mac after it in the name's size; the hairline from
    // the text's edge; a verb a filled accent capsule; a value that waits
    // says it by a dot before it.
    public enum HomeRow {
        public static let height: CGFloat = 32
        public static let twoLines: CGFloat = 40
        public static let lead: CGFloat = 28
        public static let plate: CGFloat = 28
        public static let plateGlyph: CGFloat = 15
        public static let glyph: CGFloat = 16
        public static let glyphShare = 0.75
        public static let suffixGap: CGFloat = 5
        public static let inset: CGFloat = 48
        public static let sparkGap: CGFloat = 6
        public static let dot: CGFloat = 8
        public static let valueDot: CGFloat = 7
        public static let valueGap: CGFloat = 6
        public static let verb = EdgeInsets(top: 0, leading: 12, bottom: 0, trailing: 12)
        public static let verbHeight: CGFloat = 24
        public static let verbInk = Color.white
    }
}

private struct HomeRowsKey: EnvironmentKey {
    static let defaultValue = false
}

extension EnvironmentValues {
    // value: whether the rows drawn are Home's, in the board's grammar.
    public var stationHomeRows: Bool {
        get { self[HomeRowsKey.self] }
        set { self[HomeRowsKey.self] = newValue }
    }
}

extension Token.Role {
    public static let homeName = Token.Role(13, .semibold, .t1)
    public static let homeSuffix = Token.Role(13, .regular, .t2)
    public static let homeValue = Token.Role(13, .regular, .t2)
    public static let homeVerb = Token.Role(13, .medium, .t1)
}
