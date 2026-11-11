import SwiftUI

extension Token {
    public enum Object {
        public static func mac(lid: Bool) -> String { lid ? "laptopcomputer" : "macmini" }
        public static let server = "server.rack"
        public static let workspace = "shippingbox"
        public static let agent = "station.agent"
        public static let unknown = "questionmark"
        public static let account = "key"
        public static let port = "web"

        public static let accounts = ["github", "claude", "npm", "tailscale", "cloudflare", "docker"]
        public static let ports = ["vite", "astro", "postgres", "traefik", "kubernetes", "node", "storybook", "api", "web"]

        public static func mark(_ brand: String) -> String { "station.mark.\(brand)" }

        public static func brand(_ id: String, among known: [String]) -> String? {
            let key = id.lowercased()
            return known.first { key.contains($0) }
        }
    }

    public enum Hot {
        public static let cpu = 0.75
        public static let leftSeconds: Int64 = 1800
    }
}

extension Token.Metric.Marks {
    public static let badgeDot: CGFloat = 8
    public static let badgeRing: CGFloat = 2
    public static let badgeProgress: CGFloat = 11
    public static let badgeDrop: CGFloat = 3
    public static let dashStroke: CGFloat = 1.2
    public static let dash: [CGFloat] = [1.6, 1.25]
    public static let squareRadius: CGFloat = 1.6
    public static let pingScale: CGFloat = 2.6
    public static let pingShare = 0.55
    public static let macGlyph: CGFloat = 13
    public static let asleepDash: [CGFloat] = [2.2, 2.6]
    public static let ringMargin: CGFloat = -2
    public static let spinnerSmallStroke: CGFloat = 1.4
    public static let spinnerArc = 0.25
    public static let spinnerPeriod = 0.8
    public static let spinnerStill = Angle.degrees(-45)
    public static let doneShare = 0.6
    public static let chipEdge: CGFloat = 0.5
    public static let chipEdgeShare = 0.12
    public static let progressLeast = 0.02
    public static let logoKey: CGFloat = 10
    public static let perMille: Double = 1000
    public static let percent: Double = 100
}

extension Token.Metric.Control {
    public static let pillPadding: CGFloat = 7
    public static let pillGap: CGFloat = 5
    public static let pillDot: CGFloat = 6
    public static let pillMax: CGFloat = 190
    public static let pillDash: CGFloat = 1
    public static let pillDashes: [CGFloat] = [3, 2]
    public static let keyCapTracking: CGFloat = 0.22
    public static let buttonGap: CGFloat = 5
    public static let buttonGlyph: CGFloat = 13
    public static let verbPadding: CGFloat = 10
    public static let roundGlyph: CGFloat = 13
    public static let roundSmallGlyph: CGFloat = 11
    public static let roundTopBarGlyph: CGFloat = 14
    public static let iconGlyph: CGFloat = 15
    public static let softShare = 0.15
    public static let disabledShare = 0.4
    public static let keyOnTintShare = 0.2
    public static let segmentedSize: CGFloat = 12
    public static let timeLeft: CGFloat = 86
    public static let timeLeftBar: CGFloat = 3
    public static let timeLeftGap: CGFloat = 4
    public static let knobInset: CGFloat = 2
}

extension Token.Motion {
    public static let ping = Animation.timingCurve(smoothCurve, duration: 0.6)
    public static let progress = Animation.timingCurve(smoothCurve, duration: 0.34)
}

extension Token.Resolved {
    public func signal(_ signal: String?) -> Color {
        switch signal {
        case "active": color.accent
        case "waits": color.orange
        case "failing": color.red
        case "well": color.green
        case "faint", "asleep", "stale": color.t3
        default: color.t2
        }
    }

    public func signalText(_ signal: String?) -> Color {
        switch signal {
        case "active": color.accentText
        case "waits": color.orangeText
        case "failing": color.redText
        case "well": color.greenText
        case "faint", "asleep", "stale": color.t3
        default: color.t2
        }
    }

    public var switchOff: Color { scheme == .dark ? color.qq : Token.rgba(0, 0, 0, 0.12) }
    public var thumb: Color { scheme == .dark ? Token.rgba(255, 255, 255, 0.17) : Token.hex(0xFFFFFF) }
    public var knob: Color { Token.hex(0xFFFFFF) }
    public var onTint: Color { Token.hex(0xFFFFFF) }
    public var chipEdge: Color { Token.rgba(0, 0, 0, Token.Metric.Marks.chipEdgeShare) }
    public var contrastRim: Color { Token.rgba(0, 0, 0, Token.Metric.Control.contrastRimShare) }
}

extension Token.Shadow {
    public static let switchKnob = Token.Shadow(color: Token.rgba(0, 0, 0, 0.3), radius: 1, y: 1)
    public static let segmentThumb = Token.Shadow(color: Token.rgba(0, 0, 0, 0.14), radius: 1, y: 1)
}

extension Token.Key {
    static let modifiers: [(String, String)] = [("cmd", "⌘"), ("opt", "⌥"), ("shift", "⇧"), ("ctrl", "⌃")]
    static let named: [String: String] = ["enter": "↩", "esc": "esc", "up": "↑", "down": "↓", "left": "←", "right": "→", "space": "space", "tab": "⇥", "delete": "⌫"]

    public static func cap(_ key: String) -> String {
        key.split(separator: "+").map { part in
            let p = String(part)
            return modifiers.first { $0.0 == p }?.1 ?? named[p] ?? p.uppercased()
        }.joined()
    }
}

extension Token {
    // value: how a failing mark is drawn: its symbol's solid variant.
    public enum Solid {
        public static let variant = SymbolVariants.fill
    }

    public enum Live {
        public static let tenths: Double = 10
        public static let giga: Double = 1_000_000_000
    }
}
