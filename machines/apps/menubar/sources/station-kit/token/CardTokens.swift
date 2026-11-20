import SwiftUI

extension Token.Metric {
    public enum CopyFooter {
        public static let gap: CGFloat = 5
        public static let glyph: CGFloat = 11
    }
}

extension Token.Role {
    public static var cardTitle: Token.Role { Token.Role(13, .semibold, .t1) }
    public static var copyCommand: Token.Role { Token.Role(12, .regular, .t2, design: .monospaced) }
}
