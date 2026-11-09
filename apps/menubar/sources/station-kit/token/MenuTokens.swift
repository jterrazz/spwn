import SwiftUI

extension Token {
    // value: SPEC A5's menu preview beside Token.Metric.Menu: its header,
    // a separator's band, the gaps of an entry, the ✓ slot, its ring and
    // its shadow (0 12 32 at 50 %), where a menu opens from its anchor.
    public enum MenuPreview {
        public static let header: CGFloat = 20
        public static let line: CGFloat = 11
        public static let ring: CGFloat = 0.5
        public static let gap: CGFloat = 8
        public static let side: CGFloat = 8
        public static let check: CGFloat = 12
        public static let checkGlyph: CGFloat = 11
        public static let chevron: CGFloat = 10
        public static let trailingRoom: CGFloat = 16
        public static let shadow = Shadow(color: Color.black.opacity(0.5), radius: 16, y: 12)
        public static let litInk = Color.white
        public static let inset: CGFloat = 4
        public static let pointer: CGFloat = 12
        public static let overlap: CGFloat = 4
    }

    // value: the glyphs a menu says a key by.
    public enum MenuKey {
        public static let command = "⌘"
        public static let shift = "⇧"
        public static let option = "⌥"
        public static let control = "⌃"
        public static let enter = "↩"
    }
}

extension Token.Symbol {
    public static let checkmark = "checkmark"
    public static let submenu = "chevron.right"
}
