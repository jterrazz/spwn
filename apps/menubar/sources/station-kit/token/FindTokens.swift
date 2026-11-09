import SwiftUI

extension Token {
    // value: Find's parts beyond SPEC A1's Find metrics, read from the
    // prototype's `.find` rules: the field's glyph and padding, the
    // results' padding, a group's label, a result's padding, the footer
    // bar, the no-match lines; and the words Find says of itself.
    public enum Find {
        public static let recent = "Recent"
        public static let close = "Close"
        public static let show = "Show"
        public static let open = "Open"
        public static let wordBreaks = CharacterSet.whitespaces.union(CharacterSet(charactersIn: "\u{00B7}:-"))
        public static let hint = "Try a machine name, a port number, or \u{201C}agent\u{201D}."
        public static func noMatch(_ query: String) -> String { "No match for \(query)" }
        public static func more(_ n: Int) -> String { "Show \(n) more" }

        public static let fieldGlyph: CGFloat = 16
        public static let fieldPadding = EdgeInsets(top: 0, leading: 12, bottom: 0, trailing: 8)
        public static let fieldGap: CGFloat = 8
        public static let fieldRim: CGFloat = 1
        public static let results = EdgeInsets(top: 4, leading: 12, bottom: 4, trailing: 12)
        public static let group = EdgeInsets(top: 8, leading: 8, bottom: 3, trailing: 8)
        public static let groupLine: CGFloat = 13
        public static let result = EdgeInsets(top: 0, leading: 8, bottom: 0, trailing: 10)
        public static let gap: CGFloat = 8
        public static let scopeGap: CGFloat = 8
        public static let glyphSlot: CGFloat = 24
        public static let moreGlyph: CGFloat = 11
        public static let address: CGFloat = 12
        public static let scope: CGFloat = 12
        public static let footerTop: CGFloat = 4
        public static let footerPadding: CGFloat = 14
        public static let footerGap: CGFloat = 6
        public static let footerText: CGFloat = 11
        public static let divider = CGSize(width: 1, height: 12)
        public static let dividerMargin: CGFloat = 4
        public static let empty = EdgeInsets(top: 14, leading: 20, bottom: 12, trailing: 20)
        public static let emptySpacing: CGFloat = 4.5
        public static let onTintSoft = 0.85
        public static let moreSymbol = Symbol.down
        public static let closeKey = "esc"
        public static let runKey = "enter"
        public static let altKey = "cmd+enter"
    }
}
