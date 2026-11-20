import SwiftUI

extension Token.Metric {
    public enum Accounts {
        public static let plus: CGFloat = 12
        public static let footerGap: CGFloat = 8
        public static let linkGap: CGFloat = 3
    }
}

extension Token.Role {
    public static var addRow: Token.Role { Token.Role(13, .regular, .t1) }
    public static var addRowEnd: Token.Role { Token.Role(12, .regular, .t3) }
    public static var noneShared: Token.Role { Token.Role(12, .regular, .t2) }
    public static var accountsFooterLink: Token.Role { Token.Role(12, .regular, .t2) }
}
