import SwiftUI

extension Token.Metric {
    public enum Accounts {
        public static let plus: CGFloat = 12
        public static let footerGap: CGFloat = 8
        public static let linkGap: CGFloat = 3
    }
}

extension Token.Role {
    public static let addRow = Token.Role(13, .regular, .t1)
    public static let addRowEnd = Token.Role(12, .regular, .t3)
    public static let noneShared = Token.Role(12, .regular, .t2)
    public static let footerLink = Token.Role(12, .regular, .t2)
}
