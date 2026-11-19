import SwiftUI

extension Token {
    // value: the hero's parts beyond SPEC A1's Hero metrics, read from the
    // prototype's `.hero` rules: the busy spinner in the glyph slot, the
    // facts under the title, the rows' hairline band.
    public enum HeroParts {
        public static let spinner: CGFloat = 22
        public static let spinnerStroke: CGFloat = 2.4
        public static let titleLines = 2
        public static let factsTop: CGFloat = 2
        public static let titleGap: CGFloat = 7
        public static let rowsTop: CGFloat = 6
        public static let rowsBleed: CGFloat = 6
        public static let rowsBottom: CGFloat = 5
        public static let rowsPad: CGFloat = 2
        public static let alsoMark = "exclamationmark.circle.fill"
        public static let more = Symbol.forward
    }

    // value: Home's header: its title and its two round buttons.
    public enum HomeHeader {
        public static let title = "All machines"
        public static let gap: CGFloat = 6
        public static let plus = Symbol.plus
        public static let menu = Symbol.gear
        public static let menuLabel = "More options for station"
    }

    // value: the Find capsule of Home's footer.
    public enum FindCapsule {
        public static let words = "Find a machine, a port, an action\u{2026}"
        public static let key = "cmd+k"
        public static let gap: CGFloat = 7
        public static let glyph: CGFloat = 12
        public static let padding: CGFloat = 11
    }

    // value: a line of Home's cards that is no row: a fold, Add a Mac.
    public enum HomeCard {
        public static let line: CGFloat = 32
    }
}
