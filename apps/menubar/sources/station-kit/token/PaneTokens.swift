import SwiftUI

extension Token {
    public enum Pane {
        public static let rimWidth = Metric.Panel.rim
        public static let solidEdge = Metric.Row.hairline
        public static let outline: CGFloat = 0.5
        public static let outlineRadius = Glass.window + outline
        public static let fade = Metric.Panel.fade
        public static let opaque = Color.black
        public static let depth = 3

        public static func rim(_ scheme: ColorScheme) -> [Gradient.Stop] {
            let white = Color.white
            return scheme == .dark
                ? [.init(color: white.opacity(0.28), location: 0), .init(color: white.opacity(0.06), location: 0.22),
                   .init(color: white.opacity(0), location: 0.60), .init(color: white.opacity(0.05), location: 1)]
                : [.init(color: white.opacity(0.95), location: 0), .init(color: white.opacity(0.35), location: 0.22),
                   .init(color: white.opacity(0), location: 0.60), .init(color: white.opacity(0), location: 1)]
        }

        public struct Shade: Sendable {
            public let color: Color
            public let radius: CGFloat
            public let y: CGFloat
        }

        static let navy = hex(0x14141E)

        public static func ring(_ scheme: ColorScheme) -> Color { scheme == .dark ? Color.black.opacity(0.70) : Color.black.opacity(0.14) }

        public static func shades(_ scheme: ColorScheme) -> [Shade] {
            scheme == .dark
                ? [Shade(color: Color.black.opacity(0.45), radius: 30, y: 24), Shade(color: Color.black.opacity(0.30), radius: 9, y: 6)]
                : [Shade(color: navy.opacity(0.20), radius: 30, y: 24), Shade(color: navy.opacity(0.10), radius: 9, y: 6)]
        }
    }
}
