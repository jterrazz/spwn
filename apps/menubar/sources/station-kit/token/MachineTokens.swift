import SwiftUI

extension Token.Metric {
    public enum PageHeader {
        public static let back: CGFloat = 13
        public static let backChevron: CGFloat = 11
        public static let backGap: CGFloat = 3
        public static let backBottom: CGFloat = 8
        public static let glyph: CGFloat = 20
        public static let glyphSlot: CGFloat = 24
        public static let titleGap: CGFloat = 8
        public static let title: CGFloat = 28
        public static let factsTop: CGFloat = 2
        public static let segmentsTop: CGFloat = 10
        public static let buttonsGap: CGFloat = 6
    }

    public enum CardLabel {
        public static let height: CGFloat = 16
        public static let inset: CGFloat = 4
    }

    public enum Care {
        public static let switchTrail: CGFloat = 2
    }

    public enum Birth {
        public static let bar: CGFloat = 2
        public static let inset = EdgeInsets(top: 0, leading: 40, bottom: 2, trailing: 16)
    }

    public enum Footnote {
        public static let padding = EdgeInsets(top: 2, leading: 4, bottom: 0, trailing: 4)
        public static let lineSpacing: CGFloat = 2
    }
}

extension Token.Role {
    public static let back = Token.Role(13, .regular, .t2)
    public static let footnote = Token.Role(12, .regular, .t3)
}

extension Token.Symbol {
    public static let menu = "ellipsis"
    public static let fold = "chevron.down"
}
