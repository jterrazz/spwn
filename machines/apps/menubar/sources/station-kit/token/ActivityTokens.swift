import SwiftUI

extension Token.Metric {
    public enum Activity {
        public static let picker: CGFloat = 150
        public static let cardGap: CGFloat = 6
        public static let legendGap: CGFloat = 12
        public static let legendDot: CGFloat = 6
        public static let legendDotGap: CGFloat = 4
    }
}

extension Token.Role {
    public static var chartTitle: Token.Role { Token.Role(13, .semibold, .t1) }
    public static var chartValue: Token.Role { Token.Role(13, .regular, .t2) }
    public static var chartNote: Token.Role { Token.Role(11, .medium, .t2) }
    public static var legend: Token.Role { Token.Role(11, .regular, .t2) }
}

extension Token.Glyph {
    public static let space = " "
}
