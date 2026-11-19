import SwiftUI

extension Token.Metric {
    public enum CopyFooter {
        public static let gap: CGFloat = 5
        public static let glyph: CGFloat = 11
    }
}

extension Token.Role {
    public static let cardTitle = Token.Role(13, .semibold, .t1)
    public static let copyCommand = Token.Role(12, .regular, .t2, design: .monospaced)
}
