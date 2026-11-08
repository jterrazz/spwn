import SwiftUI

extension Token.Metric {
    public enum Health {
        public static let headGap: CGFloat = 10
        public static let headLines: CGFloat = 1
        public static let mark: CGFloat = 22
        public static let checksIndent: CGFloat = 32
        public static let checksTop: CGFloat = 4
        public static let buttonsTop: CGFloat = 8
        public static let buttonsGap: CGFloat = 12
        public static let banner: CGFloat = 52
        public static let lit: CGFloat = 4
    }

    public enum Service {
        public static let subTop: CGFloat = 2
        public static let chartTop: CGFloat = 12
        public static let chart: CGFloat = 36
        public static let endsTop: CGFloat = 6
    }

    public enum Uptime {
        public static let cellsTop: CGFloat = 10
        public static let endsTop: CGFloat = 6
    }

    public enum SSHFooter {
        public static let copyGap: CGFloat = 4
        public static let copyGlyph: CGFloat = 11
        public static let disabledShare = 0.45
    }
}

extension Token.Role {
    public static let healthTitle = Token.Role(13, .medium, .t1)
    public static let healthSub = Token.Role(11, .regular, .t2)
    public static let bannerTitle = Token.Role(15, .semibold, .t1)
    public static let serviceTitle = Token.Role(13, .semibold, .t1)
    public static let serviceValue = Token.Role(13, .regular, .t2)
    public static let serviceSub = Token.Role(11, .regular, .t2)
    public static let footerCopy = Token.Role(12, .regular, .t2, design: .monospaced)
    public static let footerNote = Token.Role(12, .regular, .t3)
    public static let toolingNote = Token.Role(11, .regular, .t3)
}
