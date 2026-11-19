import SwiftUI

extension Token {
    // value: Home's header: its two round buttons, then the find field
    // under its title.
    public enum HomeHeader {
        public static let gap: CGFloat = 6
        public static let plus = Symbol.plus
        public static let menu = Symbol.gear
        public static let menuLabel = "More options for station"
        public static let field: CGFloat = 8
    }

    // value: a line of Home's cards that is no row: Add a Mac.
    public enum HomeCard {
        public static let line: CGFloat = 32
    }
}
