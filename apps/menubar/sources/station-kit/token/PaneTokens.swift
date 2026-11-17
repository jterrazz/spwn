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

        // value: the material's even inner edge, bright in light as Tahoe draws
        // it (refs/passwords-tahoe row 13), faint in dark; no top light.
        public static func rim(_ scheme: ColorScheme) -> Color { Color.white.opacity(scheme == .dark ? 0.09 : 0.55) }

        public struct Shade: Sendable {
            public let color: Color
            public let radius: CGFloat
            public let y: CGFloat
        }

        static let navy = hex(0x14141E)

        public static func ring(_ scheme: ColorScheme) -> Color { scheme == .dark ? Color.black.opacity(0.70) : Color.black.opacity(0.18) }

        public static func shades(_ scheme: ColorScheme) -> [Shade] {
            scheme == .dark
                ? [Shade(color: Color.black.opacity(0.45), radius: 30, y: 24), Shade(color: Color.black.opacity(0.30), radius: 9, y: 6)]
                : [Shade(color: navy.opacity(0.20), radius: 30, y: 24), Shade(color: navy.opacity(0.10), radius: 9, y: 6)]
        }
    }

    public enum Busy {
        public static let size: CGFloat = 9
        public static let stroke: CGFloat = 1.4
        public static let open = 0.12
        public static let start = Angle.degrees(-90 - 360 * open / 2)
        public static let fps = 10.0
        public static let period = 0.8
        public static let frames = Int((fps * period).rounded())

        public static func turn(frame: Int) -> Double { Double(frame % frames) / Double(frames) }
    }
}
