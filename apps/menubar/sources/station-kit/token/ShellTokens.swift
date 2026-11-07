import SwiftUI

extension Token {
    public enum Item {
        public static let countGap: CGFloat = 5
        public static let pressedRadius: CGFloat = 5
        public static let pressedPad = CGSize(width: 7, height: 3)
        public static let strip: [String] = ["wifi", "battery.75percent", "switch.2"]
        public static let stripGap: CGFloat = 18

        public static func red(_ scheme: ColorScheme) -> Color { Token.hex(scheme == .dark ? 0xFF453A : 0xE0281D) }
        public static func pressed(_ scheme: ColorScheme) -> Color { scheme == .dark ? Color.white.opacity(0.22) : Color.black.opacity(0.10) }
        public static func bar(_ scheme: ColorScheme) -> Color { scheme == .dark ? Color.black.opacity(0.32) : Color.white.opacity(0.42) }
    }
}
