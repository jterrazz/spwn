import SwiftUI

extension Token.Metric.Alert {
    public static let iconBelow: CGFloat = 8
    public static let iconRadius: CGFloat = 8
    public static let iconMark: CGFloat = 17
    public static let bodyAbove: CGFloat = 4
    public static let lossesAbove: CGFloat = 8
    public static let lossesInset: CGFloat = 4
    public static let lossGap: CGFloat = 2
    public static let fieldAbove: CGFloat = 10
    public static let labelBelow: CGFloat = 5
    public static let labelGap: CGFloat = 8
    public static let field: CGFloat = 24
    public static let fieldRadius: CGFloat = 6
    public static let fieldPadding: CGFloat = 8
    public static let fieldSize: CGFloat = 13
    public static let focusRing: CGFloat = 3
    public static let focusShare = 0.4
    public static let caret: CGFloat = 1.5
    public static let caretHeight: CGFloat = 14
    public static let buttonGap: CGFloat = 8
    public static let edge: CGFloat = 0.5
    public static let shadowRadius: CGFloat = 25
    public static let shadowY: CGFloat = 20
    public static let shadowShare = 0.45
    public static let lineSpacing: CGFloat = 2
}

extension Token.Role {
    public static let alertLabel = Token.Role(12, .medium, .t2)
    public static let alertName = Token.Role(12, .bold, .t1, design: .monospaced)
}

extension Token {
    // value: the app's icon an alert leads with, as an NSAlert shows it:
    // the rack on its blue squircle.
    public enum AppIcon {
        public static let top = hex(0x4A8CFF)
        public static let bottom = hex(0x1A4FD6)
        public static let mark = Color.white
        public static let shadow = Color.black.opacity(0.25)
        public static let shadowRadius: CGFloat = 1
        public static let shadowY: CGFloat = 1
    }

    public static let alertShadow = Color.black
}

extension Token.Glyph {
    public static let bullet = "•"
}
