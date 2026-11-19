import SwiftUI

extension Token.Metric.Popover {
    public static let arrow: CGFloat = 12
    public static let arrowDepth: CGFloat = 6
    public static let margin: CGFloat = 8
    public static let gap: CGFloat = 4
    public static let qrRadius: CGFloat = 10
    public static let qrInset: CGFloat = 8
    public static let column: CGFloat = 12
    public static let lines: CGFloat = 2
    public static let buttons: CGFloat = 8
    public static let stopTop: CGFloat = 2
    public static let paragraph: CGFloat = 6
}

extension Token.Metric.Tooltip {
    public static let gap: CGFloat = 4
    public static let lead: CGFloat = 12
    public static let delay: Double = 0.7
}

extension Token.Metric.HUD {
    public static let above: CGFloat = 8
}

extension Token.Symbol {
    public static let hudCopy = "doc.on.doc"
    public static let hudTerminal = "terminal"
    public static let hudOpen = "safari"
    public static let hudDone = "checkmark.circle.fill"
}

extension Token.Role {
    public static let popoverTitle = Token.Role(13, .semibold, .t1)
    public static let popoverLink = Token.Role(12, .regular, .t2, design: .monospaced)
    public static let popoverDomain = Token.Role(12, .regular, .t2)
    public static let popoverReach = Token.Role(11, .regular, .t3)
    public static let popoverBody = Token.Role(12, .regular, .t1)
}
