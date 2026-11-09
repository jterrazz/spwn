import SwiftUI

extension Token.Metric {
    public enum Wait {
        public static let gap: CGFloat = 6
        public static let headGap: CGFloat = 6
        public static let buttonsGap: CGFloat = 8
        public static let buttonsTop: CGFloat = 4
        public static let prGap: CGFloat = 5
        public static let bodySpacing: CGFloat = 2
    }

    public enum Reply {
        public static let height: CGFloat = 26
        public static let radius: CGFloat = 13
        public static let padding: CGFloat = 10
        public static let gap: CGFloat = 6
    }

    public enum NowCard {
        public static let gap: CGFloat = 8
        public static let headGap: CGFloat = 6
    }

    public enum CopyFooter {
        public static let gap: CGFloat = 5
        public static let glyph: CGFloat = 11
    }
}

extension Token.Role {
    public static let cardTitle = Token.Role(13, .semibold, .t1)
    public static let cardTask = Token.Role(13, .regular, .t2)
    public static let waitSince = Token.Role(12, .regular, .t3)
    public static let waitTitle = Token.Role(13, .semibold, .t1)
    public static let waitBody = Token.Role(12, .regular, .t2)
    public static let waitPR = Token.Role(12, .regular, .t2)
    public static let reply = Token.Role(12, .regular, .t1)
    public static let replyHint = Token.Role(12, .regular, .t3)
    public static let copyCommand = Token.Role(12, .regular, .t2, design: .monospaced)
}

extension Token.Symbol {
    public static let waits = "text.bubble"
    public static let pullRequest = "arrow.triangle.pull"
    public static let send = "arrow.up"
}

extension Token.Glyph {
    public static let minus = "\u{2212}"
}
