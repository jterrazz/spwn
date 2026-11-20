import SwiftUI

extension Token.Metric {
    public enum Ports {
        public static let subhead = EdgeInsets(top: 6, leading: 40, bottom: 2, trailing: 8)
        public static let empty = EdgeInsets(top: 6, leading: 8, bottom: 6, trailing: 8)
        public static let exposedGap: CGFloat = 8
        public static let exposedButtons: CGFloat = 28
        public static let exposedButtonsTop: CGFloat = 8
        public static let exposedButtonsGap: CGFloat = 6
        public static let footerGap: CGFloat = 4
    }
}

extension Token.Role {
    public static var portsSubhead: Token.Role { Token.Role(11, .regular, .t3) }
    public static var exposedTitle: Token.Role { Token.Role(13, .semibold, .t1) }
    public static var footerLink: Token.Role { Token.Role(13, .regular, .t2) }
}

extension Token.Symbol {
    public static let lock = "lock"
}
