import SwiftUI

extension Token.Metric.Row {
    public static let suffixGap: CGFloat = 6
    public static let subGap: CGFloat = 1
    public static let chevron: CGFloat = 11
    public static let quickSlide: CGFloat = 4
    public static let dimShare = 0.55
    public static let staleShare = 0.45
    public static let verbTrail: CGFloat = 2
}

extension Token.Metric {
    public enum Need {
        public static let height: CGFloat = 32
        public static let padding: CGFloat = 6
        public static let mark: CGFloat = 14
    }

    public enum Port {
        public static let pointer: CGFloat = 20
        public static let pointerGlyph: CGFloat = 12
    }

    public enum Check {
        public static let height: CGFloat = 26
        public static let gap: CGFloat = 8
    }

    public enum Steps {
        public static let rowGap: CGFloat = 3
        public static let columnGap: CGFloat = 10
    }

    public enum Log {
        public static let lineGap: CGFloat = 2
        public static let moreTop: CGFloat = 4
        public static let moreGap: CGFloat = 3
    }

    public enum TileParts {
        public static let captionGap: CGFloat = 4
        public static let captionGlyph: CGFloat = 11
        public static let toggle: CGFloat = 18
        public static let toggleGlyph: CGFloat = 12
        public static let toggleMargin = EdgeInsets(top: -2, leading: -4, bottom: -2, trailing: 0)
        public static let plusGap: CGFloat = 4
        public static let unitGap: CGFloat = 3
        // value: the dot a tile's signal wears before its number, 8 pt,
        // its centre half the number's cap height over the baseline.
        public static let dot: CGFloat = 8
        public static let dotLift: CGFloat = 2.5
        public static let dotGap: CGFloat = 5
        // value: what a tile's number and word may shrink to so the dot
        // leaves "1 signed out" whole in a tile of the strip.
        public static let fit = 0.85
    }

    public enum GaugeParts {
        public static let bar: CGFloat = 6
        public static let hot = 0.85
        public static let full = 0.95
    }

    public enum Fold {
        public static let padding = EdgeInsets(top: 5, leading: 6, bottom: 5, trailing: 6)
        public static let gap: CGFloat = 4
        public static let chevron: CGFloat = 11
        public static let turned = Angle.degrees(180)
    }

    public enum EmptyParts {
        public static let symbol: CGFloat = 28
        public static let padding = EdgeInsets(top: 14, leading: 8, bottom: 10, trailing: 8)
        public static let boldTop: CGFloat = 6
        public static let greyTop: CGFloat = 2
        public static let buttonsTop: CGFloat = 10
        public static let buttonsGap: CGFloat = 6
    }
}

extension Token.Metric.HUD {
    public static let gap: CGFloat = 10
}

extension Token.Role {
    public static let needWhat = Token.Role(12, .regular, .t1)
    public static let needSubject = Token.Role(12, .semibold, .t1)
    public static let check = Token.Role(13, .regular, .t1)
    public static let checkResult = Token.Role(11, .regular, .t3)
    public static let stepAt = Token.Role(12, .regular, .t3)
    public static let step = Token.Role(12, .regular, .t2)
    public static let stepCurrent = Token.Role(12, .medium, .t1)
    public static let logLine = Token.Role(10.5, .regular, .t2, design: .monospaced, tracking: -0.105)
    public static let logSource = Token.Role(10.5, .medium, .t1, design: .monospaced, tracking: -0.105)
    public static let link = Token.Role(12, .regular, .t2)
    public static let fold = Token.Role(12, .regular, .t2)
    public static let emptyBold = Token.Role(13, .semibold, .t1)
    public static let emptyGrey = Token.Role(12, .regular, .t2)
    public static let tilePlus = Token.Role(11, .semibold, .t1)
}

extension Token.Symbol {
    public static let passed = "checkmark.circle.fill"
    public static let failed = "xmark.circle.fill"
    public static let outward = "arrow.up.right"
}

extension Token.Glyph {
    public static let outward = "↗"
    public static let percent = "%"
}
